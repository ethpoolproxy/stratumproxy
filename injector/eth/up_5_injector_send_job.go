package eth

import (
	"github.com/sirupsen/logrus"
	"stratumproxy/connection"
	"stratumproxy/protocol/eth"
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

	// 如果是从抽水矿池发来的
	if payload.UpstreamClient.DownstreamClient == nil {
		m := payload.UpstreamClient.WorkerMiner

		if !m.DropUpstream {
			return
		}

		if m.CurFeeInstance.UpstreamClient != payload.UpstreamClient {
			return
		}

		// 群发给要抽水的
		for _, c := range *payload.UpstreamClient.WorkerMiner.DownstreamClients.Copy() {
			err = c.Write(payload.In)
			if err != nil {
				logrus.Errorf("[UpInjectorSendJob-FeeFw][%s][%s][%s] 上游转发到下游失败: %s", m.PoolServer.Config.Name, m.GetID(), c.Connection.Conn.RemoteAddr().String(), err)
				c.Shutdown()
				continue
			}

			continue
		}

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
		logrus.Errorf("[UpInjectorSendJob-Fw][%s][%s][%s] 上游转发到下游失败: %s", payload.UpstreamClient.PoolServer.Config.Name, payload.UpstreamClient.DownstreamClient.WorkerMiner.GetID(), payload.UpstreamClient.DownstreamClient.Connection.Conn.RemoteAddr(), err.Error())
		payload.UpstreamClient.DownstreamClient.Shutdown()
		return
	}
}
