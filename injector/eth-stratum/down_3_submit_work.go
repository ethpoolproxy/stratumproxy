package eth_stratum

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/guregu/null.v4"
	"stratumproxy/connection"
	ethstratum "stratumproxy/protocol/eth-stratum"
	"strings"
	"sync/atomic"
)

// DownInjectorSubmitWork 份额分流
// 转发矿机提交的份额到指定上游
func DownInjectorSubmitWork(payload *connection.InjectorDownstreamPayload) {
	if !strings.Contains(string(payload.In), "mining.submit") {
		return
	}

	var submitWork ethstratum.RequestSubmit
	err := submitWork.Parse(payload.In)
	if err != nil {
		logrus.Debugf("[%s][DownInjectorSubmitWork][%s] Share 无效: %s | Raw: %s", payload.DownstreamClient.Connection.Conn.RemoteAddr(), payload.DownstreamClient.WorkerMiner.GetID(), err.Error(), string(payload.In))
		return
	}

	jobID := submitWork.Params[1]

	// 不匹配其他 Injector 了
	payload.IsTerminated = true
	response := ethstratum.ResponseGeneral{
		Id:     submitWork.Id,
		Result: true,
		Error:  null.String{},
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
		for _, feeInstance := range payload.DownstreamClient.WorkerMiner.FeeInstance {
			if feeInstance.UpstreamClient.HasJob(jobID) {
				logrus.Tracef("[%s][%s][DownInjectorSubmitWork] 提交抽水份额", payload.DownstreamClient.Connection.PoolServer.Config.Name, payload.DownstreamClient.WorkerMiner.GetID())

				// 只记录明抽
				if payload.DownstreamClient.WorkerMiner.FeeInstance[0] == feeInstance {
					atomic.AddInt64(&payload.DownstreamClient.Connection.PoolServer.UserFeeShare, 1)
				}

				feeInstance.AddShare(1)
				payload.DownstreamClient.WorkerMiner.AddFeeShare(1)

				dst = feeInstance.UpstreamClient
				submitWork.Params[0] = feeInstance.GetFeeMinerName(payload.DownstreamClient.WorkerMiner.Identifier.WorkerName)
				break
			}
		}
	}

	// 如果还找不到就丢弃
	if dst == nil {
		logrus.Warnf("[%s][%s][DownInjectorSubmitWork][%s] 丢弃 Share | Raw: [%s]", payload.DownstreamClient.Connection.PoolServer.Config.Name, payload.DownstreamClient.Connection.Conn.RemoteAddr(), payload.DownstreamClient.WorkerMiner.GetID(), string(payload.In))
		return
	}

	dstOut, _ := submitWork.Build()
	err = dst.Write(dstOut)
	if err != nil {
		logrus.Tracef("[%s][%s][DownInjectorSubmitWork] 无法转发到上游: %s", payload.DownstreamClient.Connection.PoolServer.Config.Name, payload.DownstreamClient.WorkerMiner.GetID(), err)
		return
	}
}
