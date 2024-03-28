package main

import (
	"fmt"
	"net/http"
	"ratelimit-template/middleware"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// すべてのリクエストにセッション管理機能を適用
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

	// ログイン機能をシミュレートするハンドラー
	router.GET("/login", func(c *gin.Context) {
		// セッションの取得
		session := sessions.Default(c)
		// セッションにユーザーIDを設定
		session.Set("userID", "12345")
		if err := session.Save(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
			return
		}
		fmt.Println("Logged in user ID:", session.Get("userID"))
		c.JSON(http.StatusOK, gin.H{"message": "You are logged in"})
	})

	// ルートパスのハンドラー
	router.GET("/", func(c *gin.Context) {
		// セッションからユーザーIDを取得
		session := sessions.Default(c)
		userID := session.Get("userID")

		// ターミナルにセッションデータを出力
		fmt.Printf("Session user ID: %v\n", userID)

		c.JSON(http.StatusOK, gin.H{"message": "Welcome to the secure app!", "userID": userID})
	})

	router.Run(":8080")
}
