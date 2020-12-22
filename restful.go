package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/panjjo/gosip/sip"
	"github.com/panjjo/gosip/utils"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

func apiAuthCheck(h httprouter.Handle, requiredPassword string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// Get the Basic Authentication credentials
		if ok, msg := checkSign(r.RequestURI, requiredPassword, r.Form); ok {
			// Delegate request to the given handle
			h(w, r, ps)
		} else {
			// Request Basic Authentication otherwise
			_apiResponse(w, statusAuthERR, msg)
		}
	}
}

type apiResponse struct {
	C  string      `json:"code"`
	D  interface{} `json:"data"`
	T  int64       `json:"time"`
	ID string      `json:"id"`
}

func _apiResponse(w http.ResponseWriter, code string, data interface{}) {
	w.WriteHeader(code2code[code])
	w.Header().Add("Content-Type", "application/json")
	_, err := w.Write(utils.JSONEncode(apiResponse{
		code, data, time.Now().Unix(), utils.RandString(16),
	}))
	if err != nil {
		logrus.Errorln("send response api fail.", err)
	}
}

// 注册NVR用户设备
func apiNewUsers(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pwd := r.URL.Query().Get("pwd")
	if pwd == "" {
		_apiResponse(w, statusParamsERR, "密码不能为空")
		return
	}
	user := NVRDevices{
		DeviceID: fmt.Sprintf("%s%06d", _sysinfo.UID, _sysinfo.UNUM+1),
		Region:   _sysinfo.Region,
		PWD:      pwd,
		Name:     r.URL.Query().Get("name"),
	}
	if user.Name == "" {
		user.Name = user.DeviceID
	}
	if err := dbClient.Insert(userTB, user); err != nil {
		_apiResponse(w, statusDBERR, err)
		return
	}
	if err := dbClient.Update(sysTB, M{}, M{"$inc": M{"unum": 1}}); err != nil {
		_apiResponse(w, statusDBERR, err)
		return
	}
	_sysinfo.UNUM++
	user.Sys = _sysinfo
	_apiResponse(w, statusSucc, user)
}

// 更新NVR用户设备
func apiUpdateUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	if id == "" {
		_apiResponse(w, statusParamsERR, "缺少用户设备ID")
		return
	}
	user := NVRDevices{}
	err := dbClient.Get(userTB, M{"deviceid": id}, &user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			_apiResponse(w, statusParamsERR, "用户设备不存在")
			return
		}
		_apiResponse(w, statusDBERR, err)
		return
	}
	pwd := r.URL.Query().Get("pwd")
	if pwd == "" {
		_apiResponse(w, statusParamsERR, "密码不能为空")
		return
	}
	update := M{}
	if pwd != user.PWD {
		update["pwd"] = pwd
		user.PWD = pwd
	}
	name := r.URL.Query().Get("name")
	if name != user.Name {
		update["name"] = name
		user.Name = name
	}
	err = dbClient.Update(userTB, M{"deviceid": user.DeviceID}, M{"$set": update})
	if err != nil {
		_apiResponse(w, statusDBERR, err)
		return
	}
	_apiResponse(w, statusSucc, user)
}

// 删除NVR用户设备，同时会删除所有归属的通道设备
func apiDelUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	if id == "" {
		_apiResponse(w, statusParamsERR, "缺少用户设备ID")
		return
	}
	user := NVRDevices{}
	err := dbClient.Get(userTB, M{"deviceid": id}, &user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			_apiResponse(w, statusParamsERR, "用户设备不存在")
			return
		}
		_apiResponse(w, statusDBERR, err)
		return
	}
	dbClient.DelMany(deviceTB, M{"pdid": user.DeviceID})
	dbClient.Del(userTB, M{"deviceid": user.DeviceID})
	_apiResponse(w, statusSucc, "")
}

// 注册通道设备
func apiNewDevices(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	if id == "" {
		_apiResponse(w, statusParamsERR, "缺少用户设备ID")
		return
	}
	user := NVRDevices{}
	err := dbClient.Get(userTB, M{"deviceid": id}, &user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			_apiResponse(w, statusParamsERR, "用户设备不存在")
			return
		}
		_apiResponse(w, statusDBERR, err)
		return
	}
	device := DeviceItem{
		DeviceID: fmt.Sprintf("%s%06d", _sysinfo.DID, _sysinfo.DNUM+1),
		PDID:     id,
	}
	if err := dbClient.Insert(deviceTB, device); err != nil {
		_apiResponse(w, statusDBERR, err)
		return
	}
	if err := dbClient.Update(sysTB, M{}, M{"$inc": M{"dnum": 1}}); err != nil {
		_apiResponse(w, statusDBERR, err)
		return
	}
	_sysinfo.DNUM++
	_apiResponse(w, statusSucc, device)
}

