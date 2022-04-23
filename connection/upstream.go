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

type UpstreamClient struct {
	Uuid       string
	PoolServer *PoolServer

	Config config.Upstream

	Connection net.Conn
	reader     *bufio.Reader

	Context *sync.Map

	InjectorWaiter *sync.WaitGroup

	LastJobAt    time.Time
	jobQueue     []string
	JobQueueLock *sync.RWMutex

	DownstreamClient     *DownstreamClient
	DownstreamIdentifier MinerIdentifier

	IsShutdown     bool
	IsReconnecting bool
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

	if len(client.jobQueue)+1 > 40 {
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

func (client *UpstreamClient) Write(in []byte) error {
	if !strings.HasSuffix(string(in), "\n") {
		in = append(in, '\n')
	}
	_, err := client.Connection.Write(in)
	return err
}

func (client *UpstreamClient) Shutdown() {
	if client == nil {
		return
	}

	client.IsShutdown = true
	_ = client.Connection.Close()
}

func (client *UpstreamClient) readOnce() ([]byte, error) {
	return client.reader.ReadBytes('\n')
}

func (client *UpstreamClient) sendKeepAlive() {
	for {
		_, err := client.Connection.Write([]byte(""))
		if err != nil {
			client.Shutdown()
			return
		}
		time.Sleep(8 * time.Second)
	}
}

// watchDog 检测是不是没下发任务 | 重启后退出当前的
func (client *UpstreamClient) watchDog() {
	for !client.IsShutdown {
		time.Sleep(10 * time.Second)

		if time.Since(client.LastJobAt).Seconds() < 30 {
			continue
		}

		if !client.IsReconnecting && !client.IsShutdown {
			log.Warnf("[%s][WatchDog][%s] 上游在30秒内没发送过任务！开始重启...", client.PoolServer.Config.Name, client.Uuid)
			client.Reconnect()
			log.Infof("[%s][WatchDog][%s] 上游重启成功!", client.PoolServer.Config.Name, client.Uuid)

			// 重启成功退出这个携程 因为已经开了一个
			return
		}
	}
}

func (client *UpstreamClient) Reconnect() {
	if client.IsShutdown {
		return
	}

	if client.IsReconnecting {
		return
	}

	client.IsReconnecting = true
	client.Shutdown()

	// 无论如何都自动重连
	var err error
	var conn net.Conn
	for err != nil || conn == nil {
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

	// 启动携程读
	go client.processRead()
	go client.sendKeepAlive()
	go client.watchDog()

	client.IsShutdown = false
	client.IsReconnecting = false
}

func (client *UpstreamClient) processRead() {
	defer PanicHandler()

	for {
		data, err := client.readOnce()
		if err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "use of closed network connection") {
				log.Debugf("[%s][processRead] 上游断开连接!", client.Connection.RemoteAddr())
				break
			} else {
				log.Debugf("[%s][processRead] 读取上游数据失败: %s", client.Connection.RemoteAddr(), err.Error())
				break
			}
		}
		// 别有事没事瞎叫唤
		if len(data) > 0 {
			UpstreamInjector.processMsg(client, data)
		}
	}

	if !client.IsReconnecting && !client.IsShutdown {
		client.Reconnect()
		log.Infof("[processRead][AutoReconnect][%s] 上游自动重连成功!", client.Uuid)
	}
}

func (client *UpstreamClient) RequestJob() error {
	err := client.Write([]byte("{\"id\":5,\"method\":\"eth_getWork\",\"params\":[]}\n"))
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
		err := client.Write(json)
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
	client := &UpstreamClient{
		Uuid:       id.String(),
		PoolServer: pool,

		Config:               upstream,
		DownstreamIdentifier: identifier,

		Connection: conn,
		reader:     bufio.NewReader(conn),

		Context: &sync.Map{},

		InjectorWaiter: &sync.WaitGroup{},

		jobQueue:     make([]string, 0, 42),
		JobQueueLock: &sync.RWMutex{},
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
