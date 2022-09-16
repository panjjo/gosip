package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// Restful API sign 鉴权
func Auth(c *gin.Context) {
	fmt.Println(c.Request.URL.Path)
	if strings.Contains(c.Request.URL.Path, "/zlm/webhook") {

	}
	// TODO: sign auth
	c.Next()
}
