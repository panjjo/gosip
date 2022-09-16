package sipapi

import (
	"encoding/xml"
	"fmt"
	"net"
	"time"

	"github.com/panjjo/gosip/db"
	"github.com/panjjo/gosip/m"
	sip "github.com/panjjo/gosip/sip/s"
	"github.com/panjjo/gosip/utils"
	"github.com/sirupsen/logrus"
)

var (
	// sip服务用户信息
	_serverDevices Devices
	srv            *sip.Server
)

// Devices NVR  设备信息
type Devices struct {
	db.DBModel
	// Name 设备名称
	Name string `json:"name"`
	// DeviceID 设备id
	DeviceID string `json:"deviceid" gorm:"primaryKey;autoIncrement:false"`
	// Region 设备域
	Region string `json:"region" `
	// Host Via 地址
	Host string `json:"host"`
	// Port via 端口
	Port string `json:"port" `
	// TransPort via transport
	TransPort string `json:"transport"`
	// Proto 协议
	Proto string `json:"proto"`
	// Rport via rport
	Rport string `json:"report"`
	// RAddr via recevied
	RAddr string `json:"raddr" `
	// Manufacturer 制造厂商
	Manufacturer string `xml:"Manufacturer"  json:"manufacturer"`
	// 设备类型DVR，NVR
	DeviceType string `xml:"DeviceType"  json:"devicetype"`
	// Firmware 固件版本
	Firmware string ` json:"firmware"`
	// Model 型号
	Model  string `json:"model"`
	URIStr string `json:"uri" `
	// ActiveAt 最后心跳检测时间
	ActiveAt int64 `json:"active"`
	// Regist 是否注册
	Regist bool `json:"regist" `
	// PWD 密码
	PWD string `json:"pwd"`
	// Source
	Source string `json:"source" `

	Sys m.SysInfo `json:"sysinfo" gorm:"-"`

	//----
	addr   *sip.Address `gorm:"-"`
	source net.Addr     `gorm:"-"`
}

// Channels 摄像头通道信息
type Channels struct {
	db.DBModel
	// ChannelID 通道编码
	ChannelID string `xml:"DeviceID" json:"channelid" gorm:"primaryKey;autoIncrement:false"`
	// DeviceID 设备编号
	DeviceID string `xml:"-" json:"deviceid"`
	// Memo 备注（用来标示通道信息）
	MeMo string `json:"memo"`
	// Name 通道名称（摄像头设置名称）
	Name         string `xml:"Name" bson:"name" json:"name"`
	Manufacturer string `xml:"Manufacturer" bson:"manufacturer" json:"manufacturer"`
	Model        string `xml:"Model" bson:"model" json:"model"`
	Owner        string `xml:"Owner" bson:"owner" json:"owner"`
	CivilCode    string `xml:"CivilCode" bson:"civilcode" json:"civilcode"`
	// Address ip地址
	Address     string `xml:"Address" bson:"address" json:"address"`
	Parental    int    `xml:"Parental" bson:"parental" json:"parental"`
	SafetyWay   int    `xml:"SafetyWay" bson:"safetyway" json:"safetyway"`
	RegisterWay int    `xml:"RegisterWay" bson:"registerway" json:"registerway"`
	Secrecy     int    `xml:"Secrecy" bson:"secrecy" json:"secrecy"`
	// Status 状态  on 在线
	Status string `xml:"Status" bson:"status" json:"status"`
	// Active 最后活跃时间
	Active int64  `bson:"active" json:"active"`
	URIStr string `bson:"uri" json:"uri"`

	// 视频编码格式
	VF string `bson:"vf" json:"vf"`
	// 视频高
	Height int `json:"height"`
	// 视频宽
	Width int `json:"width"`
	// 视频FPS
	FPS int `json:"fps"`

	addr *sip.Address `gorm:"-"`
}

// 同步摄像头编码格式
func SyncDevicesCodec(ssrc, deviceid string) {
	resp := zlmGetMediaList(zlmGetMediaListReq{streamID: ssrc})
	if resp.Code != 0 {
		logrus.Errorln("syncDevicesCodec fail", ssrc, resp)
		return
	}
	if len(resp.Data) == 0 {
		logrus.Errorln("syncDevicesCodec fail", ssrc, "not found data", resp)
		return
	}
	for _, data := range resp.Data {
		if len(data.Tracks) == 0 {
			logrus.Errorln("syncDevicesCodec fail", ssrc, "not found tracks", resp)
		}

		for _, track := range data.Tracks {
			if track.Type == 0 {
				// 视频
				device := Channels{DeviceID: deviceid}
				if err := db.Get(db.DBClient, &device); err == nil {
					device.VF = transZLMDeviceVF(track.CodecID)
					device.Height = track.Height
					device.Width = track.Width
					device.FPS = track.FPS
					db.Save(db.DBClient, &device)
				} else {
					logrus.Errorln("syncDevicesCodec deviceid not found,deviceid:", deviceid)
				}
			}
		}
	}
}

