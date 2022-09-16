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

// PlayParams 播放请求参数
type PlayParams struct {
	// 0  直播 1 历史
	T int
	//  开始结束时间，只有t=1 时有效
	S, E       time.Time
	SSRC       string
	Resp       *sip.Response
	DeviceID   string
	ChannelID  string
	Url        string
	Ext        int64  // 推流等待的过期时间，用于判断是否请求成功但推流失败。超过还未接收到推流定义为失败，重新请求推流或者关闭此ssrc
	Stream     bool   // 是否完成推流，用于web_hook 出现stream=false时等待推流，出现stream_not_found 且 stream=true表示推流过但已关闭。释放ssrc。
	StreamType string //  pull 媒体服务器主动拉流，push 监控设备主动推流 proxy 代理
}

type Play struct {
	// 通道ID
	ChannelID string `json:"channelid"`
	// 视频流ID
	StreamID string `json:"streamid"`
	// 流标示
	SSRC string `json:"ssrc"`
	// m3u8播放地址
	HTTP string `json:"http"`
	// rtmp 播放地址
	RTMP string `json:"rtmp"`
	// rtsp 播放地址
	RTSP string `json:"rtsp"`
	// flv 播放地址
	WSFLV string `json:"ws-flv"`
}

// sip 请求播放
func SipPlay(data PlayParams) (*Play, error) {
	var succ *Play
	switch data.StreamType {
	case m.StreamTypeProxy:
		_, err := zlmAddStreamProxy(data.Url, data.ChannelID)
		if err != nil {
			return nil, err
		}
		succ = &Play{
			ChannelID: data.ChannelID,
			StreamID:  data.SSRC,
			SSRC:      data.SSRC,
			HTTP:      fmt.Sprintf("%s/rtp/%s/hls.m3u8", config.Media.HTTP, data.SSRC),
			RTMP:      fmt.Sprintf("%s/rtp/%s", config.Media.RTMP, data.SSRC),
			RTSP:      fmt.Sprintf("%s/rtp/%s", config.Media.RTSP, data.SSRC),
			WSFLV:     fmt.Sprintf("%s/rtp/%s.live.flv", config.Media.WS, data.SSRC),
		}
	default:
		channels := Channels{ChannelID: data.ChannelID}
		if err := db.Get(db.DBClient, &channels); err != nil {
			if db.RecordNotFound(err) {
				return nil, errors.New("通道不存在")
			}
			return nil, err
		}

		if time.Now().Unix()-channels.Active > 30*60 {
			return nil, errors.New("通道已离线")
		}
		device := Devices{DeviceID: channels.DeviceID}
		if err := db.Get(db.DBClient, &device); err != nil {
			if db.RecordNotFound(err) {
				return nil, errors.New("设备不存在")
			}
			return nil, err
		}

		user, ok := _activeDevices.Get(device.DeviceID)
		if !ok {
			return nil, errors.New("设备已离线")
		}
		data.DeviceID = user.DeviceID
		var err error
		data, err = sipPlayPush(data, channels, user)
		if err != nil {
			return nil, fmt.Errorf("获取视频失败:%v", err)
		}
		succ = &Play{
			ChannelID: data.ChannelID,
			SSRC:      data.SSRC,
			StreamID:  data.SSRC,
			HTTP:      fmt.Sprintf("%s/rtp/%s/hls.m3u8", config.Media.HTTP, data.SSRC),
			RTMP:      fmt.Sprintf("%s/rtp/%s", config.Media.RTMP, data.SSRC),
			RTSP:      fmt.Sprintf("%s/rtp/%s", config.Media.RTSP, data.SSRC),
			WSFLV:     fmt.Sprintf("%s/rtp/%s.live.flv", config.Media.WS, data.SSRC),
		}
	}

	data.Ext = time.Now().Unix() + 2*60 // 2分钟等待时间
	StreamList.Response.Store(data.SSRC, data)
	if data.T == 0 {
		StreamList.Succ.Store(data.ChannelID, succ)
	}
	return succ, nil
}

var ssrcLock *sync.Mutex

func sipPlayPush(data PlayParams, channel Channels, device Devices) (PlayParams, error) {
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
	if data.SSRC == "" {
		ssrcLock.Lock()
		data.SSRC = getSSRC(data.T)
		// 成功后保存mongo，用来后续系统关闭推流使用
		db.Create(db.DBClient, &Streams{
			T:          data.T,
			SSRC:       ssrc2stream(data.SSRC),
			ChannelID:  channel.ChannelID,
			DeviceID:   channel.DeviceID,
			StreamType: m.StreamTypePush, //  pull 媒体服务器主动拉流，push 监控设备主动推流
			Status:     -1,
			Time:       time.Now().Format("2006-01-02 15:04:05"),
		})
		ssrcLock.Unlock()
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
		SSRC:   data.SSRC,
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
	req.AppendHeader(&sip.GenericHeader{HeaderName: "Subject", Contents: fmt.Sprintf("%s:%s,%s:%s", channel.ChannelID, data.SSRC, _serverDevices.DeviceID, data.SSRC)})
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
	data.SSRC = ssrc2stream(data.SSRC)
	data.StreamType = m.StreamTypePush
	from, _ := response.From()
	to, _ := response.To()
	callid, _ := response.CallID()
	var cseqNo uint32
	cseq, _ := response.CSeq()
	if cseq != nil {
		cseqNo = cseq.SeqNo
	}
	toParams := db.M{}
	for k, v := range to.Params.Items() {
		toParams[k] = v.String()
	}
	fromParams := db.M{}
	for k, v := range from.Params.Items() {
		fromParams[k] = v.String()
	}
	db.UpdateAll(db.DBClient, new(Streams), db.M{"ssrc=?": data.SSRC, "stop=?": false}, db.M{"call_id": string(*callid), "ttag": toParams, "cseqno": cseqNo, "ftag": fromParams, "status": 0})
	return data, err
}

// sip 停止播放
func SipStopPlay(ssrc string) {
	data, ok := StreamList.Response.Load(ssrc)
	if !ok {
		return
	}
	play := data.(PlayParams)
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
			db.UpdateAll(db.DBClient, new(Streams), db.M{"ssrc=?": play.SSRC, "stop=?": false}, db.M{"err": err})
		} else {
			db.UpdateAll(db.DBClient, new(Streams), db.M{"ssrc=?": play.SSRC, "stop=?": false}, db.M{"status": 1, "stop": true})
		}
	}
	StreamList.Response.Delete(ssrc)
	if play.T == 0 {
		StreamList.Succ.Delete(play.ChannelID)
	}
	zlmCloseStream(ssrc)
}
