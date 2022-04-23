package eth

import (
	"errors"
	"github.com/goccy/go-json"
	"github.com/sirupsen/logrus"
	"stratumproxy/connection"
	"stratumproxy/protocol/eth"
	"strings"
	"sync"
	"time"
)

// DownInjectorEthSubmitLogin 任务是 & 篡改矿工的登录信息
func DownInjectorEthSubmitLogin(payload *connection.InjectorDownstreamPayload) {
	if !strings.Contains(string(payload.In), "eth_submitLogin") {
		return
	}

	var loginInfo eth.RequestSubmitLogin
	err := loginInfo.Parse(payload.In)
	if err != nil {
		payload.IsTerminated = true
		payload.ShouldShutdown = true
		payload.ForceShutdown = true
		logrus.Errorf("[%s][%s][DownInjectorEthSubmitLogin] 无法解析登录包: %s", payload.DownstreamClient.Connection.PoolServer.Config.Name, payload.DownstreamClient.Connection.Conn.RemoteAddr(), err.Error())
		return
	}
	err = loginInfo.Valid()
	if errors.Is(err, eth.MethodNotMatchErr) {
		return
	}
	if err != nil && !errors.Is(err, eth.MethodNotMatchErr) {
		payload.IsTerminated = true
		payload.ShouldShutdown = true
		payload.ForceShutdown = true
		logrus.Errorf("[%s][%s][DownInjectorEthSubmitLogin] 登录包有误: %s", payload.DownstreamClient.Connection.PoolServer.Config.Name, payload.DownstreamClient.Connection.Conn.RemoteAddr(), err.Error())
		return
	}

	// 防止一个连接发送多个认证包开启多个上游导致游离
	if payload.DownstreamClient.AuthPackSent {
		// 发送登陆成功
		resp, _ := eth.ResponseSubmitLogin{
			Id:     loginInfo.Id,
			Result: true,
			Error:  "",
		}.Build()
		payload.Out = resp
		payload.IsTerminated = true
		return
	}

	// 记录矿工登录信息
	id := loginInfo.Params[0] + "." + loginInfo.Worker

	// 如果矿工名和钱包地址在一起
	if strings.Contains(loginInfo.Params[0], ".") && loginInfo.Worker == "" {
		id = loginInfo.Params[0]
		loginInfo.Worker = strings.Split(id, ".")[1]
		loginInfo.Params[0] = strings.Split(id, ".")[0]
	}

	// 创建钱包类
	walletMiner, _ := payload.DownstreamClient.Connection.PoolServer.WalletMinerDB.LoadOrStore(loginInfo.Params[0], &connection.WalletMiner{
		Clients: &sync.Map{},
	})

	workerMiner, exist := walletMiner.(*connection.WalletMiner).Clients.LoadOrStore(loginInfo.Worker, &connection.WorkerMiner{
		PoolServer: payload.DownstreamClient.Connection.PoolServer,
		Identifier: &connection.MinerIdentifier{
			Wallet:     loginInfo.Params[0],
			WorkerName: loginInfo.Worker,
		},
		FeeShareIndividual:        &sync.Map{},
		LastFeeTime:               time.Unix(0, 0),
		DownstreamClients:         &connection.DownstreamClientMutexWrapper{},
		TimeIntervalShareStats:    &connection.ShareStatsIntervalMap{},
		TimeIntervalFeeShareStats: &connection.ShareStatsIntervalMap{},
	})
	workerMiner.(*connection.WorkerMiner).TimeIntervalShareStats.AddStatsSlice(&[]*connection.ShareStatsInterval{
		connection.NewShareStatsInterval(15 * time.Minute),
		connection.NewShareStatsInterval(30 * time.Minute),
		connection.NewShareStatsInterval(60 * time.Minute),
	})
	workerMiner.(*connection.WorkerMiner).TimeIntervalFeeShareStats.AddStatsSlice(&[]*connection.ShareStatsInterval{
		connection.NewShareStatsInterval(15 * time.Minute),
		connection.NewShareStatsInterval(30 * time.Minute),
	})
	workerMiner.(*connection.WorkerMiner).ConnectAt = time.Now()

	if !exist {
		logrus.Infof("[%s][%s][DownInjectorEthSubmitLogin][%s] 矿工已注册&上线!", payload.DownstreamClient.Connection.PoolServer.Config.Name, payload.DownstreamClient.Connection.Conn.RemoteAddr(), id)
	} else {
		workerMiner.(*connection.WorkerMiner).ConnectAt = time.Now()
		logrus.Infof("[%s][%s][DownInjectorEthSubmitLogin][%s] 矿工已上线!", payload.DownstreamClient.Connection.PoolServer.Config.Name, payload.DownstreamClient.Connection.Conn.RemoteAddr(), id)
	}

	// 创建专属上游
	upC, err := connection.NewUpstreamClient(payload.DownstreamClient.Connection.PoolServer, payload.DownstreamClient.Connection.PoolServer.Config.Upstream, connection.MinerIdentifier{
		Wallet:     loginInfo.Params[0],
		WorkerName: loginInfo.Worker,
	})
	if err != nil {
		// 出错了当然要打断啊亲
		logrus.Warnf("[%s][%s][DownInjectorEthSubmitLogin][%s] 无法连接上游服务器: %s", payload.DownstreamClient.Connection.PoolServer.Config.Name, payload.DownstreamClient.Connection.Conn.RemoteAddr(), id, err)
		payload.IsTerminated = true
		payload.ShouldShutdown = true
		payload.ForceShutdown = true
		response, _ := json.Marshal(eth.ResponseSubmitLogin{
			Id:     loginInfo.Id,
			Result: false,
			Error:  "无法连接上游服务器: " + err.Error(),
		})
		payload.Out = response
		return
	}
	upC.DownstreamClient = payload.DownstreamClient
	payload.DownstreamClient.Upstream = upC

	// 添加这个下游
	workerMiner.(*connection.WorkerMiner).DownstreamClients.Add(payload.DownstreamClient)

	payload.DownstreamClient.WorkerMiner = workerMiner.(*connection.WorkerMiner)
	payload.DownstreamClient.WalletMiner = walletMiner.(*connection.WalletMiner)

	payload.DownstreamClient.AuthPackSent = true

	// 发送登陆成功
	resp, _ := eth.ResponseSubmitLogin{
		Id:     loginInfo.Id,
		Result: true,
		Error:  "",
	}.Build()
	payload.Out = resp

	// 成功了就不执行其他的了
	payload.IsTerminated = true
}
