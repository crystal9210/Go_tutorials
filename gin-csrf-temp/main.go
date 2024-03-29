package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	csrf "github.com/utrack/gin-csrf"
)

// csrfのセキュリティ検証に使用する各種情報→これはたぶんハードコーディングをしないようにした方がいい気する
// const (
// 	csrfSalt   = "csrfSalt"   // CSRFトークン生成に使用するsaltキー
// 	csrfSecret = "csrfSecret" // CSRFシークレットキーを保持するコンテキストキー
// )

// AppConfig:上記の変数を設定ファイルから読み込むように修正
type AppConfig struct {
	CSRFSecret    string `json:"csrfSecret"`
	SessionSecret string `json:"sessionSecret"`
}

// loadConfig は指定されたファイルパスから設定を読み込む
func loadConfig(filePath string) (*AppConfig, error) {
	var config AppConfig
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// csrfトークン検証をスキップするhttpメソッドのデフォルトリスト
var defaultIgnoreMethods = []string{"GET", "HEAD", "OPTIONS", "TRACE"}

type Options struct {
	ignoreMethods []string
	ErrorFunc     func(*gin.Context)
	TokenGetter   func(*gin.Context) string
	Secret        string
}

// CSRFシークレットキーをコンテキストに設定するためのキー
const csrfSecretKey = "csrfSecret"

// ginフレームワーク用csrfトークン検証ミドルウェアを定義する関数、トークンが正当な送信元から来ていることを検証
func Middleware(options Options) gin.HandlerFunc {
	// Options構造体から、csrfトークン検証をスキップするhttpメソッドのリストを変数化
	// 未指定の場合はデフォルト値を使用
	if options.ignoreMethods == nil {
		options.ignoreMethods = defaultIgnoreMethods
	}
	if options.ErrorFunc == nil {
		options.ErrorFunc = func(c *gin.Context) {
			c.String(http.StatusForbidden, "CSRF token mismatch")
			c.Abort()
		}
	}
	if options.TokenGetter == nil {
		options.TokenGetter = func(c *gin.Context) string {
			// ヘッダー、クエリ、フォームデータからトークンを取得
			token := c.GetHeader("X-CSRF-Token")
			if token == "" {
				token = c.Query("csrf_token")
			}
			if token == "" {
				token = c.PostForm("csrf_token")
			}
			return token
		}
	}

	// csrfトークン取得関数が指定されていない場合、デフォルト関数を使用
	return func(c *gin.Context) {
		// セッションはユーザごとに各種状態情報をサーバに保持するための機構(確認)
		session := sessions.Default(c)

		var csrfSalt string
		var err error

		// 既存のsaltがセッションに存在するか確認
		if val := session.Get(csrfSalt); val == nil {
			// 存在しない場合は新しいsaltを生成してセッションに保存
			csrfSalt, err = GenerateRandomString(32)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate CSRF salt"})
				return
			}
			session.Set(csrfSalt, csrfSalt)
			session.Save()
		} else {
			csrfSalt = val.(string)
		}

		// 現在のリクエストコンテキストにcsrfシークレットキーを設定
		c.Set(csrfSecretKey, options.Secret)

		// 現在のリクエストメソッドがcsrfトークン検証の対象外として指定されているか(ignoreMethods)を確認
		if inArray(options.ignoreMethods, c.Request.Method) {
			c.Next() // リクエストメソッドが無視リストに含まれている場合、次んのミドルウェアまたはハンドラに処理を移す
			return
		}

		// セッションからCSRFトークン生成に使用されるsaltを取得
		// ☆saltの一意性の担保により、各セッションやリクエストごとに異なるCSRFトークンを生成することができる
		salt, ok := session.Get(csrfSalt).(string)

		// saltが取得できないor空の場合、エラーハンドラを呼び出し
		if !ok || len(salt) == 0 {
			options.ErrorFunc(c)
			return
		}

		// リクエストからcsrfトークンを取得
		token := options.TokenGetter(c)

		// セッションに保存されているsaltと設定されたシークレットキーを使用して既知されるcsrfトークンを計算、リクエストに含まれるトークンと比較、一致しない場合は不正なリクエストとみなす
		if tokenize(options.Secret, salt) != token {
			options.ErrorFunc(c)
			return
		}

		// トークンが一致する場合、または無視するメソッドのリクエストである場合、次のミドルウェアまたはハンドラに処理を遷移
		c.Next()
	}

}

func main() {
	// 設定ファイル読み込み
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// ルーター作成
	r := gin.Default()
	// セッションストア生成、ストアを使用してセッションミドルウェアを設定
	store := cookie.NewStore([]byte(config.SessionSecret))
	r.Use(sessions.Sessions("mysession", store))

	// 環境変数からCSRFシークレットキーを取得
	// csrfSecret := os.Getenv("CSRF_SECRET")
	// if csrfSecret == "" {
	// 	// シークレットキーが設定されていない場合はエラー
	// 	panic("CSRF_SECRET is not set")
	// }

	// CSRFトークン検証ミドルウェアを設定
	r.Use(csrf.Middleware(csrf.Options{
		Secret: config.CSRFSecret,
		ErrorFunc: func(c *gin.Context) {
			c.String(400, "CSRF token mismatch")
		},
	}))
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})

	// csrfトークンを返す
	r.GET("/protected", func(c *gin.Context) {
		c.String(200, csrf.GetToken(c))
	})

	// csrfトークンの有効性を検証する
	r.POST("/protected", func(c *gin.Context) {
		c.String(200, "CSRF token is valid")
	})

	r.Run(":8080")
}
