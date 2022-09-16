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
// @Success     0     {object} sipapi.Devices
// @Failure     1000  {object} string
// @Failure     1001  {object} string
// @Failure     1002  {object} string
// @Failure     1003  {object} string
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

// @Summary     设备列表接口
// @Description 通过此接口查询设备列表
// @Tags        devices
// @Accept      x-www-form-urlencoded
// @Produce     json
// @Param       limit query    string false "每页条数，默认20，最大100"
// @Param       page  query    string false "页数，默认0"
// @Success     0    {object} sipapi.Devices
// @Failure     1000 {object} string
// @Failure     1001 {object} string
// @Failure     1002 {object} string
// @Failure     1003 {object} string
// @Router      /devices [get]
func DevicesList(c *gin.Context) {
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

// func apiAuthCheck(h httprouter.Handle, requiredPassword string) httprouter.Handle {
// 	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 		// Get the Basic Authentication credentials
// 		if ok, msg := checkSign(r.RequestURI, requiredPassword, r.Form); ok {
// 			// Delegate request to the given handle
// 			h(w, r, ps)
// 		} else {
// 			// Request Basic Authentication otherwise
// 			_apiResponse(w, statusAuthERR, msg)
// 		}
// 	}
// }

// type apiResponse struct {
// 	C  string      `json:"code"`
// 	D  interface{} `json:"data"`
// 	T  int64       `json:"time"`
// 	ID string      `json:"id"`
// }

// func _apiResponse(w http.ResponseWriter, code string, data interface{}) {
// 	w.WriteHeader(code2code[code])
// 	w.Header().Add("Content-Type", "application/json")
// 	_, err := w.Write(utils.JSONEncode(apiResponse{
// 		code, data, time.Now().Unix(), utils.RandString(16),
// 	}))
// 	if err != nil {
// 		logrus.Errorln("send response api fail.", err)
// 	}
// }

// // 注册NVR用户设备
// func apiNewUsers(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

// }

// // 更新NVR用户设备
// func apiUpdateUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	id := ps.ByName("id")
// 	if id == "" {
// 		_apiResponse(w, statusParamsERR, "缺少用户设备ID")
// 		return
// 	}
// 	user := NVRDevices{}
// 	err := dbClient.Get(userTB, M{"deviceid": id}, &user)
// 	if err != nil {
// 		if err == mongo.ErrNoDocuments {
// 			_apiResponse(w, statusParamsERR, "用户设备不存在")
// 			return
// 		}
// 		_apiResponse(w, statusDBERR, err)
// 		return
// 	}
// 	pwd := r.URL.Query().Get("pwd")
// 	if pwd == "" {
// 		_apiResponse(w, statusParamsERR, "密码不能为空")
// 		return
// 	}
// 	update := M{}
// 	if pwd != user.PWD {
// 		update["pwd"] = pwd
// 		user.PWD = pwd
// 	}
// 	name := r.URL.Query().Get("name")
// 	if name != user.Name {
// 		update["name"] = name
// 		user.Name = name
// 	}
// 	err = dbClient.Update(userTB, M{"deviceid": user.DeviceID}, M{"$set": update})
// 	if err != nil {
// 		_apiResponse(w, statusDBERR, err)
// 		return
// 	}
// 	_apiResponse(w, statusSucc, user)
// }

// // 删除NVR用户设备，同时会删除所有归属的通道设备
// func apiDelUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	id := ps.ByName("id")
// 	if id == "" {
// 		_apiResponse(w, statusParamsERR, "缺少用户设备ID")
// 		return
// 	}
// 	user := NVRDevices{}
// 	err := dbClient.Get(userTB, M{"deviceid": id}, &user)
// 	if err != nil {
// 		if err == mongo.ErrNoDocuments {
// 			_apiResponse(w, statusParamsERR, "用户设备不存在")
// 			return
// 		}
// 		_apiResponse(w, statusDBERR, err)
// 		return
// 	}
// 	dbClient.DelMany(deviceTB, M{"pdid": user.DeviceID})
// 	dbClient.Del(userTB, M{"deviceid": user.DeviceID})
// 	_apiResponse(w, statusSucc, "")
// }

// // 注册通道设备
// func apiNewDevices(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
//
// }

// // 删除通道设备
// func apiDelDevices(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	id := ps.ByName("id")
// 	if id == "" {
// 		_apiResponse(w, statusParamsERR, "缺少监控设备ID")
// 		return
// 	}
// 	user := Devices{}
// 	err := dbClient.Get(deviceTB, M{"deviceid": id}, &user)
// 	if err != nil {
// 		if err == mongo.ErrNoDocuments {
// 			_apiResponse(w, statusParamsERR, "监控设备不存在")
// 			return
// 		}
// 		_apiResponse(w, statusDBERR, err)
// 		return
// 	}
// 	dbClient.Del(deviceTB, M{"deviceid": user.DeviceID})
// 	_apiResponse(w, statusSucc, "")
// }

// // 直播 同一通道设备公用一个直播流
// func apiPlay(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	deviceid := ps.ByName("id")
// 	d := playParams{S: time.Time{}, E: time.Time{}, DeviceID: deviceid}
// 	params := r.URL.Query()
// 	if params.Get("t") == "1" {
// 		d.T = 1
// 		s, _ := strconv.ParseInt(params.Get("start"), 10, 64)
// 		if s == 0 {
// 			_apiResponse(w, statusParamsERR, "开始时间错误")
// 			return
// 		}
// 		d.S = time.Unix(s, 0)
// 		e, _ := strconv.ParseInt(params.Get("end"), 10, 64)
// 		d.E = time.Unix(e, 0)
// 	} else {
// 		// 直播的判断当前是否存在播放
// 		if succ, ok := _playList.devicesSucc.Load(deviceid); ok {
// 			_apiResponse(w, statusSucc, succ)
// 			return
// 		}
// 	}
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

// // 重播，每个重播请求都会生成一个新直播流
// func apiReplay(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	r.URL.RawQuery = r.URL.RawQuery + "&t=1"
// 	apiPlay(w, r, ps)
// }

// // 停止播放（直播/重播）
// func apiStopPlay(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	id := ps.ByName("id")
// 	if _, ok := _playList.ssrcResponse.Load(id); !ok {
// 		_apiResponse(w, statusSucc, "视频流不存在或已关闭")
// 		return
// 	}
// 	sipStopPlay(id)
// 	logrus.Infoln("closeStream apiStopPlay", id)
// 	_apiResponse(w, statusSucc, "")
// }

// // 获取录像文件列表
// func apiFileList(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	id := ps.ByName("id")
// 	device := Devices{}
// 	if err := dbClient.Get(deviceTB, M{"deviceid": id}, &device); err != nil {
// 		if err == mongo.ErrNoDocuments {
// 			_apiResponse(w, statusParamsERR, "监控设备不存在")
// 			return
// 		}
// 		_apiResponse(w, statusDBERR, err)
// 		return
// 	}
// 	if time.Now().Unix()-device.Active > 30*60 {
// 		_apiResponse(w, statusParamsERR, "监控设备已掉线")
// 		return
// 	}
// 	params := r.URL.Query()
// 	start, _ := strconv.ParseInt(params.Get("start"), 10, 64)
// 	if start == 0 {
// 		_apiResponse(w, statusParamsERR, "开始时间错误")
// 		return
// 	}
// 	end, _ := strconv.ParseInt(params.Get("end"), 10, 64)
// 	if end == 0 {
// 		_apiResponse(w, statusParamsERR, "结束时间错误")
// 		return
// 	}
// 	if start >= end {
// 		_apiResponse(w, statusParamsERR, "开始时间不能小于结束时间")
// 		return
// 	}
// 	user := NVRDevices{}
// 	user, ok := _activeDevices.Get(device.PDID)
// 	if !ok {
// 		_apiResponse(w, statusParamsERR, "用户设备已掉线")
// 		return
// 	}
// 	for {
// 		if _, ok := _recordList.Load(device.DeviceID); ok {
// 			time.Sleep(1 * time.Second)
// 		} else {
// 			break
// 		}
// 	}

// 	user.DeviceID = device.DeviceID
// 	deviceURI, _ := sip.ParseURI(device.URIStr)
// 	user.addr = &sip.Address{URI: deviceURI}
// 	resp := make(chan interface{}, 1)
// 	defer close(resp)
// 	_recordList.Store(user.DeviceID, recordList{deviceid: user.DeviceID, resp: resp, data: [][]int64{}, l: &sync.Mutex{}, s: start, e: end})
// 	defer _recordList.Delete(user.DeviceID)
// 	err := sipRecordList(user, start, end)
// 	if err != nil {
// 		_apiResponse(w, statusParamsERR, "监控设备返回错误"+err.Error())
// 		return
// 	}
// 	select {
// 	case res := <-resp:
// 		_apiResponse(w, statusSucc, res)
// 	case <-time.Tick(10 * time.Second):
// 		// 10秒未完成返回当前获取到的数据
// 		if list, ok := _recordList.Load(user.DeviceID); ok {
// 			info := list.(recordList)
// 			_apiResponse(w, statusSucc, transRecordList(info.data))
// 			return
// 		}
// 		_apiResponse(w, statusSysERR, "获取超时")
// 	}
// }

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
