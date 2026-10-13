package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const RequestBodyLimit int64 = 20 << 20
const ExcelUploadLimit int64 = 10 << 20

// Bound concurrent decoding and expensive spreadsheet work. Excess requests
// receive a retryable response rather than accumulating an unbounded queue.
func ResourceLimits() gin.HandlerFunc {
	requests := make(chan struct{}, 32)
	spreadsheets := make(chan struct{}, 2)
	return func(c *gin.Context) {
		select {
		case requests <- struct{}{}:
			defer func() { <-requests }()
		default:
			c.Header("Retry-After", "2")
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "服务繁忙，请稍后重试"})
			return
		}
		bodyLimit := RequestBodyLimit
		if c.Request.Method == http.MethodPost && c.Request.URL.Path == "/api/platform/drawings" {
			bodyLimit += 1 << 20 // Multipart headers must not reduce the advertised 20MB file limit.
		}
		if c.Request.ContentLength > bodyLimit {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "请求内容超过20MB，请减少附件或分批提交"})
			return
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, bodyLimit)
		}
		if c.Request.Method == http.MethodPost && strings.Contains(c.Request.URL.Path, "excel") {
			select {
			case spreadsheets <- struct{}{}:
				defer func() { <-spreadsheets }()
			default:
				c.Header("Retry-After", "2")
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "表格正在处理，请稍后重试"})
				return
			}
		}
		defer func() {
			if c.Request.MultipartForm != nil {
				_ = c.Request.MultipartForm.RemoveAll()
			}
		}()
		c.Next()
	}
}
