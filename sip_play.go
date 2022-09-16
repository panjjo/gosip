package main

import (
	"fmt"
	"sync"
	"time"

	sdp "github.com/panjjo/gosdp"
	"github.com/panjjo/gosip/sip"
	"github.com/panjjo/gosip/utils"
	"github.com/sirupsen/logrus"
)

// playParams 播放请求参数
type playParams struct {
	// 0  直播 1 历史
	T int
	//  开始结束时间，只有t=1 时有效
	S, E       time.Time
	SSRC       string
	Resp       *sip.Response
	DeviceID   string
	UserID     string
	Url        string
	ext        int64  // 推流等待的过期时间，用于判断是否请求成功但推流失败。超过还未接收到推流定义为失败，重新请求推流或者关闭此ssrc
	stream     bool   // 是否完成推流，用于web_hook 出现stream=false时等待推流，出现stream_not_found 且 stream=true表示推流过但已关闭。释放ssrc。
	streamType string //  pull 媒体服务器主动拉流，push 监控设备主动推流 proxy 代理

}

// sip 请求播放
func sipPlay(data playParams) interface{} {
	var succ map[string]interface{}
	switch data.streamType {
	case streamTypeProxy:
		_, err := zlmAddStreamProxy(data.Url, data.DeviceID)
		if err != nil {
			return err
		}
		succ = map[string]interface{}{
			"deviceid": data.DeviceID,
			"ssrc":     data.SSRC,
			"http":     fmt.Sprintf("%s/rtp/%s/hls.m3u8", config.Media.HTTP, data.SSRC),
			"rtmp":     fmt.Sprintf("%s/rtp/%s", config.Media.RTMP, data.SSRC),
			"rtsp":     fmt.Sprintf("%s/rtp/%s", config.Media.RTSP, data.SSRC),
			"ws-flv":   fmt.Sprintf("%s/rtp/%s.live.flv", config.Media.WS, data.SSRC),
		}
	default:
		device := Devices{}
		if err := dbClient.Get(deviceTB, M{"deviceid": data.DeviceID}, &device); err != nil {
			if err == ErrRecordNouFound {
				return "监控设备不存在"
			}
			return err
		}
		if time.Now().Unix()-device.Active > 30*60 {
			return "监控设备已离线"
		}
		userT := NVRDevices{}
		if err := dbClient.Get(userTB, M{"deviceid": device.PDID}, &userT); err != nil {
			if err == ErrRecordNouFound {
				return "用户设备不存在"
			}
			return err
		}
		user, ok := _activeDevices.Get(userT.DeviceID)
		if !ok {
			return "用户设备已离线"
		}
		data.UserID = user.DeviceID
		var err error
		data, err = sipPlayPush(data, device, user)
		if err != nil {
			return fmt.Sprintf("获取视频失败:%v", err)
		}
		succ = map[string]interface{}{
			"deviceid": user.DeviceID,
			"ssrc":     data.SSRC,
			"http":     fmt.Sprintf("%s/rtp/%s/hls.m3u8", config.Media.HTTP, data.SSRC),
			"rtmp":     fmt.Sprintf("%s/rtp/%s", config.Media.RTMP, data.SSRC),
			"rtsp":     fmt.Sprintf("%s/rtp/%s", config.Media.RTSP, data.SSRC),
			"ws-flv":   fmt.Sprintf("%s/rtp/%s.live.flv", config.Media.WS, data.SSRC),
		}
		data.UserID = user.DeviceID
	}

	data.ext = time.Now().Unix() + 2*60 // 2分钟等待时间
	_playList.ssrcResponse.Store(data.SSRC, data)
	if data.T == 0 {
		_playList.devicesSucc.Store(data.DeviceID, succ)
	}
	return succ
}

var ssrcLock *sync.Mutex

