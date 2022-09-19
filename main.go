package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/gin-gonic/gin"
	"github.com/panjjo/gosip/api"
	"github.com/panjjo/gosip/api/middleware"
	"github.com/panjjo/gosip/m"
	sipapi "github.com/panjjo/gosip/sip"
	"github.com/sirupsen/logrus"

	_ "github.com/panjjo/gosip/docs"

	"github.com/robfig/cron"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title          GoSIP
// @version        2.0
// @description    GB28181 SIP服务端.
// @termsOfService https://github.com/panjjo/gosip

// @contact.name  GoSIP
// @contact.url   https://github.com/panjjo/gosip
// @contact.email panjjo@vip.qq.com

// @license.name Apache 2.0
// @license.url  http://www.apache.org/licenses/LICENSE-2.0.html

// @host     localhost:8090
// @BasePath /

// @securityDefinitions.basic BasicAuth

func main() {
	//pprof
	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil)
	}()

	sipapi.Start()

	r := gin.Default()
	r.Use(middleware.Recovery)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	api.Init(r)

	logrus.Fatal(r.Run(m.MConfig.API))
	// restapi.RestfulAPI()
}

func init() {
	m.LoadConfig()
	_cron()
}

func _cron() {
	c := cron.New()                                 // 新建一个定时任务对象
	c.AddFunc("0 */5 * * * *", sipapi.CheckStreams) // 定时关闭推送流
	c.AddFunc("0 */5 * * * *", sipapi.ClearFiles)   // 定时清理录制文件
	c.Start()
}
