package eth_stratum

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/guregu/null.v4"
	"stratumproxy/connection"
	ethstratum "stratumproxy/protocol/eth-stratum"
	"strings"
)

func DownInjectorMiningSubscribe(payload *connection.InjectorDownstreamPayload) {
	if !strings.Contains(string(payload.In), "mining.subscribe") {
		return
	}

	subsReq := &ethstratum.RequestSubscribe{}
	err := subsReq.Parse(payload.In)
	if err != nil {
		return
	}

	// 创建上游
	upC, err := connection.NewUpstreamClient(payload.DownstreamClient.Connection.PoolServer, payload.DownstreamClient.Connection.PoolServer.Config.Upstream)
	if err != nil {
		// 出错了当然要打断啊亲
		logrus.Warnf("[%s][%s][DownInjectorMiningSubscribe][%s] 无法连接上游服务器: %s", payload.DownstreamClient.Connection.PoolServer.Config.Name, payload.DownstreamClient.Connection.Conn.RemoteAddr(), subsReq.Params[0], err)
		payload.IsTerminated = true
		payload.ShouldShutdown = true
		payload.ForceShutdown = true
		var response, _ = ethstratum.ResponseMiningNotify{
			Id:     null.NewInt(int64(subsReq.Id), true),
			Method: "mining.notify",
			Error:  null.NewString("无法连接上游服务器: "+err.Error(), false),
		}.Build()
		payload.Out = response
		return
	}
	upC.DownstreamClient = payload.DownstreamClient
	payload.DownstreamClient.Upstream = upC

	// mining.notify | extranonce
	extraNonce1, _ := upC.ProtocolData.LoadOrStore("extranonce", "0000")
	extraNonce2, _ := upC.ProtocolData.LoadOrStore("extranonce2", "00")
	response, _ := ethstratum.ResponseMiningNotify{
		Id: null.NewInt(int64(subsReq.Id), true),
		Result: []interface{}{
			[]string{"mining.notify", extraNonce1.(string), "EthereumStratum/1.0.0"},
			extraNonce2.(string),
		},
	}.Build()
	payload.Out = append(payload.Out, response...)

	// mining.set_extranonce | extranonce2
	response, _ = ethstratum.ResponseMethodGeneral{
		Method: "mining.set_extranonce",
		Params: []interface{}{extraNonce2.(string)},
	}.Build()
	payload.Out = append(payload.Out, response...)

	payload.IsTerminated = true
	return
}
