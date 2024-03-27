package session

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// ReadMiddleWare sessions.Defaultを呼び出す
func ReadMiddleWare(c *gin.Context) sessions.Session {
	session := sessions.Default(c)
	return session
}

// Manager セッションマネージャを定義するインターフェース
type SessionManager interface {
	Get(*gin.Context) SessionManager
	Set(*gin.Context) error
	Destroy(*gin.Context) error
}

// Login ログイン情報セッション保存
type Login struct {
	ID    int
	Login bool
}

// Get セッションから値を取得 => 構造体に格納
func (l *Login) Get(c *gin.Context) SessionManager {
	session := ReadMiddleWare(c)
	memberID := session.Get("ID")
	if memberID != "" && memberID != nil {
		l.ID = memberID.(int)
	}
	login := session.Get("login")
	if login != "" && login != nil {
		l.Login = login.(bool)
	}
	return l
}

// Set 構造体を受け取る => セッションに各値を格納
func (l *Login) Set(c *gin.Context) error {
	session := ReadMiddleWare(c)
	session.Set("ID", l.ID)
	session.Set("login", l.Login)
	// Setしたセッション情報を保存
	if err := session.Save(); err != nil {
		return err
	}
	return nil
}

// Destroy セッションを削除
func (l *Login) Destroy(c *gin.Context) error {
	session := ReadMiddleWare(c)
	session.Delete("ID")
	session.Delete("login")
	// セッション情報の変更を保存
	if err := session.Save(); err != nil {
		return err
	}
	return nil
}