// 删除通道设备
func apiDelDevices(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	if id == "" {
		_apiResponse(w, statusParamsERR, "缺少监控设备ID")
		return
	}
	user := DeviceItem{}
	err := dbClient.Get(deviceTB, M{"deviceid": id}, &user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			_apiResponse(w, statusParamsERR, "监控设备不存在")
			return
		}
		_apiResponse(w, statusDBERR, err)
		return
	}
	dbClient.Del(deviceTB, M{"deviceid": user.DeviceID})
	_apiResponse(w, statusSucc, "")
}

// 直播 同一通道设备公用一个直播流
func apiPlay(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	deviceid := ps.ByName("id")
	d := playParams{S: time.Time{}, E: time.Time{}, DeviceID: deviceid}
	params := r.URL.Query()
	if params.Get("t") == "1" {
		d.T = 1
		s, _ := strconv.ParseInt(params.Get("start"), 10, 64)
		if s == 0 {
			_apiResponse(w, statusParamsERR, "开始时间错误")
			return
		}
		d.S = time.Unix(s, 0)
		e, _ := strconv.ParseInt(params.Get("end"), 10, 64)
		d.E = time.Unix(e, 0)
	} else {
		// 直播的判断当前是否存在播放
		if succ, ok := _playList.devicesSucc.Load(deviceid); ok {
			_apiResponse(w, statusSucc, succ)
			return
		}
	}
	res := sipPlay(d)
	switch res.(type) {
	case error, string:
		_apiResponse(w, statusParamsERR, res)
		return
	default:
		_apiResponse(w, statusSucc, res)
		return
	}
}

// 重播，每个重播请求都会生成一个新直播流
func apiReplay(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	r.URL.RawQuery = r.URL.RawQuery + "&t=1"
	apiPlay(w, r, ps)
}

// 停止播放（直播/重播）
func apiStopPlay(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	if _, ok := _playList.ssrcResponse.Load(id); !ok {
		_apiResponse(w, statusSucc, "视频流不存在或已关闭")
		return
	}
	sipStopPlay(id)
	logrus.Infoln("closeStream apiStopPlay", id)
	_apiResponse(w, statusSucc, "")
	return
}

// 获取录像文件列表
func apiFileList(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	device := DeviceItem{}
	if err := dbClient.Get(deviceTB, M{"deviceid": id}, &device); err != nil {
		if err == mongo.ErrNoDocuments {
			_apiResponse(w, statusParamsERR, "监控设备不存在")
			return
		}
		_apiResponse(w, statusDBERR, err)
		return
	}
	if time.Now().Unix()-device.Active > 30*60 {
		_apiResponse(w, statusParamsERR, "监控设备已掉线")
		return
	}
	params := r.URL.Query()
	start, _ := strconv.ParseInt(params.Get("start"), 10, 64)
	if start == 0 {
		_apiResponse(w, statusParamsERR, "开始时间错误")
		return
	}
	end, _ := strconv.ParseInt(params.Get("end"), 10, 64)
	if end == 0 {
		_apiResponse(w, statusParamsERR, "结束时间错误")
		return
	}
	if start >= end {
		_apiResponse(w, statusParamsERR, "开始时间不能小于结束时间")
		return
	}
	user := NVRDevices{}
	user, ok := _activeDevices.Get(device.PDID)
	if !ok {
		_apiResponse(w, statusParamsERR, "用户设备已掉线")
		return
	}
	for {
		if _, ok := _recordList.Load(device.DeviceID); ok {
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}

	user.DeviceID = device.DeviceID
	deviceURI, _ := sip.ParseURI(device.URIStr)
	user.addr = &sip.Address{URI: deviceURI}
	resp := make(chan interface{}, 1)
	defer close(resp)
	_recordList.Store(user.DeviceID, recordList{deviceid: user.DeviceID, resp: resp, data: [][]int64{}, l: &sync.Mutex{}, s: start, e: end})
	defer _recordList.Delete(user.DeviceID)
	err := sipRecordList(user, start, end)
	if err != nil {
		_apiResponse(w, statusParamsERR, "监控设备返回错误"+err.Error())
		return
	}
	select {
	case res := <-resp:
		_apiResponse(w, statusSucc, res)
	case <-time.Tick(10 * time.Second):
		// 10秒未完成返回当前获取到的数据
		if list, ok := _recordList.Load(user.DeviceID); ok {
			info := list.(recordList)
			_apiResponse(w, statusSucc, transRecordList(info.data))
			return
		}
		_apiResponse(w, statusSysERR, "获取超时")
	}
}

type mediaRequest struct {
	APP    string `json:"app"`
	Params string `json:"params"`
	Stream string `json:"stream"`
	Schema string `json:"schema"`
	URL    string `json:"url"`
	Regist bool   `json:"regist"`
}

func _mediaResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Add("Content-Type", "application/json")
	_, err := w.Write(utils.JSONEncode(data))
	if err != nil {
		logrus.Errorln("send response api fail.", err)
	}
}

