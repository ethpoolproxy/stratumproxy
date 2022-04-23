package webui

import (
	"github.com/kataras/iris/v12/context"
	log "github.com/sirupsen/logrus"
	"stratumproxy/config"
	"stratumproxy/connection"
	"strconv"
)

func pageDashboard(context *context.Context) {
	type dashboardData struct {
		StartTime        string
		StartTimeStr     string
		Version          string
		BuildTime        string
		MinerCount       int
		OnlineMinerCount int

		PoolServersCount  int
		OnlinePoolServers []*connection.PoolServer
	}
	data := dashboardData{
		Version:      config.GitTag,
		BuildTime:    config.BuildTime,
		StartTime:    strconv.FormatInt(config.StartTime.Unix(), 10),
		StartTimeStr: config.StartTime.Format("2006-01-02 15:04:05"),

		PoolServersCount: len(config.GlobalConfig.Pools),
	}

	connection.PoolServers.Range(func(_, s interface{}) bool {
		data.MinerCount += len(*(s.(*connection.PoolServer).GetWorkerList()))
		data.OnlineMinerCount += len(*(s.(*connection.PoolServer).GetOnlineWorker()))
		data.OnlinePoolServers = append(data.OnlinePoolServers, s.(*connection.PoolServer))
		return true
	})

	// 这里面搞定数据
	page := page{
		Pages:  []string{"layout/base", "page/dashboard"},
		Writer: context.ResponseWriter(),
		Data:   data,
	}

	err := page.Build()
	if err != nil {
		log.Errorf("[pageDashboard] 无法解析控制面板模板: %s", err)
		return
	}
}
