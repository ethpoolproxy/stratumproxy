package connection

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	uuid "github.com/iris-contrib/go.uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"stratumproxy/config"
	"stratumproxy/protocol/eth"
	"strings"
	"sync"
	"time"
)

// UpstreamClient 逻辑
// 创建时启动: processRead | KeepAlive | watchDog
// processRead: 循环读取 出错关掉上游
// KeepAlive: 发送空数据包 错误关掉上游
// watchDog: 检查上游有没有在发送份额
type UpstreamClient struct {
	Uuid       string
	PoolServer *PoolServer

	Config config.Upstream

	Connection net.Conn
	reader     *bufio.Reader

	LastJobAt    time.Time
	jobQueue     []string
	JobQueueLock *sync.RWMutex

	DownstreamClient     *DownstreamClient
	DownstreamIdentifier MinerIdentifier

	Ctx             context.Context
	CtxShutdown     context.CancelFunc
	ShutdownWaiter  *sync.WaitGroup
	SafeWriteWaiter *sync.WaitGroup

	shutdownOnce  *sync.Once
	reconnectOnce *sync.Once

	terminated   bool
	Disconnected bool
}

func (client *UpstreamClient) SetJobQueue(queue []string) {
	client.JobQueueLock.Lock()
	defer client.JobQueueLock.Unlock()
	client.jobQueue = queue
}

func (client *UpstreamClient) GetJobQueue() *[]string {
	client.JobQueueLock.Lock()
	defer client.JobQueueLock.Unlock()
	return &client.jobQueue
}

func (client *UpstreamClient) GetJobIndex(job string) int {
	client.JobQueueLock.Lock()
	defer client.JobQueueLock.Unlock()

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
	client.JobQueueLock.Lock()
	defer client.JobQueueLock.Unlock()

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

	client.JobQueueLock.Lock()
	defer client.JobQueueLock.Unlock()

	tmp := client.jobQueue[index]
	copy(client.jobQueue[index:], client.jobQueue[index+1:])
	client.jobQueue[len(client.jobQueue)-1] = tmp
}

func (client *UpstreamClient) SafeWrite(in []byte) {
	client.SafeWriteWaiter.Add(1)
	defer client.SafeWriteWaiter.Done()

	start := time.Now()
	for client.write(in) != nil {
		if client.Disconnected {
			return
		}

		if time.Since(start).Seconds() > 10 {
			log.Debugf("[%s][%s][SafeWrite] 发送超时，放弃数据包: %s", client.PoolServer.Config.Name, client.Uuid, string(in))
			return
		}

		log.Debugf("[%s][%s][SafeWrite] 等待上游重连...", client.PoolServer.Config.Name, client.Uuid)
		time.Sleep(500 * time.Millisecond)
	}
}

func (client *UpstreamClient) write(in []byte) error {
	if !strings.HasSuffix(string(in), "\n") {
		in = append(in, '\n')
	}
	_, err := client.Connection.Write(in)
	return err
}

func (client *UpstreamClient) Shutdown() {
	client.shutdownOnce.Do(func() {
		if client.Ctx.Err() != nil {
			return
		}

		if !client.Disconnected {
			client.SafeWriteWaiter.Wait()
		}

		client.shutdown()

		if client.PoolServer.Err != nil {
			return
		}

		// 查看下游状态
		if client.DownstreamClient == nil || !client.DownstreamClient.Disconnected {
			// 重启连接
			log.Infof("[%s][%s][shutdown] 上游开始自动重连...", client.PoolServer.Config.Name, client.Uuid)
			client.reconnectOnce.Do(client.Reconnect)
			log.Infof("[%s][%s][shutdown] 上游自动重连成功!", client.PoolServer.Config.Name, client.Uuid)
			return
		}

		client.terminated = true
	})
}

func (client *UpstreamClient) shutdown() {
	client.CtxShutdown()
	client.ShutdownWaiter.Wait()
	_ = client.Connection.Close()
	log.Debugf("[%s][%s][shutdown] 上游已关闭!", client.PoolServer.Config.Name, client.Uuid)
}

func (client *UpstreamClient) readOnce() ([]byte, error) {
	return client.reader.ReadBytes('\n')
}

