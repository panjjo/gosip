package main

import (
	"time"

	"github.com/panjjo/gosip/utils"
	"github.com/sirupsen/logrus"
)

// MessageNotify 心跳包xml结构
type MessageNotify struct {
	CmdType  string `xml:"CmdType"`
	SN       int    `xml:"SN"`
	DeviceID string `xml:"DeviceID"`
	Status   string `xml:"Status"`
	Info     string `xml:"Info"`
}

func sipMessageKeepalive(u NVRDevices, body string) error {
	message := &MessageNotify{}
	if err := utils.XMLDecode([]byte(body), message); err != nil {
		logrus.Errorln("Message Unmarshal xml err:", err, "body:", body)
		return err
	}
	update := M{}
	if message.Status == "OK" {
		update["active"] = time.Now().Unix()
		_activeDevices.Store(u.DeviceID, u)
	} else {
		update["active"] = -1
		_activeDevices.Delete(u.DeviceID)
	}
	return dbClient.Update(userTB, M{"deviceid": u.DeviceID}, M{"$set": update})
}
