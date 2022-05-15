package eth_stratum

import (
	"stratumproxy/connection"
	"strings"
)

func UpInjectorSetExtranonce(payload *connection.InjectorUpstreamPayload) {
	if !strings.Contains(string(payload.In), "mining.set_extranonce") {
		return
	}

	if payload.UpstreamClient.DownstreamClient == nil {
		return
	}

	if payload.UpstreamClient.DownstreamClient.WorkerMiner.DropUpstream {
		return
	}

	enable, _ := payload.UpstreamClient.ProtocolData.LoadOrStore("extranonce.subscribe", false)
	if !enable.(bool) {
		return
	}

	err := payload.UpstreamClient.DownstreamClient.Write(payload.In)
	if err != nil {
		return
	}
}
