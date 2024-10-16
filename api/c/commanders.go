package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/panjjo/gosip/m"
	sipapi "github.com/panjjo/gosip/sip"
)

// | 名称      | 形式       | 类型      | 必选  | 描述                                                                                           |
// | ------- | -------- | ------- | --- | -------------------------------------------------------------------------------------------- |
// | id      | path     | string  | 是   | 设备id                                                                                         |
// | command | formData | string  | 是   | 控制指令 允许值: left, right, up, down, upleft, upright, downleft, downright, zoomin, zoomout, stop |
// | channel | formData | string  | 否   | 通道id,默认会选择第一个通道                                                                              |
// | speed   | formData | Integer | 否   | 速度(0~255) 默认值: 129                                                                           |

func PTZCtrl(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		m.JsonResponse(c, m.StatusParamsERR, "缺少设备ID")
		return
	}
	device := sipapi.GetActiveDevice(id)
	if device == nil {
		m.JsonResponse(c, m.StatusDBERR, "该设备不在线")
		return
	}
	command := c.Query("command")
	if command == "" {
		m.JsonResponse(c, m.StatusParamsERR, "缺少控制指令")
		return
	}
	channelId := c.Query("channel")
	if channelId != "" {
		device.DeviceID = channelId
	}
	speedStr := c.Query("speed")
	speed := 50
	if speedStr != "" {
		speed1, err := strconv.Atoi(speedStr)
		if err != nil {
			m.JsonResponse(c, m.StatusParamsERR, "速度参数转换数据类型有误")
			return
		}
		if speed1 < 0 && speed1 > 255 {
			m.JsonResponse(c, m.StatusParamsERR, "速度参数不在合理范围内0~255")
			return
		}
		speed = speed1
	}

	cmdCode := 0
	switch command {
	case "left":
		cmdCode = 2
	case "right":
		cmdCode = 1
	case "up":
		cmdCode = 8
	case "down":
		cmdCode = 4
	case "upleft":
		cmdCode = 10
	case "upright":
		cmdCode = 9
	case "downleft":
		cmdCode = 6
	case "downright":
		cmdCode = 5
	case "zoomin":
		cmdCode = 16
	case "zoomout":
		cmdCode = 32
	case "stop":
		cmdCode = 0
		speed = 0
	default:
		m.JsonResponse(c, m.StatusParamsERR, "未知控制指令")
		return
	}
	cmdStr := sipapi.CmdString(cmdCode, speed, speed, speed)

	err := sipapi.DevicePTZ(*device, cmdStr)
	if err != nil {
		m.JsonResponse(c, m.StatusParamsERR, err)
		return
	}
	m.JsonResponse(c, m.StatusSucc, "")
}
