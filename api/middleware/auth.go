package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/panjjo/gosip/utils"
)

// Restful API sign 鉴权
func Auth(c *gin.Context) {
	if c.GetString("msgid") == "" {
		c.Set("msgid", utils.RandString(32))
	}
	if strings.Contains(c.Request.URL.Path, "/zlm/webhook") {
		c.Next()
		return
	}
	// TODO: sign auth
	c.Next()
}
