package api

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/panjjo/gosip/db"
	"github.com/panjjo/gosip/m"
	sipapi "github.com/panjjo/gosip/sip"
	"github.com/panjjo/gosip/utils"
	"github.com/sirupsen/logrus"
)

func ZLMWebHook(c *gin.Context) {
	method := c.Param("method")
	switch method {
	case "on_server_started":
		// zlm 启动，具体业务自行实现
		m.MConfig.GB28181.MediaServer = true
		c.JSON(http.StatusOK, map[string]any{
			"code": 0,
			"msg":  "success"})
	case "on_http_access":
		// http请求鉴权，具体业务自行实现
		c.JSON(http.StatusOK, map[string]any{
			"code":   0,
			"second": 86400})
	case "on_play":
		//视频播放触发鉴权
		c.JSON(http.StatusOK, map[string]any{
			"code": 0,
			"msg":  "",
		})
	case "on_publish":
		// 推流鉴权
		c.JSON(http.StatusOK, map[string]any{
			"code":       0,
			"enableHls":  m.MConfig.Stream.HLS,
			"enableMP4":  false,
			"enableRtxp": m.MConfig.Stream.RTMP,
			"msg":        "success",
		})
	case "on_stream_none_reader":
		// 无人阅读通知 关闭流
		zlmStreamNoneReader(c)
	case "on_stream_not_found":
		// 请求播放时，流不存在时触发
		zlmStreamNotFound(c)
	case "on_record_mp4":
		//  mp4 录制完成
		zlmRecordMp4(c)
	case "on_stream_changed":
		// 流注册和注销通知
		zlmStreamChanged(c)
	default:
		c.JSON(http.StatusOK, map[string]any{
			"code": -1,
			"msg":  "body error",
		})
	}

}

type ZLMStreamChangedData struct {
	Regist bool   `json:"regist"`
	APP    string `json:"app"`
	Stream string `json:"stream"`
	Schema string `json:"schema"`
}

func zlmStreamChanged(c *gin.Context) {
	body := c.Request.Body
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		c.JSON(http.StatusOK, map[string]any{
			"code": -1,
			"msg":  "body error",
		})
		return
	}
	req := &ZLMStreamChangedData{}
	if err := utils.JSONDecode(data, &req); err != nil {
		c.JSON(http.StatusOK, map[string]any{
			"code": -1,
			"msg":  "body error",
		})
		return
	}
	ssrc := req.Stream
	if req.Regist {
		if req.Schema == "rtmp" {
			d, ok := sipapi.StreamList.Response.Load(ssrc)
			if ok {
				// 接收到流注册事件，更新ssrc数据
				params := d.(*sipapi.Streams)
				params.Stream = true
				db.Save(db.DBClient, params)
				sipapi.StreamList.Response.Store(ssrc, params)
				// 接收到流注册后进行视频流编码分析，分析出此设备对应的编码格式并保存或更新
				sipapi.SyncDevicesCodec(ssrc, params.DeviceID)
			} else {
				// ssrc不存在，关闭流
				sipapi.SipStopPlay(ssrc)
				logrus.Infoln("closeStream on_stream_changed notfound!", req.Stream)
			}
		}
	} else {
		if req.Schema == "hls" {
			//接收到流注销事件
			_, ok := sipapi.StreamList.Response.Load(ssrc)
			if ok {
				// 流还存在，注销
				sipapi.SipStopPlay(ssrc)
				logrus.Infoln("closeStream on_stream_changed cancel!", req.Stream)
			}
		}
	}
	c.JSON(http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success"})
}

type ZLMRecordMp4Data struct {
	APP       string `json:"app"`
	Stream    string `json:"stream"`
	FileName  string `json:"file_name"`
	FilePath  string `json:"file_path"`
	FileSize  int    `json:"file_size"`
	Folder    string `json:"folder"`
	StartTime int64  `json:"start_time"`
	TimeLen   int    `json:"time_len"`
	URL       string `json:"url"`
}

func zlmRecordMp4(c *gin.Context) {
	body := c.Request.Body
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		c.JSON(http.StatusOK, map[string]any{
			"code": -1,
			"msg":  "body error",
		})
		return
	}
	req := &ZLMRecordMp4Data{}
	if err := utils.JSONDecode(data, &req); err != nil {
		c.JSON(http.StatusOK, map[string]any{
			"code": -1,
			"msg":  "body error",
		})
		return
	}
	if item, ok := sipapi.RecordList.Get(req.Stream); ok {
		sipapi.RecordList.Stop(req.Stream)
		item.Down(req.URL)
		item.Resp(fmt.Sprintf("%s/%s", m.MConfig.Media.HTTP, req.URL))
	}
	c.JSON(http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success"})
}

type ZLMStreamNotFoundData struct {
	APP    string `json:"app"`
	Params string `json:"params"`
	Stream string `json:"stream"`
	Schema string `json:"schema"`
	ID     string `json:"id"`
	IP     string `json:"ip"`
	Port   int    `json:"port"`
}

func zlmStreamNotFound(c *gin.Context) {
	body := c.Request.Body
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		c.JSON(http.StatusOK, map[string]any{
			"code": -1,
			"msg":  "body error",
		})
		return
	}
	req := &ZLMStreamNotFoundData{}
	if err := utils.JSONDecode(data, &req); err != nil {
		c.JSON(http.StatusOK, map[string]any{
			"code": -1,
			"msg":  "body error",
		})
		return
	}
	ssrc := req.Stream
	if d, ok := sipapi.StreamList.Response.Load(ssrc); ok {
		params := d.(*sipapi.Streams)
		if params.Stream {
			if params.StreamType == m.StreamTypePush {
				// 存在推流记录关闭当前，重新发起推流
				sipapi.SipStopPlay(ssrc)
				logrus.Infoln("closeStream stream pushed!", req.Stream)
			} else {
				// 拉流的，重新拉流
				sipapi.SipPlay(params)
				logrus.Infoln("closeStream stream pulled!", req.Stream)
			}
		} else {
			if time.Now().Unix() > params.Ext {
				// 发送请求，但超时未接收到推流数据，关闭流
				sipapi.SipStopPlay(ssrc)
				logrus.Infoln("closeStream stream wait timeout", req.Stream)
			}
		}
	}
	c.JSON(http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
	})
}

type ZLMStreamNoneReaderData struct {
	APP    string `json:"app"`
	Stream string `json:"stream"`
	Schema string `json:"schema"`
}

func zlmStreamNoneReader(c *gin.Context) {
	body := c.Request.Body
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		c.JSON(http.StatusOK, map[string]any{
			"code": -1,
			"msg":  "body error",
		})
		return
	}
	req := &ZLMStreamNoneReaderData{}
	if err := utils.JSONDecode(data, &req); err != nil {
		c.JSON(http.StatusOK, map[string]any{
			"code": -1,
			"msg":  "body error",
		})
		return
	}
	sipapi.SipStopPlay(req.Stream)
	c.JSON(http.StatusOK, map[string]any{
		"code":  0,
		"close": true,
	})
	logrus.Infoln("closeStream on_stream_none_reader", req.Stream)
}
