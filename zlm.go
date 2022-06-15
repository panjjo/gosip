package main

import (
	"fmt"
	"net/url"

	"github.com/panjjo/gosip/utils"
	"github.com/sirupsen/logrus"
)

//
type rtpInfo struct {
	Code  int  `json:"code"`
	Exist bool `json:"exist"`
}

// 获取流在zlm上的信息
func zlmGetMediaInfo(ssrc string) rtpInfo {
	res := rtpInfo{}
	body, err := utils.GetRequest(config.Media.RESTFUL + "/index/api/getRtpInfo?secret=" + config.Media.Secret + "&stream_id=" + ssrc)
	if err != nil {
		logrus.Errorln("get stream rtpInfo fail,", err)
		return res
	}
	if err = utils.JSONDecode(body, &res); err != nil {
		logrus.Errorln("get stream rtpInfo fail,", err)
		return res
	}
	return res
}

// zlm 关闭流
func zlmCloseStream(ssrc string) {
	utils.GetRequest(config.Media.RESTFUL + "/index/api/close_streams?secret=" + config.Media.Secret + "&stream=" + ssrc)
}

// zlm 开始录制视频流
func zlmStartRecord(values url.Values) error {
	body, err := utils.GetRequest(config.Media.RESTFUL + "/index/api/startRecord?" + values.Encode())
	if err != nil {
		return err
	}
	tmp := map[string]interface{}{}
	err = utils.JSONDecode(body, &tmp)
	if err != nil {
		return err
	}
	if code, ok := tmp["code"]; !ok || fmt.Sprint(code) != "0" {
		return utils.NewError(nil, tmp)
	}
	return nil
}

// zlm 停止录制
func zlmStopRecord(values url.Values) error {
	body, err := utils.GetRequest(config.Media.RESTFUL + "/index/api/stopRecord?" + values.Encode())
	if err != nil {
		return err
	}
	tmp := map[string]interface{}{}
	err = utils.JSONDecode(body, &tmp)
	if err != nil {
		return err
	}
	if code, ok := tmp["code"]; !ok || fmt.Sprint(code) != "0" {
		return utils.NewError(nil, tmp)
	}
	return nil
}

// zlm 添加流代理
func zlmAddStreamProxy(url, tag string) (string, error) {
	body, err := utils.PostJSONRequest(config.Media.RESTFUL+"/index/api/addStreamProxy?secret"+config.Media.Secret, map[string]interface{}{
		"secret":     config.Media.Secret,
		"app":        "rtp",
		"stream":     tag,
		"url":        url,
		"vhost":      "__defaultVhost__",
		"enable_hls": 1,
	})
	if err != nil {
		return "", err
	}
	tmp := map[string]interface{}{}
	err = utils.JSONDecode(body, &tmp)
	if err != nil {
		return "", err
	}
	if code, ok := tmp["code"]; !ok || fmt.Sprint(code) != "0" {
		return "", utils.NewError(nil, tmp)
	}
	return tmp["data"].(map[string]interface{})["key"].(string), nil
}
