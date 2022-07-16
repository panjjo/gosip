package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/panjjo/gosip/sip"
	"github.com/panjjo/gosip/utils"
	"github.com/sirupsen/logrus"
)

// 获取录像文件列表
func sipRecordList(to NVRDevices, start, end int64) error {
	hb := sip.NewHeaderBuilder().SetTo(to.addr).SetFrom(_serverDevices.addr).AddVia(&sip.ViaHop{
		Params: sip.NewParams().Add("branch", sip.String{Str: sip.GenerateBranch()}),
	}).SetContentType(&sip.ContentTypeXML).SetMethod(sip.MESSAGE)
	req := sip.NewRequest("", sip.MESSAGE, to.addr.URI, sip.DefaultSipVersion, hb.Build(), sip.GetRecordInfoXML(to.DeviceID, start, end))
	req.SetDestination(to.source)
	tx, err := srv.Request(req)
	if err != nil {
		return err
	}
	response := tx.GetResponse()
	if response.StatusCode() != http.StatusOK {
		return errors.New(response.Reason())
	}
	return nil
}

// MessageRecordInfoResponse 目录列表
type MessageRecordInfoResponse struct {
	CmdType  string       `xml:"CmdType"`
	SN       int          `xml:"SN"`
	DeviceID string       `xml:"DeviceID"`
	SumNum   int          `xml:"SumNum"`
	Item     []RecordItem `xml:"RecordList>Item"`
}

// RecordItem 目录详情
type RecordItem struct {
	// DeviceID 设备编号
	DeviceID string `xml:"DeviceID" bson:"DeviceID" json:"DeviceID"`
	// Name 设备名称
	Name      string `xml:"Name" bson:"Name" json:"Name"`
	FilePath  string `xml:"FilePath" bson:"FilePath" json:"FilePath"`
	Address   string `xml:"Address" bson:"Address" json:"Address"`
	StartTime string `xml:"StartTime" bson:"StartTime" json:"StartTime"`
	EndTime   string `xml:"EndTime" bson:"EndTime" json:"EndTime"`
	Secrecy   int    `xml:"Secrecy" bson:"Secrecy" json:"Secrecy"`
	Type      string `xml:"Type" bson:"Type" json:"Type"`
}

type recordList struct {
	deviceid string
	resp     chan interface{}
	num      int
	data     [][]int64
	l        *sync.Mutex
	s, e     int64
}

// 当前获取目录文件设备集合
var _recordList *sync.Map

func sipMessageRecordInfo(u NVRDevices, body []byte) error {
	message := &MessageRecordInfoResponse{}
	if err := utils.XMLDecode(body, message); err != nil {
		logrus.Errorln("Message Unmarshal xml err:", err, "body:", string(body))
		return err
	}
	if list, ok := _recordList.Load(message.DeviceID); ok {
		info := list.(recordList)
		info.l.Lock()
		defer info.l.Unlock()
		info.num += len(message.Item)
		var sint, eint int64
		for _, item := range message.Item {
			s, _ := time.ParseInLocation("2006-01-02T15:04:05", item.StartTime, time.Local)
			e, _ := time.ParseInLocation("2006-01-02T15:04:05", item.EndTime, time.Local)
			sint = s.Unix()
			eint = e.Unix()
			if sint < info.s {
				sint = info.s
			}
			if eint > info.e {
				eint = info.e
			}
			info.data = append(info.data, []int64{sint, eint})
		}
		if info.num == message.SumNum {
			// 获取到完整数据
			info.resp <- transRecordList(info.data)
		}
		_recordList.Store(message.DeviceID, info)
		return nil
	}
	return errors.New("recordlist devices not found")
}

// RecordResponse RecordResponse
type RecordResponse struct {
	DayTotal int           `json:"daynum"`
	TimeNum  int           `json:"timenum"`
	Data     []interface{} `json:"list"`
}

// RecordInfo RecordInfo
type RecordInfo struct {
	Start int64 `json:"start" bson:"start"`
	End   int64 `json:"end" bson:"end"`
}

// 将返回的多组数据合并，时间连续的进行合并，最后按照天返回数据，返回为某天内时间段列表
func transRecordList(data [][]int64) RecordResponse {
	if len(data) == 0 {
		return RecordResponse{}
	}
	res := RecordResponse{}
	list := map[string][]RecordInfo{}
	sort.Slice(data, func(i, j int) bool {
		return data[i][0] < data[j][0]
	})
	newData := [][]int64{}
	var newDataIE = []int64{}

	for x, d := range data {
		if x == 0 {
			newDataIE = d
			continue
		}
		if d[0] == newDataIE[1] {
			newDataIE[1] = d[1]
		} else {
			newData = append(newData, newDataIE)
			newDataIE = d
		}
	}
	newData = append(newData, newDataIE)
	var cs, ce time.Time
	dates := []string{}
	for _, d := range newData {
		s := time.Unix(d[0], 0)
		e := time.Unix(d[1], 0)
		cs, _ = time.ParseInLocation("20060102", s.Format("20060102"), time.Local)
		for {
			ce = cs.Add(24 * time.Hour)
			if e.Unix() >= ce.Unix() {
				// 当前时段跨天
				if v, ok := list[cs.Format("2006-01-02")]; ok {
					list[cs.Format("2006-01-02")] = append(v, RecordInfo{
						Start: utils.Max(s.Unix(), cs.Unix()),
						End:   ce.Unix() - 1,
					})
				} else {
					list[cs.Format("2006-01-02")] = []RecordInfo{
						{
							Start: utils.Max(s.Unix(), cs.Unix()),
							End:   ce.Unix() - 1,
						},
					}
					dates = append(dates, cs.Format("2006-01-02"))
					res.DayTotal++
				}
				res.TimeNum++
				cs = ce
			} else {
				if v, ok := list[cs.Format("2006-01-02")]; ok {
					list[cs.Format("2006-01-02")] = append(v, RecordInfo{
						Start: utils.Max(s.Unix(), cs.Unix()),
						End:   e.Unix(),
					})
				} else {
					list[cs.Format("2006-01-02")] = []RecordInfo{
						{
							Start: utils.Max(s.Unix(), cs.Unix()),
							End:   e.Unix(),
						},
					}
					dates = append(dates, cs.Format("2006-01-02"))
					res.DayTotal++
				}
				res.TimeNum++
				break
			}
		}
	}
	resData := []interface{}{}
	for _, date := range dates {
		resData = append(resData, map[string]interface{}{
			"date":  date,
			"items": list[date],
		})

	}
	res.Data = resData
	return res
}

func clearRecordFile() {
	var files []RecordFiles
	var ids []string
	for {
		files = []RecordFiles{}
		ids = []string{}
		dbClient.Find(fileTB, M{"end": M{"$lt": time.Now().Unix() - int64(config.Record.Expire)*86400}, "clear": false}, 0, 100, "start", false, &files)
		for _, file := range files {
			filename := filepath.Join(config.Record.FilePath, file.File)
			if _, err := os.Stat(filename); err == nil {
				os.Remove(filename)
			}
			ids = append(ids, file.ID)
		}
		if len(ids) > 0 {
			dbClient.UpdateMany(fileTB, M{"id": M{"$in": ids}}, M{"$set": M{"clear": true}})
		}
		if len(files) != 100 {
			break
		}
	}
}
