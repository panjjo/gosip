package sipapi

import (
	"errors"
	"fmt"
	"sync"
	"time"

	sdp "github.com/panjjo/gosdp"
	"github.com/panjjo/gosip/db"
	"github.com/panjjo/gosip/m"
	sip "github.com/panjjo/gosip/sip/s"
	"github.com/panjjo/gosip/utils"
	"github.com/sirupsen/logrus"
)

// sip 请求播放
func SipPlay(data *Streams) (*Streams, error) {

	channel := Channels{ChannelID: data.ChannelID}
	if err := db.Get(db.DBClient, &channel); err != nil {
		if db.RecordNotFound(err) {
			return nil, errors.New("通道不存在")
		}
		return nil, err
	}

	data.DeviceID = channel.DeviceID
	data.StreamType = channel.StreamType
	// 使用通道的播放模式进行处理
	switch channel.StreamType {
	case m.StreamTypePull:
		// 拉流

	default:
		// 推流模式要求设备在线且活跃
		if time.Now().Unix()-channel.Active > 30*60 || channel.Status != m.DeviceStatusON {
			return nil, errors.New("通道已离线")
		}
		user, ok := _activeDevices.Get(channel.DeviceID)
		if !ok {
			return nil, errors.New("设备已离线")
		}
		// GB28181推流
		if data.StreamID == "" {
			ssrcLock.Lock()
			data.ssrc = getSSRC(data.T)
			data.StreamID = ssrc2stream(data.ssrc)

			// 成功后保存
			db.Create(db.DBClient, data)
			ssrcLock.Unlock()
		}

		var err error
		data, err = sipPlayPush(data, channel, user)
		if err != nil {
			return nil, fmt.Errorf("获取视频失败:%v", err)
		}
	}

	data.HTTP = fmt.Sprintf("%s/rtp/%s/hls.m3u8", config.Media.HTTP, data.StreamID)
	data.RTMP = fmt.Sprintf("%s/rtp/%s", config.Media.RTMP, data.StreamID)
	data.RTSP = fmt.Sprintf("%s/rtp/%s", config.Media.RTSP, data.StreamID)
	data.WSFLV = fmt.Sprintf("%s/rtp/%s.live.flv", config.Media.WS, data.StreamID)

	data.Ext = time.Now().Unix() + 2*60 // 2分钟等待时间
	StreamList.Response.Store(data.StreamID, data)
	if data.T == 0 {
		StreamList.Succ.Store(data.ChannelID, data)
	}
	db.Save(db.DBClient, data)
	return data, nil
}

var ssrcLock *sync.Mutex

func sipPlayPush(data *Streams, channel Channels, device Devices) (*Streams, error) {
	var (
		s sdp.Session
		b []byte
	)
	name := "Play"
	protocal := "TCP/RTP/AVP"
	if data.T == 1 {
		name = "Playback"
		protocal = "RTP/RTCP"
	}

	video := sdp.Media{
		Description: sdp.MediaDescription{
			Type:     "video",
			Port:     _sysinfo.MediaServerRtpPort,
			Formats:  []string{"96", "98", "97"},
			Protocol: protocal,
		},
	}
	video.AddAttribute("recvonly")
	if data.T == 0 {
		video.AddAttribute("setup", "passive")
		video.AddAttribute("connection", "new")
	}
	video.AddAttribute("rtpmap", "96", "PS/90000")
	video.AddAttribute("rtpmap", "98", "H264/90000")
	video.AddAttribute("rtpmap", "97", "MPEG4/90000")

	// defining message
	msg := &sdp.Message{
		Origin: sdp.Origin{
			Username: _serverDevices.DeviceID, // 媒体服务器id
			Address:  _sysinfo.MediaServerRtpIP.String(),
		},
		Name: name,
		Connection: sdp.ConnectionData{
			IP:  _sysinfo.MediaServerRtpIP,
			TTL: 0,
		},
		Timing: []sdp.Timing{
			{
				Start: data.S,
				End:   data.E,
			},
		},
		Medias: []sdp.Media{video},
		SSRC:   data.ssrc,
	}
	if data.T == 1 {
		msg.URI = fmt.Sprintf("%s:0", channel.ChannelID)
	}

	// appending message to session
	s = msg.Append(s)
	// appending session to byte buffer
	b = s.AppendTo(b)
	uri, _ := sip.ParseURI(channel.URIStr)
	channel.addr = &sip.Address{URI: uri}
	_serverDevices.addr.Params.Add("tag", sip.String{Str: utils.RandString(20)})
	hb := sip.NewHeaderBuilder().SetTo(channel.addr).SetFrom(_serverDevices.addr).AddVia(&sip.ViaHop{
		Params: sip.NewParams().Add("branch", sip.String{Str: sip.GenerateBranch()}),
	}).SetContentType(&sip.ContentTypeSDP).SetMethod(sip.INVITE).SetContact(_serverDevices.addr)
	req := sip.NewRequest("", sip.INVITE, channel.addr.URI, sip.DefaultSipVersion, hb.Build(), b)
	req.SetDestination(device.source)
	req.AppendHeader(&sip.GenericHeader{HeaderName: "Subject", Contents: fmt.Sprintf("%s:%s,%s:%s", channel.ChannelID, data.StreamID, _serverDevices.DeviceID, data.StreamID)})
	req.SetRecipient(channel.addr.URI)
	tx, err := srv.Request(req)
	if err != nil {
		logrus.Warningln("sipPlayPush fail.id:", device.DeviceID, channel.ChannelID, "err:", err)
		return data, err
	}
	// response
	response, err := sipResponse(tx)
	if err != nil {
		logrus.Warningln("sipPlayPush response fail.id:", device.DeviceID, channel.ChannelID, "err:", err)
		return data, err
	}
	data.Resp = response
	// ACK
	tx.Request(sip.NewRequestFromResponse(sip.ACK, response))

	callid, _ := response.CallID()
	data.CallID = string(*callid)

	cseq, _ := response.CSeq()
	if cseq != nil {
		data.CseqNo = cseq.SeqNo
	}

	from, _ := response.From()
	to, _ := response.To()
	for k, v := range to.Params.Items() {
		data.Ttag[k] = v.String()
	}
	for k, v := range from.Params.Items() {
		data.Ftag[k] = v.String()
	}
	data.Status = 0

	return data, err
}

// sip 停止播放
func SipStopPlay(ssrc string) {
	zlmCloseStream(ssrc)
	data, ok := StreamList.Response.Load(ssrc)
	if !ok {
		return
	}
	play := data.(*Streams)
	if play.StreamType == m.StreamTypePush {
		// 推流，需要发送关闭请求
		resp := play.Resp
		u, ok := _activeDevices.Load(play.DeviceID)
		if !ok {
			return
		}
		user := u.(Devices)
		req := sip.NewRequestFromResponse(sip.BYE, resp)
		req.SetDestination(user.source)
		tx, err := srv.Request(req)
		if err != nil {
			logrus.Warningln("sipStopPlay bye fail.id:", play.DeviceID, play.ChannelID, "err:", err)
		}
		_, err = sipResponse(tx)
		if err != nil {
			logrus.Warnln("sipStopPlay response fail", err)
			play.Msg = err.Error()
		} else {
			play.Status = 1
			play.Stop = true
		}
		db.Save(db.DBClient, play)
	}
	StreamList.Response.Delete(ssrc)
	if play.T == 0 {
		StreamList.Succ.Delete(play.ChannelID)
	}
}
