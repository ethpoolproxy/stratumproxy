package eth

import (
	"github.com/sirupsen/logrus"
	"stratumproxy/connection"
	"stratumproxy/protocol/eth"
	"time"
)

// UpInjectorSendJob 只分发任务下去
func UpInjectorSendJob(payload *connection.InjectorUpstreamPayload) {
	var job eth.ResponseWorkerJob
	err := job.Parse(payload.In)
	if err != nil {
		return
	}
	err = job.Valid()
	if err != nil {
		return
	}

	// 记录任务
	payload.UpstreamClient.AddJob(job.Result[0])

	// 更新最后下发时间
	payload.UpstreamClient.LastJobAt = time.Now()

	// 如果是从抽水矿池发来的
	if payload.UpstreamClient.DownstreamClient == nil {
		// 获取抽水信息
		feeInfo := payload.UpstreamClient.PoolServer.FindFeeInfoByFeeUpstream(payload.UpstreamClient)
		if feeInfo == nil {
			return
		}

		// 获取下游
		downstream, ok := payload.UpstreamClient.PoolServer.WorkerMinerFeeDB.Load(feeInfo)
		if !ok {
			return
		}

		// 群发给要抽水的
		downstream.(*connection.WorkerMinerSliceWrapper).Range(func(i int, m *connection.WorkerMiner) bool {
			m.DownstreamClients.Range(func(i int, c *connection.DownstreamClient) bool {
				if logrus.GetLevel() == logrus.TraceLevel {
					feeShare, _ := m.FeeShareIndividual.Load(feeInfo)
					logrus.WithFields(logrus.Fields{
						"FeePct":             feeInfo.Pct,
						"FeeWallet":          feeInfo.Wallet,
						"Share":              m.TotalShare,
						"FeeShareIndividual": feeShare,
					}).Tracef("[%s][%s] 发送抽水份额", m.PoolServer.Config.Name, m.GetID())
				}

				err = c.Write(payload.In)
				if err != nil {
					logrus.Debugf("[%s][%s][%s] 上游转发到下游失败: %s", m.PoolServer.Config.Name, m.GetID(), c.Connection.Conn.RemoteAddr().String(), err.Error())
					payload.UpstreamClient.DownstreamClient.Shutdown()
					c.Shutdown()
					return true
				}

				return true
			})
			return true
		})

		return
	}

	if payload.UpstreamClient.DownstreamClient == nil {
		payload.UpstreamClient.Shutdown()
		return
	}

	if payload.UpstreamClient.DownstreamClient.WorkerMiner == nil {
		payload.UpstreamClient.Shutdown()
		return
	}

	// 分发
	if payload.UpstreamClient.DownstreamClient.WorkerMiner.DropUpstream {
		return
	}

	err = payload.UpstreamClient.DownstreamClient.Write(payload.In)
	if err != nil {
		logrus.Debugf("[%s][%s][%s] 上游转发到下游失败: %s", payload.UpstreamClient.PoolServer.Config.Name, payload.UpstreamClient.DownstreamClient.WorkerMiner.GetID(), payload.UpstreamClient.DownstreamClient.Connection.Conn.RemoteAddr(), err.Error())
		payload.UpstreamClient.DownstreamClient.Shutdown()
		return
	}
}
