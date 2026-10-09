package handlers

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"machining-erp/internal/middleware"
)

func TestExcelUploadSizeBeforeDatabase(t *testing.T) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("file", "large.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	chunk := make([]byte, 64<<10)
	for left := middleware.ExcelUploadLimit + 1; left > 0; {
		n := int64(len(chunk))
		if n > left {
			n = left
		}
		if _, err = io.CopyN(file, bytes.NewReader(chunk), n); err != nil {
			t.Fatal(err)
		}
		left -= n
	}
	if err = form.Close(); err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(middleware.ResourceLimits())
	h := &PrintTemplateHandler{}
	r.POST("/import-excel", h.ImportExcel)
	req := httptest.NewRequest("POST", "/import-excel", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != 413 {
		t.Fatalf("large file reached database: %d %s", res.Code, res.Body.String())
	}
}
