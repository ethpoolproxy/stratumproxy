package eth_stratum

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/guregu/null.v4"
	"stratumproxy/connection"
	ethstratum "stratumproxy/protocol/eth-stratum"
	"strings"
	"sync"
	"time"
)

func DownInjectorAuth(payload *connection.InjectorDownstreamPayload) {
	if !strings.Contains(string(payload.In), "mining.authorize") {
		return
	}

	request := &ethstratum.RequestAuthorize{}
	err := request.Parse(payload.In)
	if err != nil {
		payload.IsTerminated = true
		payload.ShouldShutdown = true
		logrus.Errorf("[%s][%s][DownInjectorEthSubmitLogin] 无法解析登录包: %s", payload.DownstreamClient.Connection.PoolServer.Config.Name, payload.DownstreamClient.Connection.Conn.RemoteAddr(), err.Error())
		return
	}

	// 防止一个连接发送多个认证包开启多个上游导致游离
	if payload.DownstreamClient.AuthPackSent {
		// 发送登陆成功
		resp, _ := ethstratum.ResponseGeneral{
			Id:     request.Id,
			Result: true,
			Error:  null.NewString("", false),
		}.Build()
		payload.Out = resp
		payload.IsTerminated = true
		return
	}

	// 记录矿工登录信息
	id := request.Params[0] + "." + request.Worker

	// 如果矿工名和钱包地址在一起
	if strings.Contains(request.Params[0], ".") && request.Worker == "" {
		id = request.Params[0]
		request.Worker = strings.Split(id, ".")[1]
		request.Params[0] = strings.Split(id, ".")[0]
	}

	walletMiner, _ := payload.DownstreamClient.Connection.PoolServer.WalletMinerDB.LoadOrStore(request.Params[0], &connection.WalletMiner{
		Clients: &sync.Map{},
	})

	workerMiner, exist := walletMiner.(*connection.WalletMiner).Clients.LoadOrStore(request.Worker, &connection.WorkerMiner{
		PoolServer: payload.DownstreamClient.Connection.PoolServer,
		Identifier: &connection.MinerIdentifier{
			Wallet:     request.Params[0],
			WorkerName: request.Worker,
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

	if payload.DownstreamClient.Upstream == nil {
		payload.IsTerminated = true
		payload.ShouldShutdown = true
		payload.ForceShutdown = true
		return
	}
	err = payload.DownstreamClient.Upstream.AuthInitial(connection.MinerIdentifier{
		Wallet:     request.Params[0],
		WorkerName: request.Worker,
	})
	if err != nil {
		payload.IsTerminated = true
		payload.ShouldShutdown = true
		response, _ := ethstratum.ResponseGeneral{
			Id:     request.Id,
			Result: false,
			Error:  null.NewString("登录矿池失败: "+err.Error(), true),
		}.Build()
		payload.Out = response
		return
	}

	workerMiner.(*connection.WorkerMiner).DownstreamClients.Add(payload.DownstreamClient)

	payload.DownstreamClient.WorkerMiner = workerMiner.(*connection.WorkerMiner)
	payload.DownstreamClient.WalletMiner = walletMiner.(*connection.WalletMiner)

	payload.DownstreamClient.AuthPackSent = true

	// 发送登陆成功
	response, _ := ethstratum.ResponseGeneral{
		Id:     request.Id,
		Result: true,
		Error:  null.NewString("", false),
	}.Build()
	payload.Out = append(payload.Out, response...)

	// mining.set_difficulty | set_difficulty
	difficulty, _ := payload.DownstreamClient.Upstream.ProtocolData.LoadOrStore("difficulty", 4)
	response, _ = ethstratum.ResponseMethodGeneral{
		Id:     null.NewInt(0, false),
		Method: "mining.set_difficulty",
		Params: []interface{}{difficulty.(float64)},
	}.Build()
	payload.Out = append(payload.Out, response...)

	// 成功了就不执行其他的了
	payload.IsTerminated = true
}
