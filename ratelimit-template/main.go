package main

import (
	"net/http"
	"ratelimit-template/middleware"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// セッション管理の設定
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("mysession", store))

	// セキュリティヘッダーの設定
	router.Use(func(c *gin.Context) {
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "no-referrer")
	})

	// レートリミティングのミドルウェアを適用
	router.Use(middleware.RateLimiterMiddleware())

	// ルートパスのハンドラー
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Welcome to the secure app!"})
	})

	router.Run(":8080")
}