func (client *UpstreamClient) sendKeepAlive() {
	client.ShutdownWaiter.Add(1)

	defer func() {
		client.ShutdownWaiter.Done()
	}()

	var err error
loop:
	for {
		select {
		case <-client.Ctx.Done():
			break loop
		case <-time.After(8 * time.Second):
			_, err = client.Connection.Write([]byte(""))
			if err != nil {
				break loop
			}
		}
	}

	if err != nil {
		log.Warnf("[%s][%s][sendKeepAlive] 上游发送心跳包错误: %s", client.PoolServer.Config.Name, client.Uuid, err)
	} else {
		log.Debugf("[%s][%s][sendKeepAlive] 上游停止发送心跳包!", client.PoolServer.Config.Name, client.Uuid)
	}
	go client.Shutdown()
}

// watchDog 检测是不是没下发任务 | 重启后退出当前的
func (client *UpstreamClient) watchDog() {
	client.ShutdownWaiter.Add(1)

	defer func() {
		log.Debugf("[%s][%s][watchDog] 上游监测停止!", client.PoolServer.Config.Name, client.Uuid)
		client.ShutdownWaiter.Done()
	}()

	for {
		select {
		case <-client.Ctx.Done():
			return
		case <-time.After(10 * time.Second):
			if time.Since(client.LastJobAt).Seconds() < 30 {
				continue
			}

			log.Warnf("[%s][%s][WatchDog] 上游在30秒内没发送过任务!", client.PoolServer.Config.Name, client.Uuid)
			go client.Shutdown()
			return
		}
	}
}

func (client *UpstreamClient) Reconnect() {
	// 无论如何都自动重连
	var err error
	var conn net.Conn
	for err != nil || conn == nil {
		if client.terminated {
			log.Debugf("[Reconnect] 上游关闭取消重连!")
			return
		}

		conn, err = newUpstreamConn(client.Config, 8)
		if err != nil {
			log.Warnf("[Reconnect] 连接到上游服务器失败: %s", err)
			time.Sleep(2 * time.Second)
			continue
		}

		// 直接替换连接 比较 hacky
		client.Connection = conn
		client.reader = bufio.NewReader(conn)

		err = client.SendAuth()
		if err != nil {
			log.Warnf("[Reconnect] 无法登陆上游矿池: %s", err)
			time.Sleep(2 * time.Second)
			continue
		}

		err = client.RequestJob()
		if err != nil {
			log.Warnf("[Reconnect] 无法从上游矿池获取任务: %s", err)
			time.Sleep(2 * time.Second)
			continue
		}
	}

	client.Ctx, client.CtxShutdown = context.WithCancel(context.Background())
	client.shutdownOnce = &sync.Once{}
	client.reconnectOnce = &sync.Once{}

	// 启动携程读
	go client.processRead()
	go client.sendKeepAlive()
	go client.watchDog()
}

func (client *UpstreamClient) processRead() {
	client.ShutdownWaiter.Add(1)

	defer func() {
		log.Debugf("[%s][%s][processRead] 上游停止读取!", client.PoolServer.Config.Name, client.Uuid)
		PanicHandler()
		client.Disconnected = true
		client.ShutdownWaiter.Done()
		go client.Shutdown()
	}()

	type readOnce struct {
		data []byte
		err  error
	}

	readCh := make(chan readOnce)

	for {
		go func() {
			err := client.Connection.SetReadDeadline(time.Now().Add(32 * time.Second))
			if err != nil {
				readCh <- readOnce{
					data: []byte(""),
					err:  err,
				}
				return
			}

			d, e := client.readOnce()
			readCh <- readOnce{
				data: d,
				err:  e,
			}
		}()

		select {
		case <-client.Ctx.Done():
			return
		case result := <-readCh:
			if client.DownstreamClient != nil && client.DownstreamClient.Disconnected {
				return
			}

			if result.err != nil {
				if result.err == io.EOF || strings.Contains(result.err.Error(), "use of closed network connection") {
					return
				} else {
					log.Warnf("[%s][%s][processRead] 读取上游数据失败: %s", client.PoolServer.Config.Name, client.Uuid, result.err)
					return
				}
			}
			// 别有事没事瞎叫唤
			if len(result.data) > 0 {
				UpstreamInjector.processMsg(client, result.data)
			}
		}
	}
}

func (client *UpstreamClient) RequestJob() error {
	err := client.write([]byte("{\"id\":5,\"method\":\"eth_getWork\",\"params\":[]}\n"))
	if err != nil {
		return err
	}
	return nil
}

var UpstreamInvalidUserErr = errors.New("抽水矿池身份验证失败: 请检查钱包/用户名")

