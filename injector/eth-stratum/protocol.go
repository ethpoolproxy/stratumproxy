package eth_stratum

import (
	"errors"
	"fmt"
	"io"
	"stratumproxy/config"
	"stratumproxy/connection"
	"stratumproxy/injector/eth"
	ethcommon "stratumproxy/injector/eth-common"
	ethstratum "stratumproxy/protocol/eth-stratum"
	"strings"
)

func RegisterProtocol() {
	connection.Protocols["eth-stratum"] = &connection.Protocol{
		ProtocolHandler:  protocolHandler,
		ProtocolInjector: *protocolInjector,
	}
	config.ProtocolList = append(config.ProtocolList, "eth-stratum")
}

var protocolHandler = &ProtocolHandler{}

type ProtocolHandler struct {
	connection.ProtocolHandler
}

func (p *ProtocolHandler) HandleFeeControl(pool *connection.PoolServer) {
	ethcommon.EthFeeController(pool)
}

func (p *ProtocolHandler) HandleDownstreamDisconnect(_ *connection.DownstreamClient) {
}

func (p *ProtocolHandler) InitialUpstreamConn(upstream *connection.UpstreamClient) error {
	subscribePayload := fmt.Sprintf("{\"id\": 114514, \"method\": \"mining.subscribe\", \"params\": [\"StratumProxy/%s\",\"EthereumStratum/1.0.0\"]}\n", config.GitTag)
	err := upstream.Write([]byte(subscribePayload))
	if err != nil {
		return err
	}

	data, err := upstream.ReadOnce(8)
	if err != nil {
		if errors.Is(io.EOF, err) {
			return err
		}
		return err
	}

	// 认证结果
	if strings.Contains(string(data), "114514") {
		response := &ethstratum.ResponseHandshakeNotify{}
		err := response.Parse(data)
		if err != nil {
			return err
		}

		upstream.ProtocolData.Store("extranonce", response.Result[0].([]interface{})[1])
		upstream.ProtocolData.Store("extranonce2", response.Result[1])
	}

	return nil
}

// InitialUpstreamAuth 一直读到下发任务
func (p *ProtocolHandler) InitialUpstreamAuth(upstream *connection.UpstreamClient, id connection.MinerIdentifier) error {
	upstream.DownstreamIdentifier = id

	subscribePayload := fmt.Sprintf("{\"id\": 1919810, \"method\": \"mining.authorize\", \"params\": [\"%s.%s\", \"\"]}", upstream.DownstreamIdentifier.Wallet, upstream.DownstreamIdentifier.WorkerName)
	err := upstream.Write([]byte(subscribePayload))
	if err != nil {
		return errors.New("发送登陆包失败: " + err.Error())
	}

	for {
		data, err := upstream.ReadOnce(8)
		if err != nil {
			return errors.New("获取登陆结果失败: " + err.Error())
		}

		if strings.Contains(string(data), "\"method\":\"mining.set_difficulty\"") {
			response := &ethstratum.ResponseMiningSetDifficulty{}
			err = response.Parse(data)
			if err != nil {
				return err
			}
			upstream.ProtocolData.Store("difficulty", response.Params[0])
		}

		if strings.Contains(string(data), "1919810") || strings.Contains(string(data), "\"result\":false") || strings.Contains(string(data), "\"id\":999") {
			response := &ethstratum.ResponseGeneral{}
			err = response.Parse(data)
			if err != nil {
				return err
			}

			if !response.Result {
				return errors.New("身份验证失败: " + response.Error.String)
			}
		}

		if strings.Contains(string(data), "\"method\":\"mining.notify\"") {
			response := &ethstratum.ResponseGeneral{}
			err = response.Parse(data)
			if err != nil {
				return err
			}
			break
		}

	}

	return nil
}

var protocolInjector = &connection.ProtocolInjector{
	DownstreamInjectorProcessors: []func(payload *connection.InjectorDownstreamPayload){
		DownInjectorMiningSubscribe,
		DownInjectorDropUnauthClient,
		DownInjectorAuth,
		DownInjectorSubmitWork,
		DownInjectorExtranonce,
		eth.DownInjectorEthSubmitHashrate,
	},
	UpstreamInjectorProcessors: []connection.InjectorProcessorUpstream{
		{
			DisableWhenFee: false,
			Processors:     UpInjectorSendJob,
		},
		{
			DisableWhenFee: false,
			Processors:     UpInjectorSetExtranonce,
		},
	},
}
