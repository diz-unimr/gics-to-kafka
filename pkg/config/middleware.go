package config

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"time"
)

func LoggingMiddleware(log *logrus.Logger) gin.HandlerFunc {
	if log == nil {
		log = logrus.StandardLogger()
	}
	return func(c *gin.Context) {
		// Starting time request
		startTime := time.Now()

		// Processing request
		c.Next()

		// End Time request
		endTime := time.Now()

		// execution time
		latencyTime := endTime.Sub(startTime)

		// Request method
		reqMethod := c.Request.Method

		// Request route
		reqUri := c.Request.RequestURI

		// status code
		statusCode := c.Writer.Status()

		// Request IP
		clientIP := c.ClientIP()

		entry := log.WithFields(logrus.Fields{
			"method":    reqMethod,
			"uri":       reqUri,
			"status":    statusCode,
			"latency":   latencyTime,
			"client_ip": clientIP,
		})

		if c.Writer.Status() >= 500 {
			entry.Error(c.Errors.String())
		} else {
			entry.Info("HTTP request")
		}

		c.Next()
	}
}
