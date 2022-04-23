package connection

var UpstreamInjector = &InjectorUpstream{}

type InjectorUpstream struct{}

type InjectorProcessorUpstream struct {
	// 如果是抽水上游就不调用这个
	DisableWhenFee bool
	Processors     func(c *InjectorUpstreamPayload)
}

// InjectorUpstreamPayload 这个东西里面装着要传递的东西，仅限单个数据包的链式处理
type InjectorUpstreamPayload struct {
	// 连接
	UpstreamClient *UpstreamClient

	// 在各个 Processor 中传递的数据
	// 会被用作最终发送到下游的数据
	In []byte

	IsCancelled      bool
	ShouldDisconnect bool
}

// processMsg 链式地处理消息
func (injector *InjectorUpstream) processMsg(client *UpstreamClient, in []byte) {
	payload := &InjectorUpstreamPayload{
		In:             in,
		UpstreamClient: client,
	}

	defer client.InjectorWaiter.Done()
	client.InjectorWaiter.Wait()
	client.InjectorWaiter.Add(1)
	for _, p := range client.PoolServer.Protocol.UpstreamInjectorProcessors {
		if p.DisableWhenFee && client.DownstreamClient == nil {
			continue
		}
		p.Processors(payload)
		if payload.IsCancelled {
			if payload.ShouldDisconnect {
				payload.UpstreamClient.Shutdown()
			}
			break
		}
	}

	return
}
