package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// BoundedPage protects both allocations and integer multiplication in OFFSET.
func boundedPage(c *gin.Context, defaultSize int) (page, size, offset int) {
	page, _ = strconv.Atoi(c.Query("page"))
	size, _ = strconv.Atoi(c.Query("pageSize"))
	if page < 1 {
		page = 1
	}
	if page > 1000000 {
		page = 1000000
	}
	if size < 1 {
		size = defaultSize
	}
	if size > 100 {
		size = 100
	}
	return page, size, (page - 1) * size
}

func trimPage[T any](rows []T, size int) ([]T, bool) {
	more := len(rows) > size
	if more {
		rows = rows[:size]
	}
	return rows, more
}
