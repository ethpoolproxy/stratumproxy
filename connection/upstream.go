package connection

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	uuid "github.com/iris-contrib/go.uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
	"net"
	"stratumproxy/config"
	"strings"
	"sync"
	"time"
)

var ErrUpstreamInvalidUser = errors.New("矿池身份验证失败: 请检查钱包/用户名")

// UpstreamClient 逻辑
type UpstreamClient struct {
	Uuid       string
	Config     config.Upstream
	PoolServer *PoolServer

	Connection net.Conn
	reader     *bufio.Reader

	jobQueue     []string
	jobQueueLock *sync.RWMutex

	// ProtocolData 记录协议的一些数据
	ProtocolData *sync.Map

	WorkerMiner          *WorkerMiner
	DownstreamClient     *DownstreamClient
	DownstreamIdentifier MinerIdentifier

	shutdownWaiter *sync.WaitGroup

	Disconnected bool
	terminate    bool
}

func (client *UpstreamClient) SetJobQueue(queue []string) {
	client.jobQueueLock.Lock()
	defer client.jobQueueLock.Unlock()
	client.jobQueue = queue
}

func (client *UpstreamClient) GetJobQueue() *[]string {
	client.jobQueueLock.Lock()
	defer client.jobQueueLock.Unlock()
	return &client.jobQueue
}

func (client *UpstreamClient) GetJobIndex(job string) int {
	client.jobQueueLock.Lock()
	defer client.jobQueueLock.Unlock()

	for i, s := range client.jobQueue {
		if s == job {
			return i
		}
	}

	return -1
}

func (client *UpstreamClient) HasJob(job string) bool {
	return client.GetJobIndex(job) != -1
}

func (client *UpstreamClient) AddJob(job string) {
	client.jobQueueLock.Lock()
	defer client.jobQueueLock.Unlock()

	if len(client.jobQueue)+1 > 80 {
		copy(client.jobQueue, client.jobQueue[1:])
		client.jobQueue = client.jobQueue[:len(client.jobQueue)-1]
	}

	client.jobQueue = append(client.jobQueue, job)
}

// DoneJob 把已经提交过的放到数组的第一位等待回收
func (client *UpstreamClient) DoneJob(job string) {
	// 不能在这里 wait 因为要这个方法本来就是阻塞的
	index := client.GetJobIndex(job)
	if index == -1 {
		return
	}

	client.jobQueueLock.Lock()
	defer client.jobQueueLock.Unlock()

	tmp := client.jobQueue[index]
	copy(client.jobQueue[index:], client.jobQueue[index+1:])
	client.jobQueue[len(client.jobQueue)-1] = tmp
}

func (client *UpstreamClient) ReadOnce(timeout int) ([]byte, error) {
	err := client.Connection.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Second))
	if err != nil {
		return []byte(""), err
	}
	return client.reader.ReadBytes('\n')
}

func (client *UpstreamClient) Write(in []byte) error {
	if !strings.HasSuffix(string(in), "\n") {
		in = append(in, '\n')
	}
	err := client.Connection.SetWriteDeadline(time.Now().Add(8 * time.Second))
	if err != nil {
		return err
	}

	_, err = client.Connection.Write(in)
	return err
}

func (client *UpstreamClient) AuthInitial(id MinerIdentifier) error {
	err := client.PoolServer.Protocol.InitialUpstreamAuth(client, id)
	if err != nil {
		return err
	}

	// 启动同步携程
	go client.SyncTick()

	return nil
}

func (client *UpstreamClient) Shutdown() {
	if client.PoolServer.Err != nil {
		client.terminate = true
	}

	if client.DownstreamClient != nil && client.DownstreamClient.Disconnected {
		client.terminate = true
	}

	if client.Disconnected {
		return
	}
	client.Disconnected = true

	_ = client.Connection.Close()
	log.Debugf("[%s][%s][shutdown] 上游已关闭!", client.PoolServer.Config.Name, client.Uuid)

	if client.terminate {
		return
	}

	log.Infof("[%s][%s][shutdown] 上游开始自动重连...", client.PoolServer.Config.Name, client.Uuid)
	client.Reconnect()
	log.Infof("[%s][%s][shutdown] 上游自动重连成功!", client.PoolServer.Config.Name, client.Uuid)

	return
}

