package m

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	StatusSucc      = "0"
	StatusAuthERR   = "1000"
	StatusDBERR     = "1001"
	StatusParamsERR = "1002"
	StatusSysERR    = "1003"

	StreamTypePull  = "pull"
	StreamTypePush  = "push"
	StreamTypeProxy = "proxy"
)

var CC = map[string]int{
	StatusSucc:      http.StatusOK,
	StatusDBERR:     http.StatusServiceUnavailable,
	StatusParamsERR: http.StatusBadRequest,
	StatusAuthERR:   http.StatusUnauthorized,
	StatusSysERR:    http.StatusInternalServerError,
}

type Response struct {
	Data  interface{} `json:"data"`
	MsgID string      `json:"msgid"`
	Code  string      `json:"code"`
}

func JsonResponse(c *gin.Context, code string, data interface{}) {
	c.JSON(CC[code], Response{MsgID: c.GetString("msgid"), Code: code, Data: data})
}
