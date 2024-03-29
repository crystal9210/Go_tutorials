package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type LoginForm struct {
	Username string `form:"username"`
	Password string `form:"password"`
}

// ユーザーのログイン処理を行うハンドラ
func loginHandler(c *gin.Context) {
	// ユーザー認証のロジック...

	// 認証成功後、新しいcsrfSaltを生成してセッションに保存
	session := sessions.Default(c)
	csrfSalt, err := GenerateRandomString(32) // 新しい CSRF salt を生成
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate CSRF salt"})
		return
	}
	session.Set(csrfSalt, csrfSalt) // CSRF salt をセッションに保存
	session.Save()

	c.JSON(http.StatusOK, gin.H{"message": "Login successful", "csrfToken": tokenize("your-csrf-secret-here", csrfSalt)})
}

// カスタムのトークン取得関数、httpリクエストからcsrfトークンを取得
func customTokenGetter(c *gin.Context) string {
	// ヘッダーからトークンを試みる
	token := c.GetHeader("X-CSRF-Token")
	if token != "" {
		return token
	}
	// フォームデータからトークンを試みる
	token = c.PostForm("csrf_token")
	return token
}

// csrfトークン取得関数、httpリクエストからcsrfトークンを抽出
// テンプレ的にリクエストのヘッダーやフォームデータからトークンを取得するシンプルな実装
func defaultTokenGetter(c *gin.Context) string {
	// ヘッダーからトークン取得をtry
	token := c.GetHeader("X-CSRF-Token")
	if token != "" {
		token = c.Query("csrf_token")
	}

	// フォームデータからトークン取得をtry
	token = c.PostForm("csrf_token")
	return token

}

// トークンの計算＋正当性のチェックをするメソッド
func tokenize(secret, salt string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(salt))
	return hex.EncodeToString(h.Sum(nil))
}

// 指定された文字列がリスト内(引数arr)に存在するかどうかを確認
func inArray(arr []string, str string) bool {
	for _, v := range arr {
		if v == str {
			return true
		}
	}
	return false
}

// カスタムのエラーハンドラ関数
func customErrorFunc(c *gin.Context) {
	c.String(http.StatusForbidden, "Custom CSRF token mismatch error.")
	c.Abort()
}

// デフォルトのエラー処理関数、csrfトークンの不一致が検出された場合に呼び出し
func defaultErrorFunc(c *gin.Context) {
	c.String(http.StatusForbidden, "CSRF token mismatch") // エラーコード403(；アクセス禁止)を返す
	// c.Abort():Ginフレームワークにおいて、現在処理中のリクエストチェーンを直ちに停止するために使用、現在のHTTPリクエストのコンテキストを表し、Abortメソッドを呼び出すことで、それ以降のミドルウェアやハンドラの実行をスキップする(認証失敗、権限不足などのケースでただちにリクエストを終了させることでセキュリティ上のリスクを即座に回避できるというメリット)
	c.Abort()
}

// GenerateRandomString は指定された長さのランダムな文字列を生成する
func GenerateRandomString(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
