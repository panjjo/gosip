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

func sipMessageKeepalive(u NVRDevices, body []byte) error {
	message := &MessageNotify{}
	if err := utils.XMLDecode(body, message); err != nil {
		logrus.Errorln("Message Unmarshal xml err:", err, "body:", string(body))
		return err
	}
	update := M{"host": u.Host, "port": u.Port, "report": u.Rport, "raddr": u.RAddr, "source": u.Source}
	if message.Status == "OK" {
		update["active"] = time.Now().Unix()
		_activeDevices.Store(u.DeviceID, u)
	} else {
		update["active"] = -1
		_activeDevices.Delete(u.DeviceID)
	}
	go notify(notifyUserAcitve(u.DeviceID, message.Status))
	return dbClient.Update(userTB, M{"deviceid": u.DeviceID}, M{"$set": update})
}
