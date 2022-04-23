package webui

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"net/http"
	"stratumproxy/config"
	"stratumproxy/webui/api"
	"stratumproxy/webui/middleware"
)

func StartWebServer() error {
	app := iris.New()

	app.Use(middleware.BasicAuth)

	app.Handle(iris.MethodGet, "/assets/*", iris.FileServer(http.FS(assets), iris.DirOptions{Compress: true}))
	app.Handle(iris.MethodGet, "/", func(context *context.Context) { context.Redirect("/dashboard", 302) })

	app.Handle(iris.MethodGet, "/dashboard", pageDashboard)

	app.Handle(iris.MethodGet, "/pool", pagePoolManger)
	app.Handle(iris.MethodGet, "/pool/create", pagePoolAdd)
	app.Handle(iris.MethodGet, "/pool/edit/{name:string}", pagePoolEdit)
	app.Handle(iris.MethodGet, "/pool/worker/{name:string}", pageWorkerList)

	app.Handle(iris.MethodGet, "/cfg/edit", pageCfgEdit)

	/**
	API
	*/
	app.Handle(iris.MethodPost, "/api/v1/pool/create", api.PoolCreate)
	app.Handle(iris.MethodPost, "/api/v1/pool/edit", api.PoolEdit)
	app.Handle(iris.MethodGet, "/api/v1/pool/delete/{name:string}", api.PoolDelete)
	app.Handle(iris.MethodGet, "/api/v1/pool/power/{action:string}/{name:string}", api.PoolPower)

	app.Handle(iris.MethodPost, "/api/v1/cfg/auth", api.CfgAuthEdit)

	err := app.Listen(config.GlobalConfig.WebUI.Bind, iris.WithConfiguration(iris.Configuration{
		LogLevel:          "info",
		DisableStartupLog: true,
	}))
	if err != nil {
		return err
	}

	return nil
}