// 从请求中解析出设备信息
func parserDevicesFromReqeust(req *sip.Request) (Devices, bool) {
	u := Devices{}
	header, ok := req.From()
	if !ok {
		logrus.Warningln("not found from header from request", req.String())
		return u, false
	}
	if header.Address == nil {
		logrus.Warningln("not found from user from request", req.String())
		return u, false
	}
	if header.Address.User() == nil {
		logrus.Warningln("not found from user from request", req.String())
		return u, false
	}
	u.DeviceID = header.Address.User().String()
	u.Region = header.Address.Host()
	via, ok := req.ViaHop()
	if !ok {
		logrus.Info("not found ViaHop from request", req.String())
		return u, false
	}
	u.Host = via.Host
	u.Port = via.Port.String()
	report, ok := via.Params.Get("rport")
	if ok && report != nil {
		u.Rport = report.String()
	}
	raddr, ok := via.Params.Get("received")
	if ok && raddr != nil {
		u.RAddr = raddr.String()
	}

	u.TransPort = via.Transport
	u.URIStr = header.Address.String()
	u.addr = sip.NewAddressFromFromHeader(header)
	u.Source = req.Source().String()
	u.source = req.Source()
	return u, true
}

// 获取设备信息（注册设备）
func sipDeviceInfo(to Devices) {
	hb := sip.NewHeaderBuilder().SetTo(to.addr).SetFrom(_serverDevices.addr).AddVia(&sip.ViaHop{
		Params: sip.NewParams().Add("branch", sip.String{Str: sip.GenerateBranch()}),
	}).SetContentType(&sip.ContentTypeXML).SetMethod(sip.MESSAGE)
	req := sip.NewRequest("", sip.MESSAGE, to.addr.URI, sip.DefaultSipVersion, hb.Build(), sip.GetDeviceInfoXML(to.DeviceID))
	req.SetDestination(to.source)
	tx, err := srv.Request(req)
	if err != nil {
		logrus.Warnln("sipDeviceInfo  error,", err)
		return
	}
	_, err = sipResponse(tx)
	if err != nil {
		logrus.Warnln("sipDeviceInfo  response error,", err)
		return
	}
}

// sipCatalog 获取注册设备包含的列表
func sipCatalog(to Devices) {
	hb := sip.NewHeaderBuilder().SetTo(to.addr).SetFrom(_serverDevices.addr).AddVia(&sip.ViaHop{
		Params: sip.NewParams().Add("branch", sip.String{Str: sip.GenerateBranch()}),
	}).SetContentType(&sip.ContentTypeXML).SetMethod(sip.MESSAGE)
	req := sip.NewRequest("", sip.MESSAGE, to.addr.URI, sip.DefaultSipVersion, hb.Build(), sip.GetCatalogXML(to.DeviceID))
	req.SetDestination(to.source)
	tx, err := srv.Request(req)
	if err != nil {
		logrus.Warnln("sipCatalog  error,", err)
		return
	}
	_, err = sipResponse(tx)
	if err != nil {
		logrus.Warnln("sipCatalog  response error,", err)
		return
	}
}

// MessageDeviceInfoResponse 主设备明细返回结构
type MessageDeviceInfoResponse struct {
	CmdType      string `xml:"CmdType"`
	SN           int    `xml:"SN"`
	DeviceID     string `xml:"DeviceID"`
	DeviceType   string `xml:"DeviceType"`
	Manufacturer string `xml:"Manufacturer"`
	Model        string `xml:"Model"`
	Firmware     string `xml:"Firmware"`
}

func sipMessageDeviceInfo(u Devices, body []byte) error {
	message := &MessageDeviceInfoResponse{}
	if err := utils.XMLDecode([]byte(body), message); err != nil {
		logrus.Errorln("sipMessageDeviceInfo Unmarshal xml err:", err, "body:", body)
		return err
	}
	db.UpdateAll(db.DBClient, new(Devices), db.M{"device_id=?": u.DeviceID}, Devices{
		Model:        message.Model,
		DeviceType:   message.DeviceType,
		Firmware:     message.Firmware,
		Manufacturer: message.Manufacturer,
	})
	return nil
}

// MessageDeviceListResponse 设备明细列表返回结构
type MessageDeviceListResponse struct {
	XMLName  xml.Name   `xml:"Response"`
	CmdType  string     `xml:"CmdType"`
	SN       int        `xml:"SN"`
	DeviceID string     `xml:"DeviceID"`
	SumNum   int        `xml:"SumNum"`
	Item     []Channels `xml:"DeviceList>Item"`
}

func sipMessageCatalog(u Devices, body []byte) error {
	message := &MessageDeviceListResponse{}
	if err := utils.XMLDecode(body, message); err != nil {
		logrus.Errorln("Message Unmarshal xml err:", err, "body:", string(body))
		return err
	}
	if message.SumNum > 0 {
		for _, d := range message.Item {
			channel := Channels{ChannelID: d.ChannelID, DeviceID: message.DeviceID}
			if err := db.Get(db.DBClient, &channel); err == nil {
				d.DeviceID = channel.DeviceID
				d.Active = time.Now().Unix()
				d.URIStr = fmt.Sprintf("sip:%s@%s", d.ChannelID, _sysinfo.Region)
				d.Status = transDeviceStatus(d.Status)
				d.ID = channel.ID
				d.MeMo = channel.MeMo
				db.Save(db.DBClient, &d)
				go notify(notifyChannelsActive(d))
			} else {
				logrus.Infoln("deviceid not found,deviceid:", d.DeviceID, "pdid:", message.DeviceID, "err", err)
			}
		}
	}
	return nil
}

var deviceStatusMap = map[string]string{
	"ON":     "ON",
	"OK":     "ON",
	"ONLINE": "ON",
	"OFFILE": "OFF",
	"OFF":    "OFF",
}

func transDeviceStatus(status string) string {
	if v, ok := deviceStatusMap[status]; ok {
		return v
	}
	return status
}
