package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const minSecretKeySize = 32

// JWTMaker is a JSON Web Token maker;二者間で安全な通信を実現するための認証トークン；デジタル署名で検証される
type JWTMaker struct {
	secretKey string
}

// 新しいJWTMakerインスタンスを作成;秘密キーの長さが最小要件を満たしていない場合、エラーを返す、これはセキュリティを確保するために重要
func NewJWTMaker(secretKey string) (*JWTMaker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid key size: must be at least %d characters", minSecretKeySize)
	}
	return &JWTMaker{secretKey}, nil
}

// 特定のユーザ名と有効期間で新しいトークンを作成する、jwt.NewWithClaims関数でペイロードと秘密キーで署名された新しいJWTトークンを生成
func (maker *JWTMaker) CreateToken(username string, role string, duration time.Duration) (string, *Payload, error) {
	payload, err := NewPayload(username, role, duration)
	if err != nil {
		// 関数の返り値の制約を満たすために
		return "", nil, err
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	token, err := jwtToken.SignedString([]byte(maker.secretKey))
	return token, payload, err
}

// 提供されたトークンが有効かどうかを検証、jwt.ParseWithClaims関数でトークンを解析しペイロードを取得、キー関数keyFuncでトークンが期待する署名方法で署名されているかを検証、トークンが無効のとき、適切なエラーが返される
func (maker *JWTMaker) VerifyToken(token string) (*Payload, error) {
	// jwt署名を鑑賞するための関数
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		// 【メモ】
		//　token.Method.(*jwt.SigningMethodHMAC)は、トークンの署名方法がHMACであるかをチェックします。これは型アサーションを使用
		// 型アサーションは、Go言語の特徴の一つであり、インターフェース型の変数が特定の型を持つ値を格納しているかをチェックし、その型の値を取り出すために使用

		// token.Method.(*jwt.SigningMethodHMAC)：トークンの署名アルゴリズムが指定したjwt.SigningMethodHMAC型（つまり、HMACを使用した署名方法）であるかどうかをチェックするために使用
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, ErrInvalidToken
		}
		return []byte(maker.secretKey), nil
	}

	jwtToken, err := jwt.ParseWithClaims(token, &Payload{}, keyFunc)
	if err != nil {
		verr, ok := err.(*jwt.ValidationError)
		if ok && errors.Is(verr.Inner, ErrExpiredToken) {
			// 解析または検証のエラーが期限切れか同課を確認
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	// トークンのクレーム情報を用いてペイロード内容を確認し、実行時エラーを避けるためのバリデーション処理
	payload, ok := jwtToken.Claims.(*Payload)
	if !ok {
		return nil, ErrInvalidToken
	}
	// すべての件しょぐあ成功した場合、抽出されたペイロードを返す。これにより呼び出し元はトークンの内容を使用できるようになる
	return payload, nil
}

// 【参考】トークンのクレーム情報を用いてペイロード内容を確認し、実行時エラーを避けるためのバリデーション処理ということでいいでしょうか
// はい、その通りです。トークンのクレーム情報を用いてペイロードの内容を確認し、実行時エラーを避けるためのバリデーション処理と考えることができます。JWT（JSON Web Token）では、トークンのペイロード部分には認証や認可プロセスに必要な情報が含まれており、この情報をクレームと呼びます。

// クレーム情報には、以下のようなデータが含まれることが一般的です：

// ユーザー識別子（例：ユーザーID）
// トークンの発行者（Issuer）
// トークンの有効期限（Expiration Time）
// トークンの対象者（Subject）
// アクセス権限（例：ユーザーロールや権限レベル）
// クレーム情報を用いたバリデーション処理の目的は、セキュリティを確保し、アプリケーションのロジックを適切に実行するために、トークンが正しい形式であり、期待される値を持っていることを確認することです。このプロセスには、以下のようなステップが含まれます：

// トークンの解析: トークンからペイロードを抽出し、その内容を解析します。
// 型アサーション: 抽出したペイロード（クレーム情報）がアプリケーションが期待する特定のデータ構造（例：ユーザー定義のPayload構造体）に適合するかを確認します。
// クレームのバリデーション: トークンの有効期限が切れていないか、トークンの発行者が正しいかなど、具体的なクレームの値を検証します。
// エラーハンドリング: いずれかのバリデーションチェックに失敗した場合は、適切なエラーメッセージを返し、実行時エラーを避けます。
// このバリデーション処理を行うことで、アプリケーションは不正なトークンを拒否し、認証されたユーザーのみがリソースにアクセスできるように制御することができます。これにより、セキュリティが強化され、アプリケーションの信頼性が向上します。
