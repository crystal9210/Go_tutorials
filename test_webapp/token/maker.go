package token

import (
	"time"
)

// トークン作成のメソッドのインターフェース用ファイル

// Maker is a interface for managing tokens
type Maker interface {
	// CreateToken creates a new token with a specific username and duration
	CreateToken(username string, role string, duration time.Duration) (string, *Payload, error)
	// VerifyToken chrcks if the token is valid or not
	VerifyToken(token string) (*Payload, error)
}
