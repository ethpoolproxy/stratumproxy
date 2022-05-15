package eth

import (
	"errors"
	"stratumproxy/config"
	"stratumproxy/connection"
	ethcommon "stratumproxy/injector/eth-common"
	"stratumproxy/protocol/eth"
	"strings"
)

func RegisterProtocol() {
	connection.Protocols["eth"] = &connection.Protocol{
		ProtocolHandler:  protocolHandler,
		ProtocolInjector: *protocolInjector,
	}
	connection.Protocols["etc"] = &connection.Protocol{
		ProtocolHandler:  protocolHandler,
		ProtocolInjector: *protocolInjector,
	}
	config.ProtocolList = append(config.ProtocolList, "eth")
	config.ProtocolList = append(config.ProtocolList, "etc")
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

func (p *ProtocolHandler) InitialUpstreamConn(_ *connection.UpstreamClient) error {
	return nil
}

func (p *ProtocolHandler) InitialUpstreamAuth(upstream *connection.UpstreamClient, identifier connection.MinerIdentifier) error {
	upstream.DownstreamIdentifier = identifier

	json := []byte("{\"compact\":true,\"id\":1,\"method\":\"eth_submitLogin\",\"params\":[\"" + upstream.DownstreamIdentifier.Wallet + "\",\"\"],\"worker\":\"" + upstream.DownstreamIdentifier.WorkerName + "\"}\n")
	err := upstream.Write(json)
	if err != nil {
		return errors.New("发送登陆包失败: " + err.Error())
	}

	// 等待登陆返回
	data, err := upstream.ReadOnce(8)
	if err != nil {
		return errors.New("获取登陆结果失败: " + err.Error())
	}

	// 验证登陆包
	var loginResponse eth.ResponseSubmitLogin
	err = loginResponse.Parse(data)
	if err != nil {
		return err
	}

	// 验证返回是否成功
	if !loginResponse.Result {
		if strings.Contains(loginResponse.Error, "Invalid user") || strings.Contains(loginResponse.Error, "Bad user name") {
			return connection.ErrUpstreamInvalidUser
		}

		return errors.New(loginResponse.Error)
	}

	err = upstream.Write([]byte("{\"id\":5,\"method\":\"eth_getWork\",\"params\":[]}\n"))
	if err != nil {
		return errors.New("无法发送 getWork: " + err.Error())
	}

	return nil
}

var protocolInjector = &connection.ProtocolInjector{
	DownstreamInjectorProcessors: []func(payload *connection.InjectorDownstreamPayload){
		DownInjectorDropUnauthClient,
		DownInjectorEthSubmitLogin,
		DownInjectorRecordGetWork,
		DownInjectorEthSubmitHashrate,
		DownInjectorSubmitWork,
		DownInjectorCapture,
	},
	UpstreamInjectorProcessors: []connection.InjectorProcessorUpstream{
		{
			DisableWhenFee: false,
			Processors:     UpInjectorSendJob,
		},
	},
}
