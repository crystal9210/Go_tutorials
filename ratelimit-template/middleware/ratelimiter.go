package middleware

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"

	"github.com/gin-gonic/gin"
)

// リミッター構造体
type Limiter struct {
	limiter *rate.Limiter
}

// キー(string型、クライアントのIPアドレス)と値(Limiterへのポインタ)のペアを格納するコレクションlimiterMapの宣言
// マップはキーと値のペアを格納するデータ構造で、キーを通じて迅速に値を検索することができる
var limiterMap = make(map[string]*Limiter)

// スレッドセーフ用変数
var mtx sync.Mutex

func NewLimiter(r rate.Limit, b int) *Limiter {
	return &Limiter{
		limiter: rate.NewLimiter(r, b),
	}
}

// ミドルウェアとしての実装のメソッド
// (メソッドの返り値)	gin.HandlerFunc：Ginフレームワークでリクエストを処理するためのハンドラ関数の型、生成後リクエストが来るたび呼び出され処理を行う
func RateLimiterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// ☆リクエストを処理する前に、mtx（sync.Mutex型の変数）をロック→複数のリクエストが同時に処理される際の競合を防ぐため
		mtx.Lock()

		// Go言語のマップにアクセスする際、mapの値と、そのキーが存在するかどうかをチェックするための2つの値を受け取ることができる
		if _, exists := limiterMap[c.ClientIP()]; !exists {
			// 変更後: limiterMap[c.ClientIP()] = NewLimiter(5, 5) と、バケットサイズも同じ数に設定
			if _, exists := limiterMap[c.ClientIP()]; !exists {
				// 5秒間に5リクエストを許容するように変更
				limiterMap[c.ClientIP()] = NewLimiter(rate.Limit(1), 5)
				// NewLimiter():
				// 第一引数：レート(rate)；許可されるイベント数を指定、rate.Limit(n/m):m秒間にn回のイベント数を処理できるように指定
				// 第二引数：バースト(burst)；短期間に許可される最大イベント数を指定、瞬間的なリクエストの増加に対応できる

				// 参考：rate.Limiterは作成時に指定されたレート（秒間のリクエスト数）とバーストサイズ（瞬間的に許可するリクエストの最大数）に基づいて動作し、これらの値はそのインスタンスに対して固定されます。つまり、レートやバーストの値を変更したい場合は、新しいrate.Limiterインスタンスを作成する必要があります。

				// 【重要】バースト値は、トークンバケットレートリミッターにおける一種の"容量"と考えることができます。バースト値bが設定されている場合、リミッターは最大b個のリクエストを即座に（つまり、ほぼ同時に）許可する能力を持ちます。ただし、これらのリクエストが処理された後、リミッターのトークン（または「許可」）は、指定されたレートrに従って徐々に再生成されます。

				// つまり、一度バースト上限を超過すると、その後のリクエストは再びレートrに従ってのみ許可されます。ただし、バーストが適用されなくなるわけではありません。リミッターのトークンは時間とともに再び蓄積され、バースト値bまで回復することができます。このプロセスにより、リミッターは定期的にバーストトラフィックを許可する能力を回復します。
			}

			// // ここで新しいレートリミッターを作成。例として、ここでは 1秒に1リクエスト
			// limiterMap[c.ClientIP()] = NewLimiter(1, 1)
		}
		mtx.Unlock()

		// 現在のクライアントIPアドレスに対応するリミッターを取得する
		limiter := limiterMap[c.ClientIP()]

		// Allow():リクエストがレートに従って許可される場合true、レートリミットに達した場合falseを返す
		if !limiter.limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
			return
		}

		c.Next()
	}
}
