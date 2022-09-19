package m

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	StatusSucc      = "0"
	StatusAuthERR   = "1000"
	StatusDBERR     = "1001"
	StatusParamsERR = "1002"
	StatusSysERR    = "1003"

	StreamTypePull = "pull"
	StreamTypePush = "push"
)

var CC = map[string]int{
	StatusSucc:      http.StatusOK,
	StatusDBERR:     http.StatusServiceUnavailable,
	StatusParamsERR: http.StatusBadRequest,
	StatusAuthERR:   http.StatusUnauthorized,
	StatusSysERR:    http.StatusInternalServerError,
}

type Response struct {
	Data  any    `json:"data"`
	MsgID string `json:"msgid"`
	Code  string `json:"code"`
}

func JsonResponse(c *gin.Context, code string, data any) {
	switch d := data.(type) {
	case error:
		data = d.Error()
	}
	c.JSON(CC[code], Response{MsgID: c.GetString("msgid"), Code: code, Data: data})
}

const (
	DeviceStatusON  = "ON"
	DeviceStatusOFF = "OFF"
	defaultLimit    = 20
	defaultSort     = "-addtime"
)

func GetLimit(c *gin.Context) int {
	value := c.Query("limit")
	if value == "" {
		return defaultLimit
	}
	if d, e := strconv.Atoi(value); e == nil {
		return d
	} else {
		return defaultLimit
	}
}
func GetSort(c *gin.Context) string {
	value := c.Query("sort")
	if value == "" {
		return defaultSort
	}
	return value
}
func GetSkip(c *gin.Context) int {
	value := c.Query("skip")
	if value == "" {
		return 0
	}
	if d, e := strconv.Atoi(value); e == nil {
		return d
	} else {
		return 0
	}
}
