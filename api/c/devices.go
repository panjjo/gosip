package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/panjjo/gorm"
	"github.com/panjjo/gosip/db"
	"github.com/panjjo/gosip/m"
	sipapi "github.com/panjjo/gosip/sip"
)

// @Summary     设备新增接口
// @Description 通过此接口新增一个设备，获取设备id
// @Tags        devices
// @Accept      x-www-form-urlencoded
// @Produce     json
// @Param       pwd  formData string true "设备密码(GB28181认证密码)"
// @Param       name formData string true "设备名称"
// @Success     0    {object} sipapi.Devices
// @Failure     1000 {object} string
// @Failure     1001 {object} string
// @Failure     1002 {object} string
// @Failure     1003 {object} string
// @Router      /devices [post]
func DevicesCreate(c *gin.Context) {
	pwd := c.PostForm("pwd")
	if pwd == "" {
		m.JsonResponse(c, m.StatusParamsERR, "密码不能为空")
		return
	}
	name := c.PostForm("name")
	device := sipapi.Devices{
		DeviceID: fmt.Sprintf("%s%06d", m.MConfig.GB28181.DID, m.MConfig.GB28181.DNUM+1),
		Region:   m.MConfig.GB28181.Region,
		PWD:      pwd,
		Name:     name,
	}
	if device.Name == "" {
		device.Name = device.DeviceID
	}
	tx, err := db.NewTx(db.DBClient)
	if err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	defer tx.End()
	if err := db.Create(tx.DB(), &device); err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	if _, err := db.UpdateAll(tx.DB(), new(m.SysInfo), db.M{}, db.M{"dnum": gorm.Expr("dnum+1")}); err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	tx.Commit()
	m.MConfig.GB28181.DNUM += 1

	device.Sys = *m.MConfig.GB28181
	m.JsonResponse(c, m.StatusSucc, device)
}

// @Summary     设备修改接口
// @Description 调整设备信息
// @Tags        devices
// @Accept      x-www-form-urlencoded
// @Produce     json
// @Param       id   path     string true "设备id"
// @Param       pwd  formData string false "设备密码(GB28181认证密码)"
// @Param       name formData string false "设备名称"
// @Success     0    {object} sipapi.Devices
// @Failure     1000 {object} string
// @Failure     1001 {object} string
// @Failure     1002 {object} string
// @Failure     1003 {object} string
// @Router      /devices/{id} [post]
func DevicesUpdate(c *gin.Context) {
	deviceid := c.Param("id")

	device := &sipapi.Devices{
		DeviceID: deviceid,
	}
	if err := db.Get(db.DBClient, device); err != nil {
		if db.RecordNotFound(err) {
			m.JsonResponse(c, m.StatusParamsERR, "设备id不存在")
			return
		}
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}

	pwd := c.PostForm("pwd")
	if pwd != "" {
		device.PWD = pwd
	}
	name := c.PostForm("name")
	if name != "" {
		device.Name = name
	}
	if err := db.Save(db.DBClient, device); err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	m.JsonResponse(c, m.StatusSucc, device)
}

type DevicesListResponse struct {
	Total int64
	List  []sipapi.Devices
}

// @Summary     设备列表接口
// @Description 可以根据查询条件查询设备列表
// @Tags        devices
// @Accept      x-www-form-urlencoded
// @Produce     json
// @Param       limit   query    integer false "条数(0-100) 默认20"
// @Param       skip    query    integer false "间隔 默认0"
// @Param       sort    query    string  false "排序,例:-key,根据key倒序,key,根据key正序"
// @Param       filters query    string  false "查询条件,使用规则详情请看帮助"
// @Success     0       {object} DevicesListResponse
// @Failure     1000    {object} string
// @Failure     1001    {object} string
// @Failure     1002    {object} string
// @Failure     1003    {object} string
// @Router      /devices [get]
func DevicesList(c *gin.Context) {
	limit := m.GetLimit(c)
	skip := m.GetSkip(c)
	sort := m.GetSort(c)
	devices := []sipapi.Devices{}
	total, err := db.FindWithJson(db.DBClient, new(sipapi.Devices), &devices, c.Query("filters"), sort, skip, limit, true)
	if err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	m.JsonResponse(c, m.StatusSucc, DevicesListResponse{
		Total: total,
		List:  devices,
	})
}

