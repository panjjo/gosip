package main

import "net/http"

const (
	statusSucc      = "0"
	statusAuthERR   = "1000"
	statusDBERR     = "1001"
	statusParamsERR = "1002"
	statusSysERR    = "1003"

	streamTypePull  = "pull"
	streamTypePush  = "push"
	streamTypeProxy = "proxy"
)

var code2code = map[string]int{
	statusSucc:      http.StatusOK,
	statusDBERR:     http.StatusInternalServerError,
	statusParamsERR: http.StatusBadRequest,
	statusAuthERR:   http.StatusUnauthorized,
	statusSysERR:    http.StatusRequestTimeout,
}
