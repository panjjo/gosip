package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/panjjo/gosip/utils"
	"github.com/sirupsen/logrus"
)

const (
	// NotifyMethodUserActive 用户活跃状态通知
	NotifyMethodUserActive = "users.active"
	// NotifyMethodUserRegister 用户注册通知
	NotifyMethodUserRegister = "users.regiester"
	// NotifyMethodDeviceActive 设备活跃通知
	NotifyMethodDeviceActive = "devices.active"
	// NotifyMethodRecordStop 视频录制结束
	NotifyMethodRecordStop = "records.stop"
)

// Notify 消息通知结构
type Notify struct {
	Method string      `json:"method"`
	Data   interface{} `json:"data"`
}

func notify(data *Notify) {
	if url, ok := config.notifyMap[data.Method]; ok {
		res, err := utils.PostJSONRequest(url, data)
		if err != nil {
			logrus.Warningln(data.Method, "send notify fail.", err)
		}
		if strings.ToUpper(string(res)) != "OK" {
			logrus.Warningln(data.Method, "send notify resp fail.", string(res), "len:", len(res), config.Notify, data)
		} else {
			logrus.Debug("notify send succ:", data.Method, data.Data)
		}
	} else {
		logrus.Traceln("notify config not found", data.Method)
	}
}

func notifyUserAcitve(id, status string) *Notify {
	return &Notify{
		Method: NotifyMethodUserActive,
		Data: map[string]interface{}{
			"deviceid": id,
			"status":   status,
			"time":     time.Now().Unix(),
		},
	}
}
func notifyUserRegister(u NVRDevices) *Notify {
	u.Sys = _sysinfo
	return &Notify{
		Method: NotifyMethodUserRegister,
		Data:   u,
	}
}

func notifyDeviceActive(d Devices) *Notify {
	return &Notify{
		Method: NotifyMethodDeviceActive,
		Data: map[string]interface{}{
			"deviceid": d.DeviceID,
			"status":   d.Status,
			"time":     time.Now().Unix(),
		},
	}
}
func notifyRecordStop(url string, req url.Values) *Notify {
	d := map[string]interface{}{
		"url": fmt.Sprintf("%s/%s", config.Media.HTTP, url),
	}
	for k, v := range req {
		d[k] = v[0]
	}
	return &Notify{
		Method: NotifyMethodRecordStop,
		Data:   d,
	}
}
