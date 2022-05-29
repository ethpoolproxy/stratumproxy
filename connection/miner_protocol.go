package connection

import (
	"strings"
)

var Protocols = make(map[string]*Protocol)

func GetProtocol(name string) *Protocol {
	name = strings.ToLower(name)

	if name == "etc" {
		name = "eth"
	}

	return Protocols[name]
}

type ProtocolHandler interface {
	HandleDownstreamDisconnect(client *DownstreamClient)
	HandleFeeControl(worker *WorkerMiner)
	InitialUpstreamConn(upstream *UpstreamClient) error
	InitialUpstreamAuth(upstream *UpstreamClient, identifier MinerIdentifier) error
}

type ProtocolInjector struct {
	DownstreamInjectorProcessors []func(payload *InjectorDownstreamPayload)
	UpstreamInjectorProcessors   []InjectorProcessorUpstream
}

type Protocol struct {
	ProtocolInjector
	ProtocolHandler
}
