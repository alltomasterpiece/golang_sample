package middlewares

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const headerXRequestID = "X-Request-ID"
const headerXTransactionID = "X-Transaction-ID"
const headerJWTToken = "X-JWT"

var jwtSigningToken = os.Getenv("JWT_SIGNING_TOKEN") // @todo change to KTOKEN

type MiddlewareFunc func(c *gin.Context)

func AttachRequestIDs(c *gin.Context) {
	// use assigned reqId from Heroku router; else generate one
	reqID := c.GetHeader(headerXTransactionID)
	if reqID == "" {
		reqID = uuid.New().String()
	}
	// if transactionId provided; use it
	trxID := c.GetHeader(headerXTransactionID)
	if trxID == "" {
		trxID = uuid.New().String()
	}
	c.Header(headerXRequestID, reqID)
	c.Header(headerXTransactionID, trxID)
	c.Next()
}

func AccessLogger(c *gin.Context) {
	out := os.Stdout

	// Start timer
	start := time.Now()
	path := c.Request.URL.Path
	raw := c.Request.URL.RawQuery

	// Process request
	c.Next()

	// Stop timer
	timeStamp := time.Now()
	latency := timeStamp.Sub(start)

	clientIP := c.ClientIP()
	method := c.Request.Method
	statusCode := c.Writer.Status()

	bodySize := c.Writer.Size()

	if raw != "" {
		path = path + "?" + raw
	}

	env := os.Getenv("ENV")
	if env == "dev" {
		s := fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\"\n",
			clientIP,
			timeStamp.Format(time.RFC1123),
			method,
			path,
			c.Request.Proto,
			statusCode,
			latency,
			c.Request.UserAgent(),
		)
		fmt.Fprint(out, s)
		return
	}
	m := map[string]interface{}{
		"clientIP":      clientIP,
		"ts":            timeStamp.Format(time.RFC1123),
		"method":        method,
		"path":          path,
		"protocol":      c.Request.Proto,
		"status":        statusCode,
		"latency":       latency.Milliseconds(),
		"userAgent":     c.Request.UserAgent(),
		"bodyBytes":     bodySize,
		"requestId":     c.Writer.Header().Get(headerXRequestID),
		"transactionId": c.Writer.Header().Get(headerXTransactionID),
	}
	b, err := json.Marshal(m)
	if err != nil {
		s := fmt.Sprintf("%+v", m) + "\n"
		fmt.Fprint(out, s)
		return
	}
	fmt.Fprint(out, string(b)+"\n")
}
