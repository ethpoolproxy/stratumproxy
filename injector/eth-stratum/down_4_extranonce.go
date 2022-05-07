package eth_stratum

import (
	"gopkg.in/guregu/null.v4"
	"stratumproxy/connection"
	ethstratum "stratumproxy/protocol/eth-stratum"
	"strings"
)

func DownInjectorExtranonce(payload *connection.InjectorDownstreamPayload) {
	if !strings.Contains(string(payload.In), "mining.extranonce.subscribe") {
		return
	}

	request := &ethstratum.RequestGeneral{}
	err := request.Parse(payload.In)
	if err != nil {
		return
	}

	payload.IsTerminated = true
	response := ethstratum.ResponseGeneral{
		Id:     request.Id,
		Result: true,
		Error:  null.String{},
	}
	out, _ := response.Build()
	payload.Out = out

	payload.DownstreamClient.Upstream.ProtocolData.Store("extranonce.subscribe", true)
}
