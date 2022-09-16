package sipapi

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/panjjo/gosip/db"
	sip "github.com/panjjo/gosip/sip/s"
	"github.com/sirupsen/logrus"
)

// Streams Streams
type Streams struct {
	db.DBModel
	// 0  直播 1 历史
	T          int
	SSRC       string
	DeviceID   string
	ChannelID  string
	StreamType string //  pull 媒体服务器主动拉流，push 监控设备主动推流
	Status     int    // 0正常 1关闭 -1 尚未开始
	Ftag       db.M   `gorm:"column:ftag" sql:"type:json"` // header from params
	Ttag       db.M   `gorm:"column:ttag" sql:"type:json"` // header to params
	CallID     string // header callid
	Time       string
	Stop       bool
	Msg        string
	CseqNo     uint32 `gorm:"column:cseqno"`
}

// 当前系统中存在的流列表
type streamsList struct {
	// key=ssrc value=PlayParams  播放对应的PlayParams 用来发送bye获取tag，callid等数据
	Response *sync.Map
	// key=channelid value={Play}  当前设备直播信息，防止重复直播
	Succ *sync.Map
	ssrc int
}

var StreamList streamsList

func getSSRC(t int) string {
	r := false
	for {
		StreamList.ssrc++
		key := fmt.Sprintf("%d%s%04d", t, _sysinfo.Region[3:8], StreamList.ssrc)
		stream := Streams{SSRC: ssrc2stream(key), Stop: false}
		if err := db.Get(db.DBClient, &stream); db.RecordNotFound(err) || stream.CreatedAt == 0 {
			return key
		}
		if StreamList.ssrc > 9000 && !r {
			StreamList.ssrc = 0
			r = true
		}
	}
}

// 定时检查未关闭的流
// 检查规则：
// 1. 数据库查询当前status=0在推流状态的所有流信息
// 2. 比对当前streamlist中存在的流，如果不在streamlist或者ssrc与channelid不匹配则关闭
func CheckStreams() {
	logrus.Debugln("checkStreamWithCron")
	var skip int64
	for {
		streams := []Streams{}
		db.FindT(db.DBClient, new(Streams), &streams, db.M{"status=?": 0}, "", skip, 100, false)
		for _, stream := range streams {
			logrus.Debugln("checkStreamStreamID", stream.SSRC, stream.DeviceID)
			if p, ok := StreamList.Response.Load(stream.SSRC); ok {
				playParams := p.(PlayParams)
				if stream.ChannelID == playParams.ChannelID {
					// 此流在用
					// 查询media流是否仍然存在。不存在的需要关闭。
					rtpInfo := zlmGetMediaInfo(playParams.SSRC)
					if rtpInfo.Exist {
						// 流仍然存在
						continue
					}
					if !playParams.Stream && time.Now().Unix() < playParams.Ext {
						// 推流尚未成功 未超时
						continue
					}
				}
			}
			logrus.Debugln("checkStreamActiveDevice", stream.SSRC, stream.DeviceID)
			device, ok := _activeDevices.Get(stream.DeviceID)
			if !ok {
				continue
			}
			if device.source == nil {
				logrus.Warningln("checkStreamDeviceSource is nil", stream.SSRC, stream.DeviceID)
				continue
			}
			logrus.Debugln("checkStreamClosed", stream.SSRC, stream.DeviceID)
			// 关闭此流
			channel := Channels{ChannelID: stream.ChannelID}
			if err := db.Get(db.DBClient, &channel); err != nil {
				logrus.Errorln("checkStreamGetchannelError", stream.SSRC, stream.ChannelID, err)
				stream.Msg = err.Error()
				db.Save(db.DBClient, stream)
				channel = Channels{
					ChannelID: stream.ChannelID,
					DeviceID:  stream.DeviceID,
					URIStr:    fmt.Sprintf("sip:%s@%s", stream.ChannelID, _serverDevices.Region),
				}
			}
			channelURI, _ := sip.ParseURI(channel.URIStr)
			channel.addr = &sip.Address{URI: channelURI, Params: sip.NewParams()}
			for k, v := range stream.Ttag {
				channel.addr.Params.Add(k, sip.String{Str: v.(string)})
			}
			for k, v := range stream.Ftag {
				_serverDevices.addr.Params.Add(k, sip.String{Str: v.(string)})
			}
			callid := sip.CallID(stream.CallID)

			hb := sip.NewHeaderBuilder().SetToWithParam(channel.addr).SetFrom(_serverDevices.addr).AddVia(&sip.ViaHop{
				Params: sip.NewParams().Add("branch", sip.String{Str: sip.GenerateBranch()}),
			}).SetContentType(&sip.ContentTypeSDP).SetMethod(sip.BYE).SetContact(_serverDevices.addr).SetCallID(&callid).SetSeqNo(uint(stream.CseqNo))
			req := sip.NewRequest("", sip.BYE, channel.addr.URI, sip.DefaultSipVersion, hb.Build(), nil)
			req.SetDestination(device.source)
			req.SetRecipient(channel.addr.URI)

			// 不管成功不成功 程序都删除掉，后面开新流，关闭不成功的后面重试
			StreamList.Response.Delete(stream.SSRC)
			StreamList.Succ.Delete(stream.ChannelID)

			tx, err := srv.Request(req)
			if err != nil {
				logrus.Warningln("checkStreamClosedFail", stream.SSRC, err)
				stream.Msg = err.Error()
				db.Save(db.DBClient, stream)
				continue
			}
			response := tx.GetResponse()
			if response == nil {
				logrus.Warningln("checkStreamClosedFail response is nil", channel.ChannelID, channel.DeviceID, stream.SSRC)
				continue
			}
			if response.StatusCode() != http.StatusOK {
				if response.StatusCode() == 481 {
					logrus.Infoln("checkStreamClosedFail1", stream.SSRC, response.StatusCode())
					stream.Msg = response.Reason()
					stream.Status = 1
				} else {
					logrus.Warningln("checkStreamClosedFail1", stream.SSRC, response.StatusCode())
					stream.Msg = response.Reason()
				}
			} else {
				stream.Status = 1
				stream.Stop = true
			}
			db.Save(db.DBClient, stream)

		}
		if len(streams) != 100 {
			break
		}
		skip += 100
	}
}