func (client *UpstreamClient) SyncTick() {
	client.shutdownWaiter.Add(1)
	client.Disconnected = false

	defer func() {
		client.shutdownWaiter.Done()
		client.Shutdown()
		PanicHandler()
	}()

	for {
		if client.terminate {
			return
		}
		if client.Disconnected {
			return
		}
		err := client.processRead()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed") {
				return
			}
			log.Warnf("[%s][%s][SyncTick] 读取上游数据失败: %s", client.PoolServer.Config.Name, client.Uuid, err)
			return
		}
	}
}

func (client *UpstreamClient) processRead() error {
	data, err := client.ReadOnce(30)

	if err != nil {
		return err
	}

	if len(data) > 0 {
		log.Tracef("[%s][processRead] 接收到上游数据: %s", client.Connection.RemoteAddr(), data)
		UpstreamInjector.processMsg(client, data)
	}

	return nil
}

func (client *UpstreamClient) Reconnect() {
	// 无论如何都自动重连
	for true {
		err := client.CreateConn()
		if err != nil {
			log.Warnf("[%s][Reconnect] 连接到上游服务器失败: %s", client.Uuid, err)
			time.Sleep(2 * time.Second)
			continue
		}

		err = client.PoolServer.Protocol.InitialUpstreamConn(client)
		if err != nil {
			log.Warnf("[%s][Reconnect] 与上游矿池握手失败: %s", client.Uuid, err)
			time.Sleep(2 * time.Second)
			continue
		}

		if client.DownstreamIdentifier.Wallet != "" {
			err = client.AuthInitial(client.DownstreamIdentifier)
			if err != nil {
				log.Warnf("[%s][Reconnect] 无法登录上游矿池: %s", client.Uuid, err)
				time.Sleep(2 * time.Second)
				continue
			}
		} else {
			log.Warnf("[%s][Reconnect] 上游不存在认证信息，取消重连!", client.Uuid)
			if client.DownstreamClient != nil {
				client.DownstreamClient.Shutdown()
			}
			client.terminate = true
			return
		}

		break
	}
}

func (client *UpstreamClient) CreateConn() error {
	var c net.Conn
	dialer := &net.Dialer{
		Timeout: 12 * time.Second,
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: 12 * time.Second,
				}
				return d.DialContext(ctx, "udp", "8.8.4.4:53")
			},
		},
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	if client.Config.Tls {
		if client.Config.Proxy != "" {
			proxyDialer, err := proxy.SOCKS5("tcp", client.Config.Proxy, nil, proxy.Direct)
			if err != nil {
				return err
			}
			c, err = proxyDialer.Dial("tcp", client.Config.Address)
			if err != nil {
				return err
			}
			c = tls.Client(c, tlsConfig)
		} else {
			var err error
			c, err = tls.Dial("tcp", client.Config.Address, tlsConfig)
			if err != nil {
				return err
			}
		}
	} else {
		if client.Config.Proxy != "" {
			proxyDialer, err := proxy.SOCKS5("tcp", client.Config.Proxy, nil, proxy.Direct)
			if err != nil {
				return err
			}
			c, err = proxyDialer.Dial("tcp", client.Config.Address)
			if err != nil {
				return err
			}
		} else {
			var err error
			c, err = dialer.Dial("tcp", client.Config.Address)
			if err != nil {
				return err
			}
		}
	}

	client.Connection = c
	client.reader = bufio.NewReader(c)

	return nil
}

func NewUpstreamClient(pool *PoolServer, cfg config.Upstream) (*UpstreamClient, error) {
	id, _ := uuid.NewV4()
	client := &UpstreamClient{
		Uuid:       id.String(),
		Config:     cfg,
		PoolServer: pool,

		jobQueue:     make([]string, 0, 84),
		jobQueueLock: &sync.RWMutex{},

		ProtocolData: &sync.Map{},

		shutdownWaiter: &sync.WaitGroup{},
	}

	err := client.CreateConn()
	if err != nil {
		return nil, errors.New("连接失败 " + err.Error())
	}

	err = client.PoolServer.Protocol.InitialUpstreamConn(client)
	if err != nil {
		return nil, errors.New("初始化连接失败 " + err.Error())
	}

	return client, nil
}