func (client *UpstreamClient) SendAuth() error {
	errCh := make(chan error, 1)

	go func() {
		json := []byte("{\"compact\":true,\"id\":1,\"method\":\"eth_submitLogin\",\"params\":[\"" + client.DownstreamIdentifier.Wallet + "\",\"\"],\"worker\":\"" + client.DownstreamIdentifier.WorkerName + "\"}\n")
		err := client.write(json)
		if err != nil {
			errCh <- err
			return
		}

		// 等待登陆返回
		data, err := client.readOnce()
		if err != nil {
			errCh <- err
			return
		}

		// 验证登陆包
		var loginResponse eth.ResponseSubmitLogin
		err = loginResponse.Parse(data)
		if err != nil {
			errCh <- err
			return
		}

		// 验证返回是否成功
		if !loginResponse.Result {
			if strings.Contains(loginResponse.Error, "Invalid user") || strings.Contains(loginResponse.Error, "Bad user name") {
				errCh <- UpstreamInvalidUserErr
				return
			}

			errCh <- errors.New(loginResponse.Error)
			return
		}

		errCh <- nil
	}()

	select {
	case <-time.After(6 * time.Second):
		return errors.New("登陆上游矿池超时")
	case err := <-errCh:
		return err
	}
}

func newUpstreamConn(upstream config.Upstream, timeout int) (net.Conn, error) {
	type newUpstreamConnResult struct {
		c   net.Conn
		err error
	}

	newUpstreamConnCh := make(chan newUpstreamConnResult, 1)

	go func() {
		var c net.Conn
		dialer := &net.Dialer{
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
		if upstream.Tls {
			if upstream.Proxy != "" {
				proxyDialer, err := proxy.SOCKS5("tcp", upstream.Proxy, nil, proxy.Direct)
				if err != nil {
					newUpstreamConnCh <- newUpstreamConnResult{c: nil, err: err}
					return
				}
				c, err = proxyDialer.Dial("tcp", upstream.Address)
				if err != nil {
					newUpstreamConnCh <- newUpstreamConnResult{c: nil, err: err}
					return
				}
				c = tls.Client(c, tlsConfig)
			} else {
				var err error
				c, err = tls.Dial("tcp", upstream.Address, tlsConfig)
				if err != nil {
					newUpstreamConnCh <- newUpstreamConnResult{c: nil, err: err}
					return
				}
			}
		} else {
			if upstream.Proxy != "" {
				proxyDialer, err := proxy.SOCKS5("tcp", upstream.Proxy, nil, proxy.Direct)
				if err != nil {
					newUpstreamConnCh <- newUpstreamConnResult{c: nil, err: err}
					return
				}
				c, err = proxyDialer.Dial("tcp", upstream.Address)
				if err != nil {
					newUpstreamConnCh <- newUpstreamConnResult{c: nil, err: err}
					return
				}
			} else {
				var err error
				c, err = dialer.Dial("tcp", upstream.Address)
				if err != nil {
					newUpstreamConnCh <- newUpstreamConnResult{c: nil, err: err}
					return
				}
			}
		}

		newUpstreamConnCh <- newUpstreamConnResult{c: c, err: nil}
	}()

	timeoutCh := time.After(time.Duration(timeout) * time.Second)
	select {
	case <-timeoutCh:
		return nil, errors.New("上游连接超时")
	case result := <-newUpstreamConnCh:
		return result.c, result.err
	}
}

func NewUpstreamClient(pool *PoolServer, upstream config.Upstream, identifier MinerIdentifier) (*UpstreamClient, error) {
	conn, err := newUpstreamConn(upstream, 8)
	if err != nil {
		return nil, err
	}

	id, _ := uuid.NewV4()
	ctx, terminate := context.WithCancel(context.Background())
	client := &UpstreamClient{
		Uuid:       id.String(),
		PoolServer: pool,

		Config:               upstream,
		DownstreamIdentifier: identifier,

		Connection: conn,
		reader:     bufio.NewReader(conn),

		jobQueue:     make([]string, 0, 82),
		JobQueueLock: &sync.RWMutex{},

		Ctx:             ctx,
		CtxShutdown:     terminate,
		ShutdownWaiter:  &sync.WaitGroup{},
		SafeWriteWaiter: &sync.WaitGroup{},

		shutdownOnce:  &sync.Once{},
		reconnectOnce: &sync.Once{},
	}

	// 尝试登陆 有报错则退出
	err = client.SendAuth()
	if err != nil {
		return nil, err
	}

	// 请求工作
	err = client.RequestJob()
	if err != nil {
		return nil, err
	}

	go client.processRead()
	go client.sendKeepAlive()
	go client.watchDog()

	return client, nil
}
