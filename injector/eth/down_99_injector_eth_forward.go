package eth

import (
	"github.com/sirupsen/logrus"
	"stratumproxy/connection"
	"strings"
)

// DownInjectorCapture 记录没被转发的包
func DownInjectorCapture(payload *connection.InjectorDownstreamPayload) {
	if !strings.HasSuffix(string(payload.In), "\n") {
		payload.In = []byte(string(payload.In) + "\n")
	}

	logrus.Debugf("[%s][%s][DownInjectorEthForward] 未处理的包: %s", payload.DownstreamClient.Connection.PoolServer.Config.Name, payload.DownstreamClient.Connection.Conn.RemoteAddr(), string(payload.In))
}
