package sipapi

import (
	"time"

	"github.com/panjjo/gosip/db"
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

func sipMessageKeepalive(u Devices, body []byte) error {
	message := &MessageNotify{}
	if err := utils.XMLDecode(body, message); err != nil {
		logrus.Errorln("Message Unmarshal xml err:", err, "body:", string(body))
		return err
	}
	device, ok := _activeDevices.Get(u.DeviceID)
	if !ok {
		device = Devices{DeviceID: u.DeviceID}
		if err := db.Get(db.DBClient, &device); err != nil {
			logrus.Warnln("Device Keepalive not found ", u.DeviceID, err)
		}
	}
	if message.Status == "OK" {
		device.ActiveAt = time.Now().Unix()
		_activeDevices.Store(u.DeviceID, u)
	} else {
		device.ActiveAt = -1
		_activeDevices.Delete(u.DeviceID)
	}
	go notify(notifyDevicesAcitve(u.DeviceID, message.Status))
	_, err := db.UpdateAll(db.DBClient, new(Devices), map[string]interface{}{"deviceid=?": u.DeviceID}, Devices{
		Host:     u.Host,
		Port:     u.Port,
		Rport:    u.Rport,
		RAddr:    u.RAddr,
		Source:   u.Source,
		URIStr:   u.URIStr,
		ActiveAt: device.ActiveAt,
	})
	return err
}
