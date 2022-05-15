package api

import (
	"github.com/kataras/iris/v12/context"
	"stratumproxy/config"
)

// CfgAuthEdit 修改认证 POST /api/v1/cfg/auth
func CfgAuthEdit(ctx *context.Context) {
	type requestStruct struct {
		Username string `json:"username"`
		Passwd   string `json:"passwd"`
	}

	var request requestStruct
	err := ctx.ReadJSON(&request)
	if err != nil {
		_, _ = ctx.JSON(ResponseAPI{
			Result: false,
			Msg:    "未知错误，请反馈: " + err.Error(),
		})
		return
	}

	config.GlobalConfig.WebUI.Auth.Username = request.Username
	config.GlobalConfig.WebUI.Auth.Passwd = request.Passwd

	_ = config.SaveConfig(config.ConfigFile)

	_, _ = ctx.JSON(ResponseAPI{
		Result: true,
		Msg:    "管理员认证信息修改成功！",
	})
}
