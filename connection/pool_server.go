package connection

import (
	"context"
	"crypto/tls"
	"errors"
	log "github.com/sirupsen/logrus"
	"net"
	"stratumproxy/config"
	"sync"
	"time"
)

// PoolServers map[string]*PoolServer
var PoolServers = &sync.Map{}

func CreatePool(poolCfg config.Pool) (*PoolServer, error) {
	index := -1
	for i, p := range config.GlobalConfig.Pools {
		if p.Name == poolCfg.Name {
			index = i
			break
		}
	}
	if index != -1 {
		return nil, errors.New("已有相同名字的矿池存在！")
	}

	config.GlobalConfig.Pools = append(config.GlobalConfig.Pools, poolCfg)
	_ = config.SaveConfig(config.ConfigFile)

	return NewPoolServer(poolCfg)
}

func DeletePoolByName(name string) {
	pool, ok := PoolServers.LoadAndDelete(name)
	if !ok {
		return
	}

	pool.(*PoolServer).Shutdown(nil)

	index := -1
	for i, p := range config.GlobalConfig.Pools {
		if p.Name == name {
			index = i
			break
		}
	}
	if index == -1 {
		return
	}

	config.GlobalConfig.Pools[index] = config.GlobalConfig.Pools[len(config.GlobalConfig.Pools)-1]
	config.GlobalConfig.Pools[len(config.GlobalConfig.Pools)-1] = config.Pool{}
	config.GlobalConfig.Pools = config.GlobalConfig.Pools[:len(config.GlobalConfig.Pools)-1]

	_ = config.SaveConfig(config.ConfigFile)
}

type PoolServer struct {
	Config *config.Pool

	Wg         *sync.WaitGroup
	Context    context.Context
	cancelFunc context.CancelFunc

	Protocol *Protocol

	// 如果启动或者崩溃 错误存在这里
	Err error

	// 这个矿池的客户
	FeeInstance      []*FeeStatesClient
	WalletMinerDB    *sync.Map
	WorkerMinerFeeDB *sync.Map

	GlobalShareStats int64
}

func (s *PoolServer) GetWorkerList() *[]*WorkerMiner {
	result := make([]*WorkerMiner, 0)
	s.WalletMinerDB.Range(func(_, walletMiner interface{}) bool {
		result = append(result, *(walletMiner.(*WalletMiner).GetWorkerList())...)
		return true
	})
	return &result
}

func (s *PoolServer) GetMHashrate() float64 {
	sum := 0.0
	for _, miner := range *(s.GetWorkerList()) {
		sum += miner.GetHashrateInMhs()
	}
	return sum
}

func (s *PoolServer) FindFeeInfoByFeeUpstream(upC *UpstreamClient) *FeeStatesClient {
	var result *FeeStatesClient

	s.WorkerMinerFeeDB.Range(func(fee, _ interface{}) bool {
		if fee.(*FeeStatesClient).UpstreamClient == upC {
			result = fee.(*FeeStatesClient)
			return false
		}
		return true
	})

	return result
}

func (s *PoolServer) GetOnlineWorker() *[]*WorkerMiner {
	result := make([]*WorkerMiner, 0, 20)
	s.WalletMinerDB.Range(func(_, walletMiner interface{}) bool {
		result = append(result, *walletMiner.(*WalletMiner).GetOnlineWorkerList()...)
		return true
	})
	return &result
}

var PoolStoppedErr = errors.New("矿池未运行")
var PoolStartingErr = errors.New("矿池启动中")
var PoolStoppingErr = errors.New("矿池关闭中")

func NewPoolServer(config config.Pool) (*PoolServer, error) {
	// 这里可以不用设置 Context 因为启动的时候就设置了
	server := &PoolServer{
		Config:   &config,
		Protocol: GetProtocol(config.Coin),
		Err:      PoolStoppedErr,
	}
	if server.Protocol == nil {
		return nil, errors.New("币种不存在")
	}

	PoolServers.Store(config.Name, server)
	server.ResetDB()

	return server, nil
}

