package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"github.com/kataras/iris/v12/context"
	"net/http"
	"stratumproxy/config"
)

func BasicAuth(ctx *context.Context) {
	username, password, ok := ctx.Request().BasicAuth()

	if ok {
		usernameHash := sha256.Sum256([]byte(username))
		passwordHash := sha256.Sum256([]byte(password))
		expectedUsernameHash := sha256.Sum256([]byte(config.GlobalConfig.WebUI.Auth.Username))
		expectedPasswordHash := sha256.Sum256([]byte(config.GlobalConfig.WebUI.Auth.Passwd))

		usernameMatch := subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1
		passwordMatch := subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1

		if usernameMatch && passwordMatch {
			user := &context.SimpleUser{
				ID:       username,
				Username: username,
				Password: password,
			}
			_ = ctx.SetUser(user)
			ctx.Next()
			return
		}
	}
	ctx.Header("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
	ctx.StopWithStatus(http.StatusUnauthorized)
}
