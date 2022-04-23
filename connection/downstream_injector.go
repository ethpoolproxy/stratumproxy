package connection

import (
	"errors"
	"github.com/sirupsen/logrus"
	"strings"
	"syscall"
)

var DownstreamInjector = &InjectorDownstream{}

type InjectorDownstream struct{}

// InjectorDownstreamPayload 这个东西里面装着要传递的东西，仅限单个数据包的链式处理
type InjectorDownstreamPayload struct {
	// 连接
	DownstreamClient *DownstreamClient

	// 跟 bukkit 那个 isCancelled 同理
	// 当某个 Processor 设置了返回值的时候结束后续处理
	IsTerminated bool

	// 要断开下游吗
	ShouldShutdown bool
	ForceShutdown  bool

	// 最后返回的数据 可为空
	Transmission []byte

	// 在各个 Processor 中传递的数据
	// 会被用作最终发送到上游的数据
	In []byte

	// 返回的数据
	Out []byte
}

// processMsg 链式地处理消息
func (injector *InjectorDownstream) processMsg(client *DownstreamClient, in []byte) {
	payload := &InjectorDownstreamPayload{
		In:               in,
		DownstreamClient: client,
	}

	defer client.InjectorWaiter.Done()
	client.InjectorWaiter.Wait()
	client.InjectorWaiter.Add(1)
	for _, p := range client.Connection.PoolServer.Protocol.DownstreamInjectorProcessors {
		p(payload)
		// 如果这次的处理结果设置了终止标志，则终止后续处理
		if payload.IsTerminated {
			if len(payload.Out) > 0 {
				// 就把这些东西设定成当前的,因为当前要求终止
				err := client.Write(payload.Out)
				if err != nil && !errors.Is(err, syscall.EPIPE) && !strings.Contains(err.Error(), "use of closed network connection") {
					logrus.Errorf("[%s][ProcessMsg] 在终止时发送数据失败 [%s]", client.Connection.Conn.RemoteAddr(), err.Error())
				}
			}
			if payload.ShouldShutdown {
				if payload.ForceShutdown {
					client.ForceShutdown()
				} else {
					client.Shutdown()
				}
			}
			return
		}
	}
	return
}