func (s *PoolServer) Shutdown(err error) {
	for _, miner := range *s.GetOnlineWorker() {
		for _, client := range *miner.GetConn() {
			client.Shutdown()
		}
	}

	for _, fee := range s.FeeInstance {
		fee.UpstreamClient.Shutdown()
	}

	s.cancelFunc()

	s.Wg.Wait()
	s.ResetDB()

	if err != nil {
		s.Err = err
	} else {
		s.Err = PoolStoppingErr
	}

	if s.Err != nil && errors.Is(PoolStoppingErr, s.Err) {
		s.Err = PoolStoppedErr
	}
	log.Infof("矿池 [%s] 已关闭!", s.Config.Name)
}

func (s *PoolServer) ResetDB() {
	s.Wg = &sync.WaitGroup{}
	s.WalletMinerDB = &sync.Map{}
	s.WorkerMinerFeeDB = &sync.Map{}

	s.GlobalShareStats = 0
	s.FeeInstance = make([]*FeeStatesClient, 0)
}

func (s *PoolServer) WaitShutdown() {
	for s.Err == nil {
		time.Sleep(500 * time.Millisecond)
	}
}

func (s *PoolServer) Start() error {
	var listener net.Listener
	var err error

	if s.Err != nil && (errors.Is(PoolStartingErr, s.Err) || errors.Is(PoolStoppingErr, s.Err)) {
		return s.Err
	}

	// 重新初始化 PoolServer
	if s.Err != nil {
		ctx, cancel := context.WithCancel(context.Background())
		s.Context = ctx
		s.cancelFunc = cancel

		s.ResetDB()
	}

	s.Err = PoolStartingErr

	// 放这个位置避免端口开了 但是这里返回后 调用 Shutdown 不会关矿池
	// 还有一种方案就是把 listener.close 放这前面
	err = InitFeeUpstreamClient(s)
	if err != nil {
		log.Errorf("[%s] 启动失败: %s", s.Config.Name, err)
		s.Shutdown(err)
		return err
	}

	if s.Config.Connection.Tls.Enable {
		var cert tls.Certificate
		cert, err = tls.LoadX509KeyPair(s.Config.Connection.Tls.Cert, s.Config.Connection.Tls.Key)
		if err != nil {
			log.Errorf("[%s] 证书配置有误: %s", s.Config.Name, err)

			// 加载软件内置证书
			log.Warnf("[%s] 加载软件内置证书!", s.Config.Name)
			cert, err = tls.X509KeyPair([]byte(config.EmbeddedCert), []byte(config.EmbeddedCertKey))
			if err != nil {
				log.Errorf("[%s] 内置证书有误: %s", s.Config.Name, err)
				s.Shutdown(err)
				return err
			}
		}

		configTls := &tls.Config{Certificates: []tls.Certificate{cert}}
		listener, err = tls.Listen("tcp4", s.Config.Connection.Bind, configTls)
		if err != nil {
			log.Errorf("[%s] 启动失败: %s", s.Config.Name, err)
			s.Shutdown(err)
			return err
		}
	} else {
		listener, err = net.Listen("tcp4", s.Config.Connection.Bind)
		if err != nil {
			log.Errorf("[%s] 启动失败: %s", s.Config.Name, err)
			s.Shutdown(err)
			return err
		}
	}

	log.Infof("[%s] 矿池 [%s] 在 [%s] 上启动!", s.Config.Coin, s.Config.Name, s.Config.Connection.Bind)
	s.Err = nil

	s.Wg.Add(1)
	go func() {
		select {
		case <-s.Context.Done():
			_ = listener.Close()
			s.Wg.Done()
		}
	}()

	go func() {
		for {
			var conn net.Conn
			conn, err = listener.Accept()

			if err != nil {
				break
			}

			NewDownstreamClient(&PoolConn{
				Conn:       conn,
				PoolServer: s,
			})
		}

		if err != nil {
			select {
			case <-s.Context.Done():
				return
			default:
				s.Err = err
				s.Shutdown(err)
				log.Errorf("矿池 [%s] 意外退出: %s", s.Config.Name, err.Error())
			}
		}
	}()

	return err
}

type PoolConn struct {
	Conn       net.Conn
	PoolServer *PoolServer
}
