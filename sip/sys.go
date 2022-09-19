package sipapi

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/panjjo/gosip/db"
	"github.com/panjjo/gosip/m"
	sip "github.com/panjjo/gosip/sip/s"
	"github.com/panjjo/gosip/utils"
	"github.com/sirupsen/logrus"
)

func Start() {
	// 数据库表初始化 启动时自动同步数据结构到数据库
	db.DBClient.AutoMigrate(new(Devices))
	db.DBClient.AutoMigrate(new(Channels))
	db.DBClient.AutoMigrate(new(Streams))
	db.DBClient.AutoMigrate(new(m.SysInfo))
	db.DBClient.AutoMigrate(new(Files))

	LoadSYSInfo()

	srv = sip.NewServer()
	srv.RegistHandler(sip.REGISTER, handlerRegister)
	srv.RegistHandler(sip.MESSAGE, handlerMessage)
	go srv.ListenUDPServer(config.UDP)
}

// MODDEBUG MODDEBUG
var MODDEBUG = "DEBUG"

// ActiveDevices 记录当前活跃设备，请求播放时设备必须处于活跃状态
type ActiveDevices struct {
	sync.Map
}

// Get Get
func (a *ActiveDevices) Get(key string) (Devices, bool) {
	if v, ok := a.Load(key); ok {
		return v.(Devices), ok
	}
	return Devices{}, false
}

var _activeDevices ActiveDevices

// 系统运行信息
var _sysinfo *m.SysInfo
var config *m.Config

func LoadSYSInfo() {

	config = m.MConfig
	_activeDevices = ActiveDevices{sync.Map{}}

	StreamList = streamsList{&sync.Map{}, &sync.Map{}, 0}
	ssrcLock = &sync.Mutex{}
	_recordList = &sync.Map{}
	RecordList = apiRecordList{items: map[string]*apiRecordItem{}, l: sync.RWMutex{}}

	// init sysinfo
	_sysinfo = &m.SysInfo{}
	if err := db.Get(db.DBClient, _sysinfo); err != nil {
		if db.RecordNotFound(err) {
			//  初始不存在
			_sysinfo = m.DefaultInfo()

			if err = db.Create(db.DBClient, _sysinfo); err != nil {
				logrus.Fatalf("1 init sysinfo err:%v", err)
			}
		} else {
			logrus.Fatalf("2 init sysinfo err:%v", err)
		}
	}
	m.MConfig.GB28181 = _sysinfo

	uri, _ := sip.ParseSipURI(fmt.Sprintf("sip:%s@%s", _sysinfo.LID, _sysinfo.Region))
	_serverDevices = Devices{
		DeviceID: _sysinfo.LID,
		Region:   _sysinfo.Region,
		addr: &sip.Address{
			DisplayName: sip.String{Str: "sipserver"},
			URI:         &uri,
			Params:      sip.NewParams(),
		},
	}

	// init media
	url, err := url.Parse(config.Media.RTP)
	if err != nil {
		logrus.Fatalf("media rtp url error,url:%s,err:%v", config.Media.RTP, err)
	}
	ipaddr, err := net.ResolveIPAddr("ip", url.Hostname())
	if err != nil {
		logrus.Fatalf("media rtp url error,url:%s,err:%v", config.Media.RTP, err)
	}
	_sysinfo.MediaServerRtpIP = ipaddr.IP
	_sysinfo.MediaServerRtpPort, _ = strconv.Atoi(url.Port())
}

// zlm接收到的ssrc为16进制。发起请求的ssrc为10进制
func ssrc2stream(ssrc string) string {
	if ssrc[0:1] == "0" {
		ssrc = ssrc[1:]
	}
	num, _ := strconv.Atoi(ssrc)
	return fmt.Sprintf("%08X", num)
}

func sipResponse(tx *sip.Transaction) (*sip.Response, error) {
	response := tx.GetResponse()
	if response == nil {
		return nil, utils.NewError(nil, "response timeout", "tx key:", tx.Key())
	}
	if response.StatusCode() != http.StatusOK {
		return response, utils.NewError(nil, "response fail", response.StatusCode(), response.Reason(), "tx key:", tx.Key())
	}
	return response, nil
}
