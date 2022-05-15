package eth

import (
	"github.com/sirupsen/logrus"
	"stratumproxy/connection"
	"strings"
)

func DownInjectorDropUnauthClient(payload *connection.InjectorDownstreamPayload) {
	if strings.Contains(string(payload.In), "eth_submitLogin") {
		return
	}

	// 在这里适配其他内核
	// TODO: teamredminer

	if !payload.DownstreamClient.AuthPackSent {
		payload.IsTerminated = true
		logrus.Debugf("[%s][%s][DownInjectorDropUnauthClient] 丢弃未认证请求: [%s]", payload.DownstreamClient.Connection.PoolServer.Config.Name, payload.DownstreamClient.Connection.Conn.RemoteAddr(), string(payload.In))
		return
	}
}
