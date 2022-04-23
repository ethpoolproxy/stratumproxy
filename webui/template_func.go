package webui

import (
	"html/template"
	"stratumproxy/connection"
	"time"
)

var funcMap = template.FuncMap{
	"format_hashrate": func(hs int64) string {
		return connection.HashrateFormat(float64(hs))
	},
	"format_pool_hashrate": func(pool *connection.PoolServer) string {
		return connection.HashrateFormat(pool.GetMHashrate() * 1000000)
	},
	"get_pool_worker_list": func(pool *connection.PoolServer) *[]*connection.WorkerMiner {
		return pool.GetWorkerList()
	},
	"get_pool_online_worker_list": func(pool *connection.PoolServer) *[]*connection.WorkerMiner {
		return pool.GetOnlineWorker()
	},
	"get_miner_conn": func(m *connection.WorkerMiner) *[]*connection.DownstreamClient {
		return m.GetConn()
	},
	"get_miner_share_stats": func(m *connection.WorkerMiner) []int64 {
		stats := make([]int64, 0, 3)
		stats = append(stats, m.TimeIntervalShareStats.GetStats(15*time.Minute).GetShare())
		stats = append(stats, m.TimeIntervalShareStats.GetStats(30*time.Minute).GetShare())
		stats = append(stats, m.TimeIntervalShareStats.GetStats(60*time.Minute).GetShare())
		return stats
	},
	"unix_time": func(i time.Time) string {
		return i.Format("2006-01-02 15:04:05")
	},
	"f_greater": func(a, b float64) bool {
		return a > b
	},
	"time_since": func(i time.Time) string {
		if i.Unix() == 0 {
			return "-"
		}
		return time.Since(i).String()
	},
}
