package main

import (
	"job_portal/common"
	"job_portal/session"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	csrf "github.com/utrack/gin-csrf"
)

// CSRFエラー時のハンドラー関数を定義
func ErrorCSRF(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token mismatch"})
}

func main() {
	// Ginのインスタンスを作成
	e := gin.Default()

	// セッションストアを作成;セッションデータをクライアントのブラウザにCookieとして保存するためのストアを作成
	store := cookie.NewStore([]byte("secret"))
	e.Use(sessions.Sessions("mysession", store))
	// // セッションのストアを作成
	// store := sessions.NewCookieStore([]byte("secret"))
	// e.Use(sessions.Sessions("mysession", store))

	// CSRFトークンの設定
	e.Use(csrf.Middleware(csrf.Options{
		Secret:    common.GenerateString(32),
		ErrorFunc: ErrorCSRF,
	}))

	// ルーティング
	e.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})

	// CSRFトークンの取得エンドポイントの実装
	e.GET("/csrf-token", func(c *gin.Context) {
		csrfToken := csrf.GetToken(c)
		c.JSON(http.StatusOK, gin.H{"csrfToken": csrfToken})
	})

	// ログイン処理
	e.POST("/login", func(c *gin.Context) {
		// ログイン処理
		// セッションにログイン情報を保存
		login := &session.Login{ID: 1, Login: true}
		err := login.Set(c)
		if err != nil {
			// エラーハンドリング
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
		// ログイン成功時の処理
		c.JSON(http.StatusOK, gin.H{"message": "Login Successful"})
	})

	// ログアウト処理
	e.POST("/logout", func(c *gin.Context) {
		// セッションからログイン情報を削除
		login := &session.Login{}
		err := login.Destroy(c)
		if err != nil {
			// エラーハンドリング
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
		// ログアウト成功時の処理
		c.JSON(http.StatusOK, gin.H{"message": "Logout Successful"})
	})

	// HTTPサーバーを起動
	e.Run(":8080")
}
