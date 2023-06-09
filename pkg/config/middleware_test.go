package config

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggingMiddleware(t *testing.T) {
	testLogger, hook := test.NewNullLogger()
	log.SetOutput(testLogger.Out)

	gin.SetMode(gin.TestMode)
	resp := httptest.NewRecorder()
	c, r := gin.CreateTestContext(resp)

	r.Use(LoggingMiddleware(testLogger))

	r.GET("/test", func(c *gin.Context) {
		c.Status(200)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(resp, c.Request)

	entry := hook.LastEntry()

	assert.Equal(t, testLogger, entry.Logger)
	assert.Equal(t, log.InfoLevel, entry.Level)
	assert.Equal(t, "HTTP request", entry.Message)
	assert.Equal(t, "GET", entry.Data["method"])
	assert.Equal(t, 200, entry.Data["status"])
}
