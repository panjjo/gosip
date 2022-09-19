package api

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/panjjo/gosip/db"
	"github.com/panjjo/gosip/m"
	sipapi "github.com/panjjo/gosip/sip"
	"github.com/sirupsen/logrus"
)

// @Summary     监控播放（直播/回放）
// @Description 直播一个通道最多存在一个流，回放每请求一次生成一个流
// @Tags        streams
// @Accept      x-www-form-urlencoded
// @Produce     json
// @Param       id     path     string true  "通道id"
// @Param       replay formData int    false "是否回放，1回放，0直播，默认0"
// @Param       start  formData int    false "回放开始时间，时间戳，replay=1时必传"
// @Param       end    formData int    false "回放结束时间，时间戳，replay=1时必传"
// @Success     0      {object} sipapi.Streams
// @Failure     1000 {object} string
// @Failure     1001 {object} string
// @Failure     1002 {object} string
// @Failure     1003 {object} string
// @Router      /channels/{id}/streams [post]
func Play(c *gin.Context) {
	channelid := c.Param("id")
	pm := &sipapi.Streams{S: time.Time{}, E: time.Time{}, ChannelID: channelid, Ttag: db.M{}, Ftag: db.M{}}
	if c.PostForm("replay") == "1" {
		// 回放，获取时间
		pm.T = 1
		s, _ := strconv.ParseInt(c.PostForm("start"), 10, 64)
		if s == 0 {
			m.JsonResponse(c, m.StatusParamsERR, "开始时间错误")
			return
		}
		pm.S = time.Unix(s, 0)
		e, _ := strconv.ParseInt(c.PostForm("end"), 10, 64)
		pm.E = time.Unix(e, 0)
		if s >= e {
			m.JsonResponse(c, m.StatusParamsERR, "开始时间>=结束时间")
			return
		}
	} else {
		// 直播 判断当前通道是否存在流了。
		if succ, ok := sipapi.StreamList.Succ.Load(channelid); ok {
			m.JsonResponse(c, m.StatusSucc, succ)
			return
		}
	}
	res, err := sipapi.SipPlay(pm)
	if err != nil {
		m.JsonResponse(c, m.StatusParamsERR, err.Error())
		return
	}
	m.JsonResponse(c, m.StatusSucc, res)
}

// @Summary     停止播放（直播/回放）
// @Description 无人观看5分钟自动关闭，直播流无需调用此接口。
// @Tags        streams
// @Accept      x-www-form-urlencoded
// @Produce     json
// @Param       id   path     string true "流id,播放接口返回的streamid"
// @Success     0    {object} string
// @Failure     1000   {object} string
// @Failure     1001   {object} string
// @Failure     1002   {object} string
// @Failure     1003   {object} string
// @Router      /streams/{id} [delete]
func Stop(c *gin.Context) {
	streamid := c.Param("id")
	if _, ok := sipapi.StreamList.Response.Load(streamid); !ok {
		m.JsonResponse(c, m.StatusParamsERR, "视频流不存在或已关闭")
		return
	}
	sipapi.SipStopPlay(streamid)
	logrus.Infoln("closeStream apiStopPlay", streamid)
	m.JsonResponse(c, m.StatusSucc, "")
}

type StreamsListResponse struct {
	Total int64
	List  []sipapi.Streams
}

// @Summary     视频流列表接口
// @Description 可以根据查询条件查询视频流列表
// @Tags        streams
// @Accept      x-www-form-urlencoded
// @Produce     json
// @Param       limit   query    integer false "条数(0-100) 默认20"
// @Param       skip    query    integer false "间隔 默认0"
// @Param       sort    query    string  false "排序,例:-key,根据key倒序,key,根据key正序"
// @Param       filters query    string  false "查询条件,使用规则详情请看帮助"
// @Success     0       {object} StreamsListResponse
// @Failure     1000    {object} string
// @Failure     1001    {object} string
// @Failure     1002    {object} string
// @Failure     1003    {object} string
// @Router      /streams [get]
func StreamsList(c *gin.Context) {
	limit := m.GetLimit(c)
	skip := m.GetSkip(c)
	sort := m.GetSort(c)
	streams := []sipapi.Streams{}
	total, err := db.FindWithJson(db.DBClient, new(sipapi.Streams), &streams, c.Query("filters"), sort, skip, limit, true)
	if err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	m.JsonResponse(c, m.StatusSucc, StreamsListResponse{
		Total: total,
		List:  streams,
	})
}
