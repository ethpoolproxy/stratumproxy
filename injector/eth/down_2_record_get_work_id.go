package eth

import (
	"stratumproxy/connection"
	"stratumproxy/protocol/eth"
	"strings"
)

// DownInjectorRecordGetWork 记录 getWork
// 记录对方的 id 并且设置 Flag
func DownInjectorRecordGetWork(payload *connection.InjectorDownstreamPayload) {
	if !strings.Contains(string(payload.In), "eth_getWork") {
		return
	}

	var getWork eth.RequestGetWork
	err := getWork.Parse(payload.In)
	if err != nil {
		return
	}
	err = getWork.Valid()
	if err != nil {
		return
	}

	// 不执行其他的了
	payload.IsTerminated = true
}
