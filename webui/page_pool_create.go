package webui

import (
	"github.com/kataras/iris/v12/context"
	log "github.com/sirupsen/logrus"
	"stratumproxy/config"
)

func pagePoolAdd(ctx *context.Context) {
	type dataPoolAdd struct {
		Icon    string
		Title   string
		Action  string
		PoolCfg config.Pool
	}

	var page = page{
		Pages:  []string{"layout/base", "page/pool_form"},
		Writer: ctx.ResponseWriter(),
		Data: dataPoolAdd{
			Icon:    "fa-plus",
			Title:   " 添加矿池",
			Action:  "create",
			PoolCfg: config.Pool{},
		},
	}
	err := page.Build()
	if err != nil {
		log.Errorf("[pagePoolManger] 无法解析模板: %s", err)
		return
	}
}