// @Summary     设备删除接口
// @Description 删除设备信息
// @Tags        devices
// @Accept      x-www-form-urlencoded
// @Produce     json
// @Param       id   path     string true  "设备id"
// @Success     0    {object} string
// @Failure     1000 {object} string
// @Failure     1001 {object} string
// @Failure     1002 {object} string
// @Failure     1003 {object} string
// @Router      /devices/{id} [delete]
func DevicesDelete(c *gin.Context) {
	deviceid := c.Param("id")

	device := &sipapi.Devices{
		DeviceID: deviceid,
	}
	if err := db.Get(db.DBClient, device); err != nil {
		if db.RecordNotFound(err) {
			m.JsonResponse(c, m.StatusParamsERR, "设备id不存在")
			return
		}
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	tx, err := db.NewTx(db.DBClient)
	if err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	defer tx.End()
	if err := db.Del(tx.DB(), device); err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	// 删除设备，同时删除设备下的所有通道
	if err := db.Del(tx.DB(), &sipapi.Channels{DeviceID: deviceid}); err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	tx.Commit()
	m.JsonResponse(c, m.StatusSucc, "")
}

// // 视频流录制 默认保存为mp4文件，录制最多录制10分钟，10分钟后自动停止，一个流只能存在一个录制
// func apiRecordStart(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	id := ps.ByName("id")
// 	if _, ok := _playList.ssrcResponse.Load(id); !ok {
// 		_apiResponse(w, statusParamsERR, "视频流不存在")
// 		return
// 	}
// 	if _, ok := _apiRecordList.Get(id); ok {
// 		_apiResponse(w, statusParamsERR, "视频流存在未完成录制")
// 		return
// 	}
// 	values := url.Values{}
// 	values.Set("secret", config.Media.Secret)
// 	values.Set("type", "1")
// 	values.Set("vhost", "__defaultVhost__")
// 	values.Set("app", "rtp")
// 	values.Set("stream", id)
// 	req := r.URL.Query()
// 	item := _apiRecordList.Start(id, values)
// 	item.req = req
// 	code, data := item.start()
// 	if code != statusSucc {
// 		_apiRecordList.Stop(id)
// 		data = fmt.Sprintf("录制失败:%v", data)
// 	}
// 	_apiResponse(w, code, data)
// }

// // 停止录制，传入录制时返回的data字段
// func apiRecordStop(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	id := ps.ByName("id")

// 	if item, ok := _apiRecordList.Get(id); !ok {
// 		_apiResponse(w, statusParamsERR, "录制不存在或已结束")
// 	} else {
// 		code, data := item.stop()
// 		if code == statusSucc {
// 			item.clos <- true
// 		} else {
// 			data = fmt.Sprintf("停止录制失败:%v", data)
// 		}
// 		url := <-item.resp
// 		_apiResponse(w, code, url)
// 	}
// }

// // 拉流代理，用来转换非GB28281的播放源,代理只支持直播
// func apiAddProxy(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	url := r.URL.Query().Get("url")
// 	tag := r.URL.Query().Get("tag")
// 	if tag == "" || url == "" {
// 		_apiResponse(w, statusParamsERR, "参数错误，url或tag为空")
// 	}
// 	// 代理流 tag 作为播放的ssrc和deviceid
// 	// 检查tag是否存在
// 	if succ, ok := _playList.devicesSucc.Load(tag); ok {
// 		_apiResponse(w, statusSucc, succ)
// 		return
// 	}
// 	// 不存在进行新增
// 	d := playParams{S: time.Time{}, E: time.Time{}, DeviceID: tag, SSRC: tag, streamType: streamTypeProxy, Url: url}
// 	res := sipPlay(d)
// 	switch res.(type) {
// 	case error, string:
// 		_apiResponse(w, statusParamsERR, res)
// 		return
// 	default:
// 		_apiResponse(w, statusSucc, res)
// 		return
// 	}
// }

// // 删除拉流代理，用来转换非GB28281的播放源,代理只支持直播
// func apiDelProxy(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	tag := r.URL.Query().Get("tag")
// 	if _, ok := _playList.ssrcResponse.Load(tag); !ok {
// 		_apiResponse(w, statusSucc, "视频流不存在或已关闭")
// 		return
// 	}
// 	sipStopPlay(tag)
// 	logrus.Infoln("closeStream apiStopPlay", tag)
// 	_apiResponse(w, statusSucc, "")
// }

// type mediaRequest struct {
// 	APP    string `json:"app"`
// 	Params string `json:"params"`
// 	Stream string `json:"stream"`
// 	Schema string `json:"schema"`
// 	URL    string `json:"url"`
// 	Regist bool   `json:"regist"`
// }

// func _mediaResponse(w http.ResponseWriter, data interface{}) {
// 	w.Header().Add("Content-Type", "application/json")
// 	_, err := w.Write(utils.JSONEncode(data))
// 	if err != nil {
// 		logrus.Errorln("send response api fail.", err)
// 	}
// }

// func restfulAPI() {
// 	router := httprouter.New()
// 	router.GET("/users", apiAuthCheck(apiNewUsers, config.Secret))                // 注册新用户设备
// 	router.GET("/users/:id/update", apiAuthCheck(apiUpdateUsers, config.Secret))  // 更新用户设备
// 	router.GET("/users/:id/delete", apiAuthCheck(apiDelUsers, config.Secret))     // 更新用户设备
// 	router.GET("/users/:id/devices", apiAuthCheck(apiNewDevices, config.Secret))  // 注册新通道设备
// 	router.GET("/devices/:id/delete", apiAuthCheck(apiDelDevices, config.Secret)) // 删除
// 	router.GET("/devices/:id/play", apiAuthCheck(apiPlay, config.Secret))         // 播放
// 	router.GET("/devices/:id/replay", apiAuthCheck(apiReplay, config.Secret))     // 回播
// 	router.GET("/play/:id/stop", apiAuthCheck(apiStopPlay, config.Secret))        // 停止播放
// 	router.GET("/devices/:id/files", apiAuthCheck(apiFileList, config.Secret))    // 获取历史文件
// 	router.GET("/play/:id/record", apiAuthCheck(apiRecordStart, config.Secret))   // 录制
// 	router.GET("/record/:id/stop", apiAuthCheck(apiRecordStop, config.Secret))    // 停止录制
// 	router.GET("/addproxy", apiAuthCheck(apiAddProxy, config.Secret))             // 增加拉流代理
// 	router.GET("/delproxy", apiAuthCheck(apiDelProxy, config.Secret))             // 增加拉流代理
// 	router.POST("/index/hook/:method", apiWebHooks)
// 	logrus.Fatal(http.ListenAndServe(config.API, router))
// }
