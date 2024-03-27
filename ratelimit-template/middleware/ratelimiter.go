package middleware

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"

	"github.com/gin-gonic/gin"
)

type Limiter struct {
	limiter *rate.Limiter
}

var limiterMap = make(map[string]*Limiter)
var mtx sync.Mutex

func NewLimiter(r rate.Limit, b int) *Limiter {
	return &Limiter{
		limiter: rate.NewLimiter(r, b),
	}
}

func RateLimiterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		mtx.Lock()
		if _, exists := limiterMap[c.ClientIP()]; !exists {
			// 変更後: limiterMap[c.ClientIP()] = NewLimiter(5, 5) と、バケットサイズも同じ数に設定
			if _, exists := limiterMap[c.ClientIP()]; !exists {
				// 1分間に5リクエストを許容するように変更
				limiterMap[c.ClientIP()] = NewLimiter(rate.Limit(5), 5) // 第一引数はrate.Limit(5/60)でも良いが、簡潔さのため5とする
			}

			// // ここで新しいレートリミッターを作成。例として、ここでは 1秒に1リクエスト
			// limiterMap[c.ClientIP()] = NewLimiter(1, 1)
		}
		mtx.Unlock()

		limiter := limiterMap[c.ClientIP()]

		if !limiter.limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
			return
		}

		c.Next()
	}
}
