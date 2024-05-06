package token

import (
	"fmt"
	"time"

	"github.com/aead/chacha20poly1305"
	"github.com/o1egl/paseto"
)

// PasetoMaker is a PASETO token maker
type PasetoMaker struct {
	paseto       *paseto.V2
	symmetricKey []byte
}

// PasetoMakerにmaker.goファイル内のMakerインターフェースを継承させる

// メソッドの実装*2

// ペイロードからトークンを生成するメソッドの定義
func (maker *PasetoMaker) CreateToken(username string, role string, duration time.Duration) (string, *Payload, error) {
	payload, err := NewPayload(username, role, duration)
	if err != nil {
		// 関数の返り値の制約を満たすために空文字を返すように実装
		return "", payload, err
	}
	token, err := maker.paseto.Encrypt(maker.symmetricKey, payload, nil)
	return token, payload, err
}

// トークンが有効かどうかを検証するメソッドの定義
func (maker *PasetoMaker) VerifyToken(token string) (*Payload, error) {
	// Payload構造体のインスタンスを作成し、そのポインタをpayload変数に格納している
	payload := &Payload{}

	// token:復号化するトークン,maker.symmetricKey:トークンを復号化するために使用する対称鍵、payload:復号化されたデータを格納するための変数
	err := maker.paseto.Decrypt(token, maker.symmetricKey, payload, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}

	err = payload.Valid()
	if err != nil {
		return nil, err
	}

	return payload, nil
}

// NewPasetoMaker creates a new PASETO token maker
func NewPasetoMaker(symmetricKey string) (Maker, error) {
	if len(symmetricKey) < chacha20poly1305.KeySize {
		return nil, fmt.Errorf("invalid key size: must be at least %d characters", chacha20poly1305.KeySize)
	}
	maker := &PasetoMaker{
		paseto:       paseto.NewV2(),
		symmetricKey: []byte(symmetricKey),
	}

	return maker, nil
}
