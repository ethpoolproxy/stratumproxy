package connection

import (
	"bufio"
	"github.com/goccy/go-json"
	log "github.com/sirupsen/logrus"
	"io"
	"strings"
	"sync"
)

type DownstreamClient struct {
	Connection *PoolConn

	AuthPackSent bool

	InjectorWaiter *sync.WaitGroup

	WalletMiner *WalletMiner
	WorkerMiner *WorkerMiner
	Upstream    *UpstreamClient
}

func (client *DownstreamClient) Write(b []byte) error {
	if !strings.HasSuffix(string(b), "\n") {
		b = append(b, '\n')
	}
	_, err := client.Connection.Conn.Write(b)
	return err
}

func (client *DownstreamClient) processRead() {
	defer PanicHandler()

	reader := bufio.NewReader(client.Connection.Conn)
	var err error
	for {
		var data []byte
		data, err = reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "use of closed network connection") {
				log.Debugf("[%s][processRead] 下游断开连接!", client.Connection.Conn.RemoteAddr())
				break
			} else if strings.Contains(err.Error(), "tls:") {
				log.Debugf("[%s][processRead] 客户端未使用 SSL 连接: %s", client.Connection.Conn.RemoteAddr(), err.Error())
				break
			} else {
				log.Debugf("[%s][processRead] 读取下游数据失败: %s", client.Connection.Conn.RemoteAddr(), err.Error())
				break
			}
		}
		// 别有事没事瞎叫唤
		if len(data) > 0 {
			// 验证是不是 json
			if !json.Valid(data) {
				// 不断开连接 丢弃就是了
				log.Debugf("[%s][DownInjectorEthSubmitLogin] $马玩意能不能不要扫 | Raw: %s", client.Connection.Conn.RemoteAddr(), data)
				continue
			}

			log.Tracef("[%s][processRead] 接收到下游数据: %s", client.Connection.Conn.RemoteAddr(), data)

			DownstreamInjector.processMsg(client, data)
		}
	}

	client.AuthPackSent = false
	client.Shutdown()
}

func (client *DownstreamClient) Shutdown() {
	if client.Connection.Conn != nil {
		_ = client.Connection.Conn.Close()
	}

	if client.Upstream != nil {
		client.Upstream.Shutdown()
	}

	if client.WorkerMiner != nil {
		client.WorkerMiner.HashRate = 0
		client.WorkerMiner.DownstreamClients.Remove(client)

		// 去掉抽水
		client.WorkerMiner.DropUpstream = false
		for _, feeInstance := range client.WorkerMiner.PoolServer.FeeInstance {
			feeWorkerMinersObj, ok := client.WorkerMiner.PoolServer.WorkerMinerFeeDB.Load(feeInstance)
			if ok {
				feeWorkerMinersObj.(*WorkerMinerSliceWrapper).Remove(client.WorkerMiner)
			}
		}
	}

	client.Connection.PoolServer.Protocol.HandleDownstreamDisconnect(client)
}

func (client *DownstreamClient) ForceShutdown() {
	if client.Upstream != nil {
		client.Upstream.Shutdown()
	}

	_ = client.Connection.Conn.Close()
}

func NewDownstreamClient(c *PoolConn) *DownstreamClient {
	instance := &DownstreamClient{
		Connection:     c,
		AuthPackSent:   false,
		InjectorWaiter: &sync.WaitGroup{},
	}

	go instance.processRead()

	return instance
}
