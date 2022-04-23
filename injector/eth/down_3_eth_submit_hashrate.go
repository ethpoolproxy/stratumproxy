package eth

import (
	"github.com/sirupsen/logrus"
	"stratumproxy/connection"
	"stratumproxy/protocol/eth"
	"strings"
)

// DownInjectorEthSubmitHashrate 记录算力
func DownInjectorEthSubmitHashrate(payload *connection.InjectorDownstreamPayload) {
	if !strings.Contains(string(payload.In), "eth_submitHashrate") {
		return
	}

	var hashratePack eth.RequestHashratePack
	err := hashratePack.Parse(payload.In)
	if err != nil {
		logrus.Debugf("[%s][DownInjectorEthSubmitHashrate][%s] Hashrate 解析失败: %s | Raw: %s", payload.DownstreamClient.Connection.Conn.RemoteAddr(), payload.DownstreamClient.WorkerMiner.GetID(), err.Error(), string(payload.In))
		return
	}
	err = hashratePack.Valid()
	if err != nil {
		logrus.Debugf("[%s][DownInjectorEthSubmitHashrate][%s] Hashrate 解析失败: %s | Raw: %s", payload.DownstreamClient.Connection.Conn.RemoteAddr(), payload.DownstreamClient.WorkerMiner.GetID(), err.Error(), string(payload.In))
		return
	}

	// 不匹配其他 Injector 了
	payload.IsTerminated = true
	response := eth.ResponseGeneral{
		Id:     hashratePack.Id,
		Result: true,
	}
	out, _ := response.Build()
	payload.Out = out

	if payload.DownstreamClient.WorkerMiner == nil {
		logrus.Debugf("[%s][%s][InjectorEthSubmitHashrate] 找不到 Miner", payload.DownstreamClient.Connection.PoolServer.Config.Name, payload.DownstreamClient.Connection.Conn.RemoteAddr())
		return
	}

	payload.DownstreamClient.WorkerMiner.HashRate = hashratePack.Hashrate

	logrus.Tracef("[%s][InjectorEthSubmitHashrate] 记录算力: %d MH/s", payload.DownstreamClient.WorkerMiner.GetID(), hashratePack.Hashrate/1000000)

	err = payload.DownstreamClient.Upstream.Write(payload.In)
	if err != nil {
		payload.DownstreamClient.Upstream.Reconnect()
		return
	}
}
