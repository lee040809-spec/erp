package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Output-statistics users already report qualified output through history entry.
// Normal completion uses that same reporting permission, without granting finance or process-edit rights.
var productionConfirmationModules = []string{"production_orders", "production_dashboard", "monthly_output"}

func canConfirmProduction(db *gorm.DB, c *gin.Context) bool {
	return hasModuleAccess(db, c, productionConfirmationModules...)
}
func requireProductionConfirmation(db *gorm.DB, c *gin.Context) bool {
	return requireModuleAccess(db, c, productionConfirmationModules...)
}
