package eth

import (
	"github.com/sirupsen/logrus"
	"stratumproxy/connection"
	"stratumproxy/protocol/eth"
	"strings"
	"sync/atomic"
)

// DownInjectorSubmitWork 份额分流
// 转发矿机提交的份额到指定上游
func DownInjectorSubmitWork(payload *connection.InjectorDownstreamPayload) {
	if !strings.Contains(string(payload.In), "eth_submitWork") {
		return
	}

	var submitWork eth.RequestSubmitWork
	err := submitWork.Parse(payload.In)
	if err != nil {
		logrus.Debugf("[%s][DownInjectorSubmitWork][%s] Share 解析失败: %s | Raw: %s", payload.DownstreamClient.Connection.Conn.RemoteAddr(), payload.DownstreamClient.WorkerMiner.GetID(), err.Error(), string(payload.In))
		return
	}
	err = submitWork.Valid()
	if err != nil {
		logrus.Debugf("[%s][DownInjectorSubmitWork][%s] Share 验证失败: [%s] | Raw: %s", payload.DownstreamClient.Connection.Conn.RemoteAddr(), payload.DownstreamClient.WorkerMiner.GetID(), err.Error(), string(payload.In))
		return
	}

	jobID := submitWork.Params[1]

	// 不匹配其他 Injector 了
	payload.IsTerminated = true
	response := eth.ResponseGeneral{
		Id:     submitWork.Id,
		Result: true,
	}
	out, _ := response.Build()
	payload.Out = out

	// 寻找目标上游
	var dst *connection.UpstreamClient

	// 如果是转发的矿池
	if payload.DownstreamClient.Upstream.HasJob(jobID) {
		dst = payload.DownstreamClient.Upstream
		payload.DownstreamClient.WorkerMiner.AddShare(1)
		atomic.AddInt64(&payload.DownstreamClient.Connection.PoolServer.GlobalShareStats, 1)
	}

	// 抽水的话
	if dst == nil {
		for _, feeInstance := range payload.DownstreamClient.Connection.PoolServer.FeeInstance {
			if feeInstance.UpstreamClient.HasJob(jobID) {
				logrus.Tracef("[%s][%s][DownInjectorSubmitWork] 提交抽水份额", payload.DownstreamClient.Connection.PoolServer.Config.Name, payload.DownstreamClient.WorkerMiner.GetID())

				feeShareObj, _ := payload.DownstreamClient.WorkerMiner.FeeShareIndividual.LoadOrStore(feeInstance, int64(0))
				payload.DownstreamClient.WorkerMiner.FeeShareIndividual.Store(feeInstance, feeShareObj.(int64)+1)
				feeInstance.AddShare(1)
				payload.DownstreamClient.WorkerMiner.AddFeeShare(1)

				dst = feeInstance.UpstreamClient
				submitWork.Worker = feeInstance.NamePrefix + submitWork.Worker
				break
			}
		}
	}

	// 如果还找不到就丢弃
	if dst == nil {
		logrus.Debugf("[%s][%s][DownInjectorSubmitWork][%s] 丢弃 Share | Raw: [%s]", payload.DownstreamClient.Connection.PoolServer.Config.Name, payload.DownstreamClient.Connection.Conn.RemoteAddr(), payload.DownstreamClient.WorkerMiner.GetID(), string(payload.In))
		return
	}

	dstOut, _ := submitWork.Build()
	err = dst.Write(dstOut)
	if err != nil {
		payload.DownstreamClient.Upstream.Reconnect()
		return
	}
}
