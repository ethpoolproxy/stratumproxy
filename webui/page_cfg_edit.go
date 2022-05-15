package webui

import (
	"github.com/kataras/iris/v12/context"
	log "github.com/sirupsen/logrus"
	"stratumproxy/config"
)

func pageCfgEdit(context *context.Context) {
	page := page{
		Pages:  []string{"layout/base", "page/cfg_edit"},
		Writer: context.ResponseWriter(),
		Data:   config.GlobalConfig,
	}

	err := page.Build()
	if err != nil {
		log.Errorf("[pageCfgEdit] 无法解析模板: %s", err)
		return
	}
}
