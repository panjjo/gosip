package sipapi

import (
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/panjjo/gosip/db"
	"github.com/panjjo/gosip/m"
	"github.com/panjjo/gosip/utils"
)

type apiRecordList struct {
	items map[string]*apiRecordItem
	l     sync.RWMutex
}

func (rl *apiRecordList) Get(id string) (*apiRecordItem, bool) {
	rl.l.RLock()
	defer rl.l.RUnlock()
	res, ok := rl.items[id]
	return res, ok
}

func (rl *apiRecordList) Start(id string, values url.Values) *apiRecordItem {
	item := &apiRecordItem{resp: make(chan string, 1), clos: make(chan bool, 1), params: values, id: utils.RandString(32)}
	rl.l.Lock()
	rl.items[id] = item
	rl.l.Unlock()
	return item
}
func (rl *apiRecordList) Stop(id string) {
	rl.l.Lock()
	delete(rl.items, id)
	rl.l.Unlock()
}

// key:ssrc  value=channel
// 记录流录制
var RecordList apiRecordList

type apiRecordItem struct {
	resp   chan string
	clos   chan bool
	params url.Values
	req    url.Values
	id     string
}

func (ri *apiRecordItem) Start() (string, interface{}) {
	err := zlmStartRecord(ri.params)
	if err != nil {
		return m.StatusParamsERR, err
	}
	if config.Record.Recordmax != -1 {
		go func() {
			tick := time.NewTicker(time.Duration(config.Record.Recordmax) * time.Second)
			select {
			case <-tick.C:
				// 自动停止录制
				ri.Stop()
				url := <-ri.resp
				notify(notifyRecordStop(url, ri.req))
			case <-ri.clos:
				// 调用stop接口
			}
		}()
	}
	err = db.Create(db.DBClient, Files{
		FID:    ri.id,
		Stream: ri.params.Get("stream"),
		params: ri.params,
		Start:  time.Now().Unix(),
	})
	if err != nil {
		return m.StatusDBERR, err
	}
	return m.StatusSucc, ri.id
}
func (ri *apiRecordItem) Stop() (string, interface{}) {
	err := zlmStopRecord(ri.params)
	if err != nil {
		return m.StatusSysERR, ""
	}
	return m.StatusSucc, ""
}

func (ri *apiRecordItem) Down(url string) {
	db.UpdateAll(db.DBClient, new(Files), db.M{"id=?": ri.id}, db.M{"end": time.Now().Unix(), "status": 1, "file": url})
}

func (ri *apiRecordItem) Resp(data string) {
	ri.resp <- data
}

// Files Files
type Files struct {
	db.DBModel
	Start  int64  `json:"start" bson:"start"`
	End    int64  `json:"end" bson:"end"`
	Stream string `json:"stream" bson:"stream"`
	FID    string `json:"fid" bson:"fid"`
	Status int    `json:"status" bson:"status"`
	File   string `json:"file" bson:"file"`
	Clear  bool   `json:"clear" bson:"clear"`
	params url.Values
}

func ClearFiles() {
	var files []Files
	var ids []string
	for {
		files = []Files{}
		ids = []string{}
		db.FindT(db.DBClient, new(Files), &files, db.M{"end < ?": time.Now().Unix() - int64(config.Record.Expire)*86400, "clear=?": false}, "", 0, 100, false)
		for _, file := range files {
			filename := filepath.Join(config.Record.FilePath, file.File)
			if _, err := os.Stat(filename); err == nil {
				os.Remove(filename)
			}
			ids = append(ids, file.FID)
		}
		if len(ids) > 0 {
			db.UpdateAll(db.DBClient, new(Files), db.M{"id in (?)": ids}, db.M{"clear": true})
		}
		if len(files) != 100 {
			break
		}
	}
}