func apiWebHooks(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	method := ps.ByName("method")
	body := r.Body
	defer body.Close()
	data, err := ioutil.ReadAll(body)
	if err != nil {
		_mediaResponse(w, map[string]interface{}{
			"code": -1,
			"msg":  "body error",
		})
		return
	}
	logrus.Debugln(method, string(data))
	req := &mediaRequest{}
	if err := utils.JSONDecode(data, &req); err != nil {
		_mediaResponse(w, map[string]interface{}{
			"code": -1,
			"msg":  "body error",
		})
		return
	}
	switch method {
	case "on_server_started":
		_sysinfo.MediaServer = true
		_mediaResponse(w, map[string]interface{}{
			"code": 0,
			"msg":  "success"})
	case "on_http_access":
		_mediaResponse(w, map[string]interface{}{
			"code":   0,
			"second": 86400,
		})
	case "on_play":
		//视频播放触发鉴权
		params, err := url.ParseQuery(req.Params)
		if err != nil {
			_mediaResponse(w, map[string]interface{}{
				"code": -1,
				"msg":  "body error",
			})
			return
		}
		if ok, msg := checkSign(fmt.Sprintf("/rtp/%s/hls.m3u8", req.Stream), config.Secret, params); ok {
			_mediaResponse(w, map[string]interface{}{
				"code": 0,
				"msg":  "",
			})
		} else {
			_mediaResponse(w, map[string]interface{}{
				"code": -1,
				"msg":  msg,
			})
		}
	case "on_publish":
		// 推流鉴权
		_mediaResponse(w, map[string]interface{}{
			"code":       0,
			"enableHls":  config.Stream.HLS,
			"enableMP4":  false,
			"enableRtxp": config.Stream.RTMP,
			"msg":        "success",
		})
	case "on_record_mp4":
		// // mp4 录制完成
		_mediaResponse(w, map[string]interface{}{
			"code": 0,
			"msg":  "success"})
	case "on_stream_none_reader":
		// 无人阅读通知
		sipStopPlay(req.Stream)
		_mediaResponse(w, map[string]interface{}{
			"code":  0,
			"close": true,
		})
		logrus.Infoln("closeStream on_stream_none_reader", req.Stream)
	case "on_stream_not_found":
		ssrc := req.Stream
		if d, ok := _playList.ssrcResponse.Load(ssrc); ok {
			params := d.(playParams)
			if params.stream {
				if params.streamType == streamTypePush {
					// 存在推流记录关闭当前，重新发起推流
					sipStopPlay(ssrc)
					logrus.Infoln("closeStream stream pushed!", req.Stream)
				} else {
					// 拉流的，重新拉流
					sipPlay(params)
					logrus.Infoln("closeStream stream pulled!", req.Stream)
				}
			} else {
				if time.Now().Unix() > params.ext {
					// 发送请求，但超时未接收到推流数据，关闭流
					sipStopPlay(ssrc)
					logrus.Infoln("closeStream stream wait timeout", req.Stream)
				}
			}
		}
		_mediaResponse(w, map[string]interface{}{
			"code": 0,
			"msg":  "success",
		})

	case "on_stream_changed":
		ssrc := req.Stream
		if req.Regist {
			if req.Schema == "rtmp" {
				d, ok := _playList.ssrcResponse.Load(ssrc)
				if ok {
					// 接收到流注册事件，更新ssrc数据
					params := d.(playParams)
					params.stream = true
					_playList.ssrcResponse.Store(params.SSRC, params)
				} else {
					// ssrc不存在，关闭流
					sipStopPlay(ssrc)
					logrus.Infoln("closeStream on_stream_changed notfound!", req.Stream)
				}
			}
		} else {
			if req.Schema == "hls" {
				//接收到流注销事件
				_, ok := _playList.ssrcResponse.Load(ssrc)
				if ok {
					// 流还存在，注销
					sipStopPlay(ssrc)
					logrus.Infoln("closeStream on_stream_changed cancel!", req.Stream)
				}
			}
		}
		_mediaResponse(w, map[string]interface{}{
			"code": 0,
			"msg":  "success"})
		//
	default:
		// logrus.Warnln(method)
	}
}

func restfulAPI() {
	router := httprouter.New()
	router.GET("/users", apiAuthCheck(apiNewUsers, config.Secret))                // 注册新用户设备
	router.GET("/users/:id/update", apiAuthCheck(apiUpdateUsers, config.Secret))  // 更新用户设备
	router.GET("/users/:id/delete", apiAuthCheck(apiDelUsers, config.Secret))     // 更新用户设备
	router.GET("/users/:id/devices", apiAuthCheck(apiNewDevices, config.Secret))  // 注册新通道设备
	router.GET("/devices/:id/delete", apiAuthCheck(apiDelDevices, config.Secret)) // 删除
	router.GET("/devices/:id/play", apiAuthCheck(apiPlay, config.Secret))         // 播放
	router.GET("/devices/:id/replay", apiAuthCheck(apiReplay, config.Secret))     // 回播
	router.GET("/play/:id/stop", apiAuthCheck(apiStopPlay, config.Secret))        // 停止播放
	router.GET("/devices/:id/files", apiAuthCheck(apiFileList, config.Secret))    // 获取历史文件
	router.POST("/index/hook/:method", apiWebHooks)
	logrus.Fatal(http.ListenAndServe(config.API, router))
}
