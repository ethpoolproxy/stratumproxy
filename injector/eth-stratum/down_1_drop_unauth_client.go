package eth_stratum

import (
	"github.com/sirupsen/logrus"
	"stratumproxy/connection"
	"strings"
)

func DownInjectorDropUnauthClient(payload *connection.InjectorDownstreamPayload) {
	if strings.Contains(string(payload.In), "mining.authorize") {
		return
	}

	if strings.Contains(string(payload.In), "eth_submitLogin") {
		payload.IsTerminated = true
		payload.ShouldShutdown = true
		payload.ForceShutdown = true
		return
	}

	if !payload.DownstreamClient.AuthPackSent {
		payload.IsTerminated = true
		logrus.Debugf("[%s][%s][DownInjectorDropUnauthClient] 丢弃未认证请求: [%s]", payload.DownstreamClient.Connection.PoolServer.Config.Name, payload.DownstreamClient.Connection.Conn.RemoteAddr(), string(payload.In))
		return
	}
}
