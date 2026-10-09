package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func TestResourceLimitsBodies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ResourceLimits())
	r.POST("/body", func(c *gin.Context) {
		_, err := io.Copy(io.Discard, c.Request.Body)
		var limit *http.MaxBytesError
		if errors.As(err, &limit) {
			c.Status(413)
			return
		}
		c.Status(200)
	})
	for _, declared := range []bool{true, false} {
		req := httptest.NewRequest("POST", "/body", io.LimitReader(zeroReader{}, RequestBodyLimit+1))
		if declared {
			req.ContentLength = RequestBodyLimit + 1
		} else {
			req.ContentLength = -1
		}
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		if res.Code != 413 {
			t.Fatalf("oversize declared=%v status=%d", declared, res.Code)
		}
	}
	res := httptest.NewRecorder()
	r.ServeHTTP(res, httptest.NewRequest("POST", "/body", strings.NewReader("small")))
	if res.Code != 200 {
		t.Fatal("limit did not recover")
	}
}

func TestResourceLimitsConcurrentExcel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ResourceLimits())
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	done := make(chan struct{}, 2)
	r.POST("/import-excel", func(c *gin.Context) { entered <- struct{}{}; <-release; c.Status(200) })
	for i := 0; i < 2; i++ {
		go func() {
			r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/import-excel", nil))
			done <- struct{}{}
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("request did not enter")
		}
	}
	res := httptest.NewRecorder()
	r.ServeHTTP(res, httptest.NewRequest("POST", "/import-excel", nil))
	close(release)
	<-done
	<-done
	if res.Code != 503 || res.Header().Get("Retry-After") == "" {
		t.Fatal("concurrent spreadsheet limit failed")
	}
	res = httptest.NewRecorder()
	r.ServeHTTP(res, httptest.NewRequest("POST", "/import-excel", nil))
	if res.Code != 200 {
		t.Fatal("slot leaked")
	}
}