func sipPlayPush(data playParams, device Devices, user NVRDevices) (playParams, error) {
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
		dbClient.Insert(streamTB, DeviceStream{
			T:          data.T,
			SSRC:       ssrc2stream(data.SSRC),
			DeviceID:   data.DeviceID,
			UserID:     data.UserID,
			StreamType: streamTypePush, //  pull 媒体服务器主动拉流，push 监控设备主动推流
			Status:     -1,
			Time:       time.Now().Format("2006-01-02 15:04:05"),
		})
		ssrcLock.Unlock()
	}
	video := sdp.Media{
		Description: sdp.MediaDescription{
			Type:     "video",
			Port:     _sysinfo.mediaServerRtpPort,
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
	m := &sdp.Message{
		Origin: sdp.Origin{
			Username: _serverDevices.DeviceID, // 媒体服务器id
			Address:  _sysinfo.mediaServerRtpIP.String(),
		},
		Name: name,
		Connection: sdp.ConnectionData{
			IP:  _sysinfo.mediaServerRtpIP,
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
		m.URI = fmt.Sprintf("%s:0", data.DeviceID)
	}

	// appending message to session
	s = m.Append(s)
	// appending session to byte buffer
	b = s.AppendTo(b)
	deviceURI, _ := sip.ParseURI(device.URIStr)
	device.addr = &sip.Address{URI: deviceURI}
	_serverDevices.addr.Params.Add("tag", sip.String{Str: utils.RandString(20)})
	hb := sip.NewHeaderBuilder().SetTo(device.addr).SetFrom(_serverDevices.addr).AddVia(&sip.ViaHop{
		Params: sip.NewParams().Add("branch", sip.String{Str: sip.GenerateBranch()}),
	}).SetContentType(&sip.ContentTypeSDP).SetMethod(sip.INVITE).SetContact(_serverDevices.addr)
	req := sip.NewRequest("", sip.INVITE, user.addr.URI, sip.DefaultSipVersion, hb.Build(), b)
	req.SetDestination(user.source)
	req.AppendHeader(&sip.GenericHeader{HeaderName: "Subject", Contents: fmt.Sprintf("%s:%s,%s:%s", device.DeviceID, data.SSRC, _serverDevices.DeviceID, data.SSRC)})
	req.SetRecipient(device.addr.URI)
	tx, err := srv.Request(req)
	if err != nil {
		logrus.Warningln("sipPlayPush fail.id:", device.DeviceID, "err:", err)
		return data, err
	}
	// response
	response, err := sipResponse(tx)
	if err != nil {
		logrus.Warningln("sipPlayPush response fail.id:", device.DeviceID, "err:", err)
		return data, err
	}
	data.Resp = response
	// ACK
	tx.Request(sip.NewRequestFromResponse(sip.ACK, response))
	data.SSRC = ssrc2stream(data.SSRC)
	data.streamType = streamTypePush
	from, _ := response.From()
	to, _ := response.To()
	callid, _ := response.CallID()
	toParams := map[string]string{}
	for k, v := range to.Params.Items() {
		toParams[k] = v.String()
	}
	var cseqNo uint32
	cseq, _ := response.CSeq()
	if cseq != nil {
		cseqNo = cseq.SeqNo
	}
	fromParams := map[string]string{}
	for k, v := range from.Params.Items() {
		fromParams[k] = v.String()
	}
	dbClient.Update(streamTB, M{"ssrc": data.SSRC, "stop": false}, M{"$set": M{"callid": callid, "ttag": toParams, "ftag": fromParams, "status": 0, "seqno": cseqNo}})
	return data, err
}

// sip 停止播放
func sipStopPlay(ssrc string) {
	data, ok := _playList.ssrcResponse.Load(ssrc)
	if !ok {
		return
	}
	play := data.(playParams)
	if play.streamType == streamTypePush {
		// 推流，需要发送关闭请求
		resp := play.Resp
		u, ok := _activeDevices.Load(play.UserID)
		if !ok {
			return
		}
		user := u.(NVRDevices)
		req := sip.NewRequestFromResponse(sip.BYE, resp)
		req.SetDestination(user.source)
		tx, err := srv.Request(req)
		if err != nil {
			logrus.Warningln("sipStopPlay bye fail.id:", play.DeviceID, "err:", err)
		}
		_, err = sipResponse(tx)
		if err != nil {
			logrus.Warnln("sipStopPlay response fail", err)
			dbClient.Update(streamTB, M{"ssrc": play.SSRC, "stop": false}, M{"$set": M{"err": err}})
		} else {
			dbClient.Update(streamTB, M{"ssrc": play.SSRC, "stop": false}, M{"$set": M{"status": 1, "stop": true}})
		}
	}
	_playList.ssrcResponse.Delete(ssrc)
	if play.T == 0 {
		_playList.devicesSucc.Delete(play.DeviceID)
	}
	zlmCloseStream(ssrc)
}
