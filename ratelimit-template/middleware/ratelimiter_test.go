package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiterMiddleware(t *testing.T) {
	router := gin.New()
	router.Use(RateLimiterMiddleware())

	router.GET("/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)

	// 5秒間に5回までのリクエストを送信
	for i := 0; i < 5; i++ {
		router.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code, "Expected HTTP status code 200 for request #%d", i+1)
		w = httptest.NewRecorder()
	}

	// 6回目のリクエストでレートリミットに達するかをテスト
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "Expected HTTP status code 429 for request #6")

	// レートリミットのリセットを待つ（5秒）
	time.Sleep(5 * time.Second)

	// レートリミット後の最初のリクエストが成功するかテスト
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code, "Expected HTTP status code 200 after rate limit reset")
}
