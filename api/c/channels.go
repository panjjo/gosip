package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/panjjo/gorm"
	"github.com/panjjo/gosip/db"
	"github.com/panjjo/gosip/m"
	sipapi "github.com/panjjo/gosip/sip"
)

// @Summary     通道新增接口
// @Description 通过此接口在设备下新增通道，获取通道id
// @Tags        channels
// @Accept      x-www-form-urlencoded
// @Produce     json
// @Param       id         path     string true  "设备id"
// @Param       memo       formData string false "通道备注"
// @Param       streamtype formData string false "播放类型，pull 媒体服务器拉流，push 摄像头推流,默认push"
// @Param       url        formData string false "静态拉流地址，streamtype=pull 时生效。"
// @Success     0          {object} sipapi.Channels
// @Failure     1000    {object} string
// @Failure     1001    {object} string
// @Failure     1002    {object} string
// @Failure     1003    {object} string
// @Router      /devices/{id}/channels [post]
func ChannelCreate(c *gin.Context) {

	id := c.Param("id")
	if id == "" {
		m.JsonResponse(c, m.StatusParamsERR, "缺少设备ID")
		return
	}
	device := sipapi.Devices{DeviceID: id}
	if err := db.Get(db.DBClient, &device); err != nil {
		if db.RecordNotFound(err) {
			m.JsonResponse(c, m.StatusParamsERR, "用户设备不存在")
			return
		}
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}

	channel := sipapi.Channels{
		ChannelID: fmt.Sprintf("%s%06d", m.MConfig.GB28181.CID, m.MConfig.GB28181.CNUM+1),
		DeviceID:  device.DeviceID,
		MeMo:      c.PostForm("memo"),
	}
	streamtype := c.PostForm("streamtype")
	if streamtype == m.StreamTypePull {
		channel.StreamType = m.StreamTypePull
		channel.URL = c.PostForm("url")
	} else {
		channel.StreamType = m.StreamTypePush
	}
	tx, err := db.NewTx(db.DBClient)
	if err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	defer tx.End()
	if err := db.Create(tx.DB(), &channel); err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	if _, err := db.UpdateAll(tx.DB(), new(m.SysInfo), db.M{}, db.M{"cnum": gorm.Expr("cnum+1")}); err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	tx.Commit()
	m.MConfig.GB28181.CNUM += 1

	m.JsonResponse(c, m.StatusSucc, channel)
}

// @Summary     通道修改接口
// @Description 调整通道信息
// @Tags        channels
// @Accept      x-www-form-urlencoded
// @Produce     json
// @Param       id         path     string true  "通道id"
// @Param       memo       formData string false "通道备注"
// @Param       streamtype formData string false "播放类型，pull 媒体服务器拉流，push 摄像头推流,默认push"
// @Param       url        formData string false "静态拉流地址，streamtype=pull 时生效。"
// @Success     0          {object} sipapi.Channels
// @Failure     1000       {object} string
// @Failure     1001       {object} string
// @Failure     1002       {object} string
// @Failure     1003       {object} string
// @Router      /channels/{id} [post]
func ChannelsUpdate(c *gin.Context) {
	channelid := c.Param("id")

	channel := &sipapi.Channels{
		ChannelID: channelid,
	}
	if err := db.Get(db.DBClient, channel); err != nil {
		if db.RecordNotFound(err) {
			m.JsonResponse(c, m.StatusParamsERR, "通道id不存在")
			return
		}
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}

	memo := c.PostForm("memo")
	if memo != "" {
		channel.MeMo = memo
	}
	streamtype := c.PostForm("streamtype")
	if streamtype != "" {
		channel.StreamType = streamtype
		if channel.StreamType == m.StreamTypePush {
			channel.URL = ""
		}
	}
	url := c.PostForm("url")
	if streamtype != "" && channel.StreamType == m.StreamTypePull {
		channel.URL = url
	}

	if err := db.Save(db.DBClient, channel); err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	m.JsonResponse(c, m.StatusSucc, channel)
}

type ChannelsListResponse struct {
	Total int64
	List  []sipapi.Channels
}

// @Summary     通道列表接口
// @Description 可以根据查询条件查询通道列表
// @Tags        channels
// @Accept      x-www-form-urlencoded
// @Produce     json
// @Param       limit   query    integer false "条数(0-100) 默认20"
// @Param       skip    query    integer false "间隔 默认0"
// @Param       sort    query    string  false "排序,例:-key,根据key倒序,key,根据key正序"
// @Param       filters query    string  false "查询条件,使用规则详情请看帮助"
// @Success     0       {object} ChannelsListResponse
// @Failure     1000       {object} string
// @Failure     1001       {object} string
// @Failure     1002       {object} string
// @Failure     1003       {object} string
// @Router      /channels [get]
func ChannelsList(c *gin.Context) {
	limit := m.GetLimit(c)
	skip := m.GetSkip(c)
	sort := m.GetSort(c)
	channels := []sipapi.Channels{}
	total, err := db.FindWithJson(db.DBClient, new(sipapi.Channels), &channels, c.Query("filters"), sort, skip, limit, true)
	if err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	m.JsonResponse(c, m.StatusSucc, ChannelsListResponse{
		Total: total,
		List:  channels,
	})
}

// @Summary     通道删除接口
// @Description 删除通道信息
// @Tags        channels
// @Accept      x-www-form-urlencoded
// @Produce     json
// @Param       id   path     string true "通道id"
// @Success     0    {object} string
// @Failure     1000 {object} string
// @Failure     1001 {object} string
// @Failure     1002 {object} string
// @Failure     1003 {object} string
// @Router      /channels/{id} [delete]
func ChannelsDelete(c *gin.Context) {
	channelid := c.Param("id")

	channel := &sipapi.Channels{
		ChannelID: channelid,
	}
	if err := db.Get(db.DBClient, channel); err != nil {
		if db.RecordNotFound(err) {
			m.JsonResponse(c, m.StatusParamsERR, "通道id不存在")
			return
		}
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	if err := db.Del(db.DBClient, channel); err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	m.JsonResponse(c, m.StatusSucc, "")
}
