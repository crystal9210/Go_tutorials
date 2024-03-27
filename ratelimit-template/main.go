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
	// セッションデータをクッキーに保存するためのストアを作成
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("mysession", store))

	// セキュリティヘッダーの設定
	router.Use(func(c *gin.Context) {
		//	ページが自身のドメインからのリソースのみをロードすることをブラウザに指示、、クロスサイトスクリプティング攻撃(XSS)などのセキュリティリスクを軽減
		c.Header("Content-Security-Policy", "default-src 'self'")
		// "X-Frame-Options", "DENY"：クリックジャッキング防止；他のサイトがこのページを<iframe>内に表示することを防ぐ
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "no-referrer")
	})

	// レートリミティングのミドルウェアをrouterが扱う全てのリクエストに適用
	// →認証、ロギング、CORSポリシーの適用、レートリミティングなどの共通の処理をリクエストの処理チェーンに挿入できる
	router.Use(middleware.RateLimiterMiddleware())

	// ルートパスのハンドラー
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Welcome to the secure app!"})
	})

	router.Run(":8080")
}
