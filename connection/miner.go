package connection

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type DownstreamClientMutexWrapper struct {
	sync.RWMutex
	clients []*DownstreamClient
}

func (d *DownstreamClientMutexWrapper) Add(c *DownstreamClient) {
	d.Lock()
	defer d.Unlock()

	d.clients = append(d.clients, c)
}

func (d *DownstreamClientMutexWrapper) GetIndex(c *DownstreamClient) int {
	d.Lock()
	defer d.Unlock()

	for i, client := range d.clients {
		if client == c {
			return i
		}
	}

	return -1
}

func (d *DownstreamClientMutexWrapper) Copy() *[]*DownstreamClient {
	d.Lock()
	defer d.Unlock()

	result := make([]*DownstreamClient, len(d.clients))
	copy(result, d.clients)
	return &result
}

func (d *DownstreamClientMutexWrapper) Contains(c *DownstreamClient) bool {
	return d.GetIndex(c) != -1
}

func (d *DownstreamClientMutexWrapper) Remove(c *DownstreamClient) {
	index := d.GetIndex(c)
	if index == -1 {
		return
	}

	d.Lock()
	defer d.Unlock()

	var result []*DownstreamClient
	result = append(result, d.clients[:index]...)
	result = append(result, d.clients[index+1:]...)
	d.clients = result
}

type MinerIdentifier struct {
	Wallet     string
	WorkerName string
}

type WorkerMinerSliceWrapper struct {
	sync.RWMutex
	workerMiner []*WorkerMiner
}

func (wrapper *WorkerMinerSliceWrapper) Copy() *[]*WorkerMiner {
	wrapper.Lock()
	defer wrapper.Unlock()

	result := make([]*WorkerMiner, len(wrapper.workerMiner))
	copy(result, wrapper.workerMiner)
	return &result
}

func (wrapper *WorkerMinerSliceWrapper) CopyRange(f func(i int, m *WorkerMiner) bool) {
	for i, miner := range *wrapper.Copy() {
		if !f(i, miner) {
			return
		}
	}
}

func (wrapper *WorkerMinerSliceWrapper) GetJobIndex(dw *WorkerMiner) int {
	wrapper.Lock()
	defer wrapper.Unlock()

	for i, w := range wrapper.workerMiner {
		if w == dw {
			return i
		}
	}

	return -1
}

func (wrapper *WorkerMinerSliceWrapper) HasMiner(w *WorkerMiner) bool {
	return wrapper.GetJobIndex(w) != -1
}

func (wrapper *WorkerMinerSliceWrapper) Add(w *WorkerMiner) {
	wrapper.Lock()
	defer wrapper.Unlock()

	wrapper.workerMiner = append(wrapper.workerMiner, w)
}

func (wrapper *WorkerMinerSliceWrapper) Remove(m *WorkerMiner) {
	index := wrapper.GetJobIndex(m)
	if index == -1 {
		return
	}

	wrapper.Lock()
	defer wrapper.Unlock()

	if index < len(wrapper.workerMiner) {
		copy(wrapper.workerMiner[index:], wrapper.workerMiner[index+1:])
	}

	wrapper.workerMiner[len(wrapper.workerMiner)-1] = nil
	wrapper.workerMiner = wrapper.workerMiner[:len(wrapper.workerMiner)-1]
}

type WalletMiner struct {
	Wallet string

	// Clients map[workerName]*WorkerMiner
	Clients *sync.Map
}

func (w *WalletMiner) GetOnlineWorkerList() *[]*WorkerMiner {
	list := make([]*WorkerMiner, 0)
	w.Clients.Range(func(key, value interface{}) bool {
		if value.(*WorkerMiner).IsOnline() {
			list = append(list, value.(*WorkerMiner))
		}
		return true
	})
	return &list
}

func (w *WalletMiner) GetWorkerList() *[]*WorkerMiner {
	list := make([]*WorkerMiner, 0)
	w.Clients.Range(func(key, value interface{}) bool {
		list = append(list, value.(*WorkerMiner))
		return true
	})
	return &list
}

type WorkerMiner struct {
	// 最后一次连接时间
	ConnectAt time.Time

	// 最后一次提交时间
	LastShareAt time.Time

	PoolServer *PoolServer
	Identifier *MinerIdentifier

	HashRate   int64
	TotalShare int64
	FeeShare   int64

	TimeIntervalShareStats *ShareStatsIntervalMap

	FeeInstance    []*FeeStatesClient
	CurFeeInstance *FeeStatesClient

	DropUpstream bool

	// 底下的连接对
	DownstreamClients *DownstreamClientMutexWrapper
}

func (m *WorkerMiner) IsOnline() bool {
	return len(*m.DownstreamClients.Copy()) > 0
}

func (m *WorkerMiner) AddShare(d int64) {
	atomic.AddInt64(&m.TotalShare, d)
	m.TimeIntervalShareStats.AddShare(d)
	m.LastShareAt = time.Now()
}

func (m *WorkerMiner) AddFeeShare(d int64) {
	atomic.AddInt64(&m.FeeShare, d)
}

// GetHashrateInMhs Hash/s -> MH/s
func (m *WorkerMiner) GetHashrateInMhs() float64 {
	result, err := strconv.ParseFloat(fmt.Sprintf("%.2f", float64(m.HashRate)/1000000), 64)
	if err != nil {
		return 0
	}
	return result
}

func (m *WorkerMiner) GetID() string {
	return m.Identifier.Wallet + "." + m.Identifier.WorkerName
}

func (m *WorkerMiner) FindFeeInfoByFeeUpstream(upC *UpstreamClient) *FeeStatesClient {
	var result *FeeStatesClient

	for _, fee := range m.FeeInstance {
		if fee.UpstreamClient == upC {
			result = fee
			break
		}
	}

	return result
}

type ShareStatsIntervalMap struct {
	sync.RWMutex
	sync.Map
}

func (s *ShareStatsIntervalMap) AddShare(d int64) {
	s.Range(func(_, stats interface{}) bool {
		stats.(*ShareStatsInterval).AddShare(d)
		return true
	})
}

func (s *ShareStatsIntervalMap) GetStats(duration time.Duration) *ShareStatsInterval {
	val, _ := s.LoadOrStore(duration, NewShareStatsInterval(duration))
	return val.(*ShareStatsInterval)
}

func (s *ShareStatsIntervalMap) AddStats(stats *ShareStatsInterval) {
	defer s.Unlock()
	s.Lock()
	s.Store(stats.interval, stats)
}

func (s *ShareStatsIntervalMap) AddStatsSlice(stats *[]*ShareStatsInterval) {
	defer s.Unlock()
	s.Lock()
	for _, interval := range *(stats) {
		s.Store(interval.interval, interval)
	}
}

// ShareStatsInterval 一个时间段内的份额统计
type ShareStatsInterval struct {
	share           int64
	interval        time.Duration
	intervalStartAt time.Time
}

func NewShareStatsInterval(interval time.Duration) *ShareStatsInterval {
	return &ShareStatsInterval{
		interval:        interval,
		intervalStartAt: time.Now(),
	}
}

func (s *ShareStatsInterval) Update() {
	if time.Since(s.intervalStartAt).Seconds() > s.interval.Seconds() {
		s.share = 0
		s.intervalStartAt = time.Now()
		return
	}
}

func (s *ShareStatsInterval) GetShare() int64 {
	s.Update()
	return s.share
}

func (s *ShareStatsInterval) AddShare(d int64) {
	s.Update()
	atomic.AddInt64(&s.share, d)
}
