package sipapi

import (
	"fmt"
	"math/rand"
	"time"
)

/**
 * 云台指令码计算 包括PTZ指令、FI指令、预置位指令、巡航指令、扫描指令和辅助开关指令
 *
 * @param cmdCode      指令码
 * @param parameter1   数据1
 * @param parameter2   数据2
 * @param combineCode2 组合码2
 */
func CmdString(cmdCode, parameter1, parameter2, combineCode2 int) string {
	builder := "A50F01"
	strTmp := fmt.Sprintf("%02X", cmdCode)
	builder += strTmp[0:2]
	strTmp = fmt.Sprintf("%02X", parameter1)
	builder += strTmp[0:2]
	strTmp = fmt.Sprintf("%02X", parameter2)
	builder += strTmp[0:2]
	strTmp = fmt.Sprintf("%X", combineCode2)
	builder += strTmp[0:1]
	builder += "0"

	//计算校验码
	checkCode := (0xA5 + 0x0F + 0x01 + cmdCode + parameter1 + parameter2 + (combineCode2 & 0xF0)) % 0x100
	strTmp = fmt.Sprintf("%02X", checkCode)
	builder += strTmp[0:2]
	return builder
}

func RandNumString(n int) string {
	numbers := "0123456789"
	return randStringBySoure(numbers, n)
}

func randStringBySoure(src string, n int) string {
	randomness := make([]byte, n)

	rand.Seed(time.Now().UnixNano())
	_, err := rand.Read(randomness)
	if err != nil {
		panic(err)
	}

	l := len(src)

	// fill output
	output := make([]byte, n)
	for pos := range output {
		random := randomness[pos]
		randomPos := random % uint8(l)
		output[pos] = src[randomPos]
	}

	return string(output)
}
