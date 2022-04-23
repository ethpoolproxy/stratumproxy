package webui

import (
	"github.com/kataras/iris/v12/context"
	log "github.com/sirupsen/logrus"
	"stratumproxy/connection"
)

// pageWorkerList /pool/worker/{name}
func pageWorkerList(ctx *context.Context) {
	pool, ok := connection.PoolServers.Load(ctx.Params().Get("name"))
	if !ok {
		page := page{
			Pages:  []string{"layout/base", "page/code_404"},
			Writer: ctx.ResponseWriter(),
			Data:   nil,
		}

		err := page.Build()
		if err != nil {
			log.Errorf("[pageWorkerList] 无法解析模板: %s", err)
			return
		}
	}

	type dataWorkerList struct {
		PoolServer *connection.PoolServer
	}

	data := dataWorkerList{
		PoolServer: pool.(*connection.PoolServer),
	}

	page := page{
		Pages:  []string{"layout/base", "page/pool_worker_list"},
		Writer: ctx.ResponseWriter(),
		Data:   data,
	}

	err := page.Build()
	if err != nil {
		log.Errorf("[pageWorkerList] 无法解析模板: %s", err)
		return
	}
}
