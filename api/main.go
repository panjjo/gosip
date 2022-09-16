package api

import (
	"github.com/gin-gonic/gin"
	api "github.com/panjjo/gosip/api/c"
	"github.com/panjjo/gosip/api/middleware"
)

func Init(r *gin.Engine) {
	// 中间件
	r.Use(middleware.Auth)
	r.Use(middleware.CORS())

	// 设备类接口
	{
		r.GET("/devices", api.DevicesList)
		r.POST("/devices", api.DevicesCreate)
	}
	// 通道类接口
	{
		r.POST("/devices/:id/channels", api.ChannelCreate)
	}
	// 播放类接口
	{
		r.POST("/channels/:id/streams", api.Play)
		r.DELETE("/streams/:id", api.Stop)

	}
	// zlm webhook
	{
		r.POST("/zlm/webhook/:method", api.ZLMWebHook)
	}
}
