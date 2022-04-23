package webui

import (
	"github.com/kataras/iris/v12/context"
	log "github.com/sirupsen/logrus"
	"stratumproxy/config"
	"stratumproxy/connection"
)

func pagePoolEdit(ctx *context.Context) {
	type dataPoolEdit struct {
		Icon    string
		Title   string
		Action  string
		PoolCfg config.Pool
	}

	pool, ok := connection.PoolServers.Load(ctx.Params().Get("name"))
	if !ok {
		ctx.Redirect("/pool/create", 302)
		return
	}

	var page = page{
		Pages:  []string{"layout/base", "page/pool_form"},
		Writer: ctx.ResponseWriter(),
		Data: dataPoolEdit{
			Icon:    "fa-pencil",
			Title:   " 修改矿池",
			Action:  "edit",
			PoolCfg: *pool.(*connection.PoolServer).Config,
		},
	}

	err := page.Build()
	if err != nil {
		log.Errorf("[pagePoolManger] 无法解析模板: %s", err)
		return
	}
}
