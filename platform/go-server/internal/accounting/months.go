package accounting

import "gorm.io/gorm"

// LockMonthStatus creates and locks the company/month row within the caller's
// transaction. Callers retain their own rules for open and closed months.
func LockMonthStatus(tx *gorm.DB, corp int64, month string) (string, error) {
	if err := tx.Exec("INSERT IGNORE INTO monthly_closings(corp_id,accounting_month,status,rule_version) VALUES (?,?,'open','v1')", corp, month).Error; err != nil {
		return "", err
	}
	var row struct{ Status string }
	if err := tx.Raw("SELECT status FROM monthly_closings WHERE corp_id=? AND accounting_month=? FOR UPDATE", corp, month).Scan(&row).Error; err != nil {
		return "", err
	}
	return row.Status, nil
}
