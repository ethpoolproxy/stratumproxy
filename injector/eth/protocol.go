package eth

import (
	"stratumproxy/connection"
)

func RegisterProtocol() {
	connection.Protocols["eth"] = &connection.Protocol{
		ProtocolHandler:  protocolHandler,
		ProtocolInjector: *protocolInjector,
	}
}

var protocolHandler = &ProtocolHandler{}

type ProtocolHandler struct {
	connection.ProtocolHandler
}

func (p *ProtocolHandler) HandleDownstreamDisconnect(_ *connection.DownstreamClient) {
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
