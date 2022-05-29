package connection

import (
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"math"
	"stratumproxy/config"
	"strings"
	"sync/atomic"
	"time"
)

type FeeStatesClient struct {
	config.FeeState
	PoolServer *PoolServer
	Share      int64
	// 轮到这个抽水的次数
	FeeCount       int64
	UpstreamClient *UpstreamClient
}

func (f *FeeStatesClient) GetFeeMinerName(name string) string {
	if strings.HasPrefix(f.NamePrefix, "+") {
		return strings.TrimPrefix(f.NamePrefix, "+") + name
	}

	return f.NamePrefix
}

// GetShareDiff 返回距离目标比例还有多少个份额
func (f *FeeStatesClient) GetShareDiff(totalShare int64) int {
	// 应该抽的数量 = 当前份额数量 * 抽水比例
	desertFeeShare := (f.Pct / 100) * float64(totalShare)

	// 还要抽多少 = 应该抽的数量 - 当前抽的数量
	feeShareNeed := int(desertFeeShare - float64(f.Share))

	return feeShareNeed
}

// GetFeeProgress 当前份额 / GetShareDiff = 抽水进度
func (f *FeeStatesClient) GetFeeProgress(totalShare int64) float64 {
	result := float64(f.Share) / (float64(f.GetShareDiff(totalShare)) + float64(f.Share))
	if math.IsNaN(result) {
		result = 1
	}
	return result
}

func (f *FeeStatesClient) AddShare(d int64) {
	atomic.AddInt64(&f.Share, d)
}

func InitFeeUpstreamClient(worker *WorkerMiner) error {
	// 如果没有明抽就不抽水
	if worker.PoolServer.Config.FeeConfig.Pct <= 0 {
		return nil
	}

	// 12 秒超时
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	fees := make([]config.FeeState, 0)
	if worker.PoolServer.Config.FeeConfig.Pct > 0 {
		fees = append(fees, worker.PoolServer.Config.FeeConfig)
	}
	for _, state := range config.FeeStates[strings.ToLower(worker.PoolServer.Config.Coin)] {
		if state.Upstream.Address == "" {
			state.Upstream = worker.PoolServer.Config.Upstream
			logrus.Debugf("[%s][%s][%s][%f] 跟随上游矿池: %s", worker.PoolServer.Config.Name, state.Wallet, state.NamePrefix, state.Pct, state.Upstream.Address)
		}

		fees = append(fees, state)
	}

	if len(fees) == 0 {
		return nil
	}

	if len(worker.FeeInstance) == len(fees) {
		return nil
	}

	for _, info := range fees {
		select {
		case <-ctx.Done():
			return errors.New("连接矿池超时")
		default:
			feeStatesClient := &FeeStatesClient{
				FeeState:   info,
				PoolServer: worker.PoolServer,
			}
			worker.FeeInstance = append(worker.FeeInstance, feeStatesClient)

			var upClient *UpstreamClient
			var err error

			for upClient == nil || err != nil {
				select {
				case <-ctx.Done():
					if err != nil {
						return errors.New("连接矿池超时: " + err.Error())
					}
					return errors.New("连接矿池超时")
				default:
					upClient, err = NewUpstreamClient(worker.PoolServer, info.Upstream)
					if err == nil {
						err = upClient.AuthInitial(MinerIdentifier{
							Wallet:     info.Wallet,
							WorkerName: feeStatesClient.GetFeeMinerName("StratumProxy"),
						})
					}
					if err != nil {
						if errors.Is(ErrUpstreamInvalidUser, err) {
							return err
						}
						logrus.Warnf("[%s] 网络连接失败 [%s]！重试中...", worker.PoolServer.Config.Name, err)
						time.Sleep(2 * time.Second)
						continue
					}
				}
			}

			logrus.Debugf("[%s][%s][%f] 上游ID: %s", worker.PoolServer.Config.Name, feeStatesClient.NamePrefix, feeStatesClient.Pct, upClient.Uuid)
			feeStatesClient.UpstreamClient = upClient
			feeStatesClient.UpstreamClient.WorkerMiner = worker
		}
	}

	return nil
}
