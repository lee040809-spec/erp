package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

func TestBoundedPage(t *testing.T) {
	for _, tc := range []struct {
		query      string
		page, size int
	}{{"", 1, 50}, {"?page=2&pageSize=10", 2, 10}, {"?page=-1&pageSize=-1", 1, 50}, {"?page=999999999&pageSize=999999999", 1000000, 100}} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/"+tc.query, nil)
		p, s, o := boundedPage(c, 50)
		if p != tc.page || s != tc.size || o != (p-1)*s {
			t.Fatalf("bad bounded pagination %s", tc.query)
		}
	}
}
