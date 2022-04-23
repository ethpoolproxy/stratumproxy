package connection

import (
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"math"
	"stratumproxy/config"
	"strings"
	"sync"
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

// GetShareDiff 返回距离目标比例还有多少个份额
func (f *FeeStatesClient) GetShareDiff() int {
	// 应该抽的数量 = 当前份额数量 * 抽水比例
	desertFeeShare := (f.Pct / 100) * float64(f.PoolServer.GlobalShareStats)

	// 还要抽多少 = 应该抽的数量 - 当前抽的数量
	feeShareNeed := int(desertFeeShare - float64(f.Share))

	return feeShareNeed
}

// GetFeeProgress 当前份额 / GetShareDiff = 抽水进度
func (f *FeeStatesClient) GetFeeProgress() float64 {
	result := float64(f.Share) / (float64(f.GetShareDiff()) + float64(f.Share))
	if math.IsNaN(result) {
		result = 1
	}
	return result
}

func (f *FeeStatesClient) AddShare(d int64) {
	atomic.AddInt64(&f.Share, d)
}

func InitFeeUpstreamClient(pool *PoolServer) error {
	// 12 秒超时
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	fees := make([]config.FeeState, 0)
	if pool.Config.FeeConfig.Pct > 0 {
		fees = append(fees, pool.Config.FeeConfig)
	}
	for _, state := range config.FeeStates[strings.ToLower(pool.Config.Coin)] {
		if state.Upstream.Address == "" {
			state.Upstream = pool.Config.Upstream
			logrus.Debugf("[%s][%s][%s][%f] 跟随上游矿池: %s", pool.Config.Name, state.Wallet, state.NamePrefix, state.Pct, state.Upstream.Address)
		}

		fees = append(fees, state)
	}

	for _, info := range fees {
		select {
		case <-ctx.Done():
			return errors.New("连接矿池超时")
		default:
			feeStatesClient := &FeeStatesClient{
				FeeState:   info,
				PoolServer: pool,
			}
			pool.FeeInstance = append(pool.FeeInstance, feeStatesClient)

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
					upClient, err = NewUpstreamClient(pool, info.Upstream, MinerIdentifier{
						Wallet:     info.Wallet,
						WorkerName: "StratumProxy",
					})
					if err != nil {
						if errors.Is(UpstreamInvalidUserErr, err) {
							return err
						}
						logrus.Warnf("[%s] 网络连接失败 [%s]！重试中...", pool.Config.Name, err)
						time.Sleep(2 * time.Second)
						continue
					}
				}
			}
			feeStatesClient.UpstreamClient = upClient
			pool.WorkerMinerFeeDB.Store(feeStatesClient, &WorkerMinerSliceWrapper{
				RWMutex:     sync.RWMutex{},
				workerMiner: make([]*WorkerMiner, 0),
			})
		}
	}

	pool.Wg.Add(1)
	go func() {
		for true {
			select {
			case <-pool.Context.Done():
				logrus.Debugf("[%s] 矿池关闭，抽水退出!", pool.Config.Name)
				pool.Wg.Done()
				return
			default:
				pool.Protocol.HandleFeeControl(pool)
			}
		}
	}()

	return nil
}
