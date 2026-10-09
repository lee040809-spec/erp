package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"strconv"
)

func referenceID(c *gin.Context) (int64, bool) {
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id <= 0 {
		c.JSON(400, gin.H{"error": "编号无效"})
		return 0, false
	}
	return id, true
}
func referenceError(c *gin.Context, e error) {
	if errors.Is(e, gorm.ErrRecordNotFound) {
		c.JSON(404, gin.H{"error": "记录不存在或不属于当前公司"})
		return
	}
	c.JSON(500, gin.H{"error": "资料操作失败，请核对数据后重试"})
}
