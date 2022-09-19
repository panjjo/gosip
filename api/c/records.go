package api

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/panjjo/gosip/db"
	"github.com/panjjo/gosip/m"
	sipapi "github.com/panjjo/gosip/sip"
)

// @Summary     回放文件时间列表
// @Description 用来获取通道设备存储的可回放时间段列表，注意控制时间跨度，跨度越大，数据量越多，返回越慢，甚至会超时（最多10s）。
// @Tags        records
// @Accept      x-www-form-urlencoded
// @Produce     json
// @Param       id    path     string true "通道id"
// @Param       start query    int    true "开始时间，时间戳"
// @Param       end   query    int    true "结束时间，时间戳"
// @Success     0     {object} sipapi.Records
// @Failure     1000  {object} string
// @Failure     1001  {object} string
// @Failure     1002  {object} string
// @Failure     1003  {object} string
// @Router      /channels/{id}/records [get]
func RecordsList(c *gin.Context) {
	channelid := c.Param("id")
	start := c.Query("start")
	end := c.Query("end")

	if start == "" {
		m.JsonResponse(c, m.StatusParamsERR, "开始时间错误")
		return
	}
	startStamp, err := strconv.ParseInt(start, 10, 64)
	if err != nil || startStamp <= 0 {
		m.JsonResponse(c, m.StatusParamsERR, "开始时间错误")
		return
	}
	if end == "" {
		m.JsonResponse(c, m.StatusParamsERR, "结束时间错误")
		return
	}
	endStamp, err := strconv.ParseInt(end, 10, 64)
	if err != nil || endStamp <= 0 || endStamp <= startStamp {
		m.JsonResponse(c, m.StatusParamsERR, "结束时间错误")
		return
	}

	channel := &sipapi.Channels{ChannelID: channelid}
	if err := db.Get(db.DBClient, channel); err != nil {
		if db.RecordNotFound(err) {
			m.JsonResponse(c, m.StatusParamsERR, "通道不存在")
			return
		}
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	if channel.Status != m.DeviceStatusON || time.Now().Unix()-channel.Active > 30*60 {
		m.JsonResponse(c, m.StatusParamsERR, "通道已离线")
		return
	}
	res, err := sipapi.SipRecordList(channel, startStamp, endStamp)
	if err != nil {
		m.JsonResponse(c, m.StatusParamsERR, err)
		return
	}
	m.JsonResponse(c, m.StatusSucc, res)
}
