package webui

import (
	"github.com/kataras/iris/v12/context"
	log "github.com/sirupsen/logrus"
	"stratumproxy/connection"
)

func pagePoolManger(ctx *context.Context) {
	type datePoolManger struct {
		PoolServers []*connection.PoolServer
	}

	data := datePoolManger{}

	connection.PoolServers.Range(func(_, s interface{}) bool {
		data.PoolServers = append(data.PoolServers, s.(*connection.PoolServer))
		return true
	})

	page := page{
		Pages:  []string{"layout/base", "page/pool_manager"},
		Writer: ctx.ResponseWriter(),
		Data:   data,
	}

	err := page.Build()
	if err != nil {
		log.Errorf("[pagePoolManger] 无法解析模板: %s", err)
		return
	}
}
