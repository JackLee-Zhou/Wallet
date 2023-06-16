package server

import (
	"github.com/gin-gonic/gin"
	"github.com/lmxdawn/wallet/db"
	"github.com/lmxdawn/wallet/engine"
	"net/http"
)

// AuthRequired 认证中间件
func AuthRequired() gin.HandlerFunc {

	return func(c *gin.Context) {

		token := c.GetHeader("Account")
		if token == "" {
			APIResponse(c, ErrToken, nil)
			c.Abort()
			return
		}
		if !db.CheckLoginInfo(token) {

			APIResponse(c, ErrLoginExpire, nil)
			c.Abort()
			return
		}
		// 检验有效期
		//session :=sessions.Default(c)
		//session.

	}

}

// SetEngine 设置db数据库
func SetEngine(engines ...*engine.ConCurrentEngine) gin.HandlerFunc {

	return func(c *gin.Context) {
		for _, currentEngine := range engines {
			// TODO 换掉这种处理方式
			// 跨请求取值 方便在其他地方使用
			c.Set(currentEngine.Protocol+currentEngine.CoinName, currentEngine)
			c.Set(currentEngine.Config.Rpc, currentEngine)
		}
	}
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", "*") // 可将将 * 替换为指定的域名
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
			c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}
