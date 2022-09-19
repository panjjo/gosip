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
		r.POST("/devices/:id", api.DevicesUpdate)
		r.DELETE("/devices/:id", api.DevicesDelete)

	}
	// 通道类接口
	{
		r.GET("/channels", api.ChannelsList)
		r.POST("/devices/:id/channels", api.ChannelCreate)
		r.POST("/channels/:id", api.ChannelsUpdate)
		r.DELETE("/channels/:id", api.ChannelsDelete)
	}
	// 播放类接口
	{
		r.GET("/streams", api.StreamsList)
		r.POST("/channels/:id/streams", api.Play)
		r.DELETE("/streams/:id", api.Stop)
	}
	// 录像类
	{
		r.GET("/channels/:id/records", api.RecordsList)
	}
	// zlm webhook
	{
		r.POST("/zlm/webhook/:method", api.ZLMWebHook)
	}
}
