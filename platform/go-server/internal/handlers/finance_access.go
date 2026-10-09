package handlers

import (
	"github.com/gin-gonic/gin"
	"machining-erp/pkg/response"
)

// Historical production reporting follows output-statistics permission, independently of cost administration.
func (h *ProductionFinanceHandler) canImportHistory(c *gin.Context) bool {
	return hasModuleAccess(h.db, c, "monthly_output", "btn-manage-cost-entry")
}
func (h *ProductionFinanceHandler) canManageFinanceCosts(c *gin.Context) bool {
	return hasModuleAccess(h.db, c, "btn-manage-cost-entry")
}
func (h *ProductionFinanceHandler) requireHistoryWriteAccess(c *gin.Context) bool {
	if h.canImportHistory(c) {
		return true
	}
	response.Forbidden(c, response.CodeForbidden, "当前账号无历史完工补录权限，请由管理员分配产值统计权限")
	return false
}
func (h *ProductionFinanceHandler) GetFinanceAccess(c *gin.Context) {
	if !h.requireFinanceAccess(c) {
		return
	}
	response.Success(c, gin.H{"can_import_history": h.canImportHistory(c), "can_manage_costs": h.canManageFinanceCosts(c)})
}
