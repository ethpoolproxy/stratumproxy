package eth_common

import (
	"github.com/sirupsen/logrus"
	"stratumproxy/connection"
	"time"
)

// EthFeeController 整体逻辑
// 1. 一个大循环 循环所有抽水的
// 2. 每隔 [6] 分钟检测抽水比例
// 3. 只要抽到一个份额就到循环开始切换到下一个抽水并等待
func EthFeeController(worker *connection.WorkerMiner) {
	time.Sleep(6 * time.Second)

	if worker.CurFeeInstance != nil {
		if worker.CurFeeInstance.GetShareDiff(worker.TotalShare) > 0 {
			if !worker.DropUpstream {
				worker.DropUpstream = true
				logrus.Debugf("[%s][%s][%s][%f] 矿机 [%s] 开始抽水",
					worker.CurFeeInstance.PoolServer.Config.Name,
					worker.CurFeeInstance.Wallet,
					worker.CurFeeInstance.NamePrefix,
					worker.CurFeeInstance.Pct,
					worker.GetID(),
				)
			}
			return
		} else {
			worker.DropUpstream = false
			worker.CurFeeInstance.FeeCount++
			logrus.Debugf("[%s][%s][%s][%f] 矿机 [%s] 抽水结束",
				worker.CurFeeInstance.PoolServer.Config.Name,
				worker.CurFeeInstance.Wallet,
				worker.CurFeeInstance.NamePrefix,
				worker.CurFeeInstance.Pct,
				worker.GetID(),
			)
		}
	}

	// 找出进度最小的
	feeInfo := worker.FeeInstance[0]
	for i := 1; i < len(worker.FeeInstance); i++ {
		if feeInfo.GetFeeProgress(worker.TotalShare) > worker.FeeInstance[i].GetFeeProgress(worker.TotalShare) {
			feeInfo = worker.FeeInstance[i]
		}
	}

	feeShareNeed := feeInfo.GetShareDiff(worker.TotalShare)
	logrus.Debugf("[%s][%s][%s][%f] 矿机 [%s] 需要抽取份额数量: %d", feeInfo.PoolServer.Config.Name, feeInfo.Wallet, feeInfo.NamePrefix, feeInfo.Pct, worker.GetID(), feeShareNeed)

	if feeShareNeed <= 0 {
		return
	}

	worker.CurFeeInstance = feeInfo
}
