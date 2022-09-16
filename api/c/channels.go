package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/panjjo/gorm"
	"github.com/panjjo/gosip/db"
	"github.com/panjjo/gosip/m"
	sipapi "github.com/panjjo/gosip/sip"
)

// @Summary     设备新增接口
// @Description 通过此接口在设备下新增通道，获取通道id
// @Tags        channels
// @Accept      x-www-form-urlencoded
// @Produce     json
// @Param       id   path     string true  "设备id"
// @Param       memo formData string false "通道备注"
// @Success     0    {object} sipapi.Devices
// @Failure     1000 {object} string
// @Failure     1001 {object} string
// @Failure     1002 {object} string
// @Failure     1003 {object} string
// @Router      /devices/{id}/channels [post]
func ChannelCreate(c *gin.Context) {

	id := c.Param("id")
	if id == "" {
		m.JsonResponse(c, m.StatusParamsERR, "缺少设备ID")
		return
	}
	device := sipapi.Devices{DeviceID: id}
	if err := db.Get(db.DBClient, &device); err != nil {
		if db.RecordNotFound(err) {
			m.JsonResponse(c, m.StatusParamsERR, "用户设备不存在")
			return
		}
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}

	channel := sipapi.Channels{
		ChannelID: fmt.Sprintf("%s%06d", m.MConfig.GB28181.CID, m.MConfig.GB28181.CNUM+1),
		DeviceID:  device.DeviceID,
		MeMo:      c.PostForm("memo"),
	}
	tx, err := db.NewTx(db.DBClient)
	if err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	defer tx.End()
	if err := db.Create(tx.DB(), &channel); err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	if _, err := db.UpdateAll(tx.DB(), new(m.SysInfo), db.M{}, db.M{"cnum": gorm.Expr("cnum+1")}); err != nil {
		m.JsonResponse(c, m.StatusDBERR, err)
		return
	}
	tx.Commit()
	m.MConfig.GB28181.CNUM += 1

	m.JsonResponse(c, m.StatusSucc, channel)
}
