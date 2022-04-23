package api

import (
	"github.com/kataras/iris/v12/context"
	"stratumproxy/config"
	"stratumproxy/connection"
	"strings"
)

// PoolEdit 修改矿池 POST /api/v1/pool/edit
func PoolEdit(ctx *context.Context) {
	var poolCfg config.Pool

	err := ctx.ReadJSON(&poolCfg)
	if err != nil {
		_, _ = ctx.JSON(ResponseAPI{
			Result: false,
			Msg:    "未知错误，请反馈: " + err.Error(),
		})
		return
	}

	err = poolCfg.Validate()
	if err != nil {
		_, _ = ctx.JSON(ResponseAPI{
			Result: false,
			Msg:    err.Error(),
		})
		return
	}

	server, ok := connection.PoolServers.Load(poolCfg.Name)
	if !ok {
		_, _ = ctx.JSON(ResponseAPI{
			Result: false,
			Msg:    "矿池不存在",
		})
		return
	}

	// 删除矿池
	connection.DeletePoolByName(poolCfg.Name)
	server.(*connection.PoolServer).WaitShutdown()

	// 创建
	newPoolServer, err := connection.CreatePool(poolCfg)
	if err != nil {
		_, _ = ctx.JSON(ResponseAPI{
			Result: false,
			Msg:    err.Error(),
		})
		return
	}

	err = newPoolServer.Start()
	if err != nil {
		_, _ = ctx.JSON(ResponseAPI{
			Result: false,
			Msg:    "矿池配置已更新，但是启动失败: " + err.Error(),
		})
		return
	}

	_, _ = ctx.JSON(ResponseAPI{
		Result: true,
		Msg:    "配置更新成功！矿池已启动！",
	})
}

// PoolCreate 创建矿池 POST /api/v1/pool/create
func PoolCreate(ctx *context.Context) {
	var poolCfg config.Pool

	err := ctx.ReadJSON(&poolCfg)
	if err != nil {
		_, _ = ctx.JSON(ResponseAPI{
			Result: false,
			Msg:    "未知错误，请反馈: " + err.Error(),
		})
		return
	}

	err = poolCfg.Validate()
	if err != nil {
		_, _ = ctx.JSON(ResponseAPI{
			Result: false,
			Msg:    err.Error(),
		})
	}

	_, err = connection.CreatePool(poolCfg)
	if err != nil {
		_, _ = ctx.JSON(ResponseAPI{
			Result: false,
			Msg:    err.Error(),
		})
	}

	_, _ = ctx.JSON(ResponseAPI{
		Result: true,
		Msg:    "创建成功！可在仪表盘/管理页面启动!",
	})
}

// PoolDelete 删除矿池 /api/v1/pool/delete/{name:string}
func PoolDelete(ctx *context.Context) {
	connection.DeletePoolByName(ctx.Params().Get("name"))
	_, _ = ctx.JSON(ResponseAPI{
		Result: true,
		Msg:    "删除成功!",
	})
}

// PoolPower 电源管理 /api/v1/pool/power/{action:string}/{name:string}
func PoolPower(ctx *context.Context) {
	action := strings.ToLower(ctx.Params().Get("action"))
	if action != "start" && action != "stop" {
		_, _ = ctx.JSON(ResponseAPI{
			Result: false,
			Msg:    "动作 [" + action + "] 不存在",
		})
		return
	}

	pool, ok := connection.PoolServers.Load(ctx.Params().Get("name"))
	if !ok {
		_, _ = ctx.JSON(ResponseAPI{
			Result: false,
			Msg:    "找不到矿池: " + ctx.Params().Get("name"),
		})
		return
	}

	if action == "start" {
		if pool.(*connection.PoolServer).Err == nil {
			_, _ = ctx.JSON(ResponseAPI{
				Result: false,
				Msg:    "矿池已经在运行",
			})
			return
		}

		err := pool.(*connection.PoolServer).Start()
		if err != nil {
			_, _ = ctx.JSON(ResponseAPI{
				Result: false,
				Msg:    "启动失败: " + err.Error(),
			})
			return
		}

		_, _ = ctx.JSON(ResponseAPI{
			Result: true,
			Msg:    "启动命令发送成功!",
		})
		return
	}

	if action == "stop" {
		if pool.(*connection.PoolServer).Err != nil {
			_, _ = ctx.JSON(ResponseAPI{
				Result: false,
				Msg:    "矿池已经关闭",
			})
			return
		}

		pool.(*connection.PoolServer).Shutdown(nil)
		_, _ = ctx.JSON(ResponseAPI{
			Result: true,
			Msg:    "关闭命令发送成功",
		})
	}
}
