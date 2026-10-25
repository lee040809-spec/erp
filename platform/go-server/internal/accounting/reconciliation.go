package accounting

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"machining-erp/internal/productionflow"
	"sort"
	"time"
)

type Statement struct {
	ID             int64      `json:"id"`
	CorpID         int64      `json:"-"`
	CustomerID     int64      `json:"customer_id"`
	CustomerName   string     `json:"customer_name"`
	StatementNo    string     `json:"statement_no"`
	BusinessDate   string     `json:"business_date"`
	Amount         string     `json:"amount"`
	Status         string     `json:"status"`
	Notes          string     `json:"notes"`
	RequestID      string     `json:"-"`
	RequestHash    string     `json:"-"`
	CreatedBy      int64      `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
	ReversedBy     *int64     `json:"reversed_by"`
	ReversedAt     *time.Time `json:"reversed_at"`
	ReversalReason *string    `json:"reversal_reason"`
}

func (Statement) TableName() string { return "delivery_reconciliations" }

type DeliveryLine struct {
	ID               int64  `json:"id"`
	OrderID          int64  `json:"order_id"`
	OrderItemID      int64  `json:"order_item_id"`
	CustomerID       int64  `json:"customer_id"`
	CustomerName     string `json:"customer_name"`
	OrderNumber      string `json:"order_number"`
	PartName         string `json:"part_name"`
	PartNumber       string `json:"part_number"`
	BusinessDate     string `json:"business_date"`
	Quantity         int    `json:"quantity"`
	UnitPrice        string `json:"unit_price"`
	Amount           string `json:"amount"`
	PriceReady       int    `json:"price_ready"`
	ReconciliationID *int64 `json:"reconciliation_id"`
}

const DeliverySelect = `SELECT r.id,r.order_id,r.order_item_id,o.customer_id,o.customer_name,o.order_number,i.part_name,COALESCE(i.part_number,'') AS part_number,
 DATE_FORMAT(r.business_date,'%Y-%m-%d') AS business_date,r.quantity,CAST(i.unit_price AS CHAR) AS unit_price,CAST(ROUND(r.quantity*i.unit_price,2) AS CHAR) AS amount,
 CASE WHEN i.unit_price>0 AND i.currency='CNY' THEN 1 ELSE 0 END AS price_ready,l.reconciliation_id
 FROM order_business_records r JOIN orders o ON o.corp_id=r.corp_id AND o.id=r.order_id
 JOIN order_items i ON i.corp_id=r.corp_id AND i.id=r.order_item_id
 LEFT JOIN delivery_reconciliation_lines l ON l.corp_id=r.corp_id AND l.active_delivery_id=r.id`

type ReconcileInput struct {
	CustomerID  int64   `json:"customer_id"`
	Date        string  `json:"business_date"`
	DeliveryIDs []int64 `json:"delivery_ids"`
	Notes       string  `json:"notes"`
	RequestID   string  `json:"request_id"`
}

func Reconcile(db *gorm.DB, corp, actor int64, in ReconcileInput) (Statement, error) {
	var out Statement
	if in.CustomerID <= 0 || len(in.DeliveryIDs) == 0 || len(in.DeliveryIDs) > 200 || !productionflow.RequestID(in.RequestID) || !productionflow.Text(in.Notes, 500) {
		return out, errors.New("请选择同一客户的1～200次送货记录，备注最多500字")
	}
	if _, e := productionflow.Date(in.Date, true); e != nil {
		return out, e
	}
	sort.Slice(in.DeliveryIDs, func(i, j int) bool { return in.DeliveryIDs[i] < in.DeliveryIDs[j] })
	for i, id := range in.DeliveryIDs {
		if id <= 0 || i > 0 && id == in.DeliveryIDs[i-1] {
			return out, errors.New("送货记录编号无效或重复")
		}
	}
	raw, _ := json.Marshal(in)
	hash := fmt.Sprintf("%x", sha256.Sum256(raw))
	err := db.Transaction(func(tx *gorm.DB) error {
		e := tx.Where("corp_id=? AND request_id=?", corp, in.RequestID).Take(&out).Error
		if e == nil {
			if out.RequestHash != hash {
				return errors.New("请求编号已使用，请勿更改内容后重复提交")
			}
			return nil
		}
		if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		// Serialize against delivery corrections/reversals before reading the snapshots.
		var itemIDs []int64
		if e = tx.Table("order_business_records").Where("corp_id=? AND id IN ?", corp, in.DeliveryIDs).Distinct("order_item_id").Order("order_item_id").Pluck("order_item_id", &itemIDs).Error; e != nil {
			return e
		}
		for _, id := range itemIDs {
			if _, e = productionflow.LockItem(tx, corp, id); e != nil {
				return e
			}
		}
		var lines []DeliveryLine
		if e = tx.Raw(DeliverySelect+" WHERE r.corp_id=? AND r.id IN ? AND r.kind='delivery' AND r.status='confirmed' ORDER BY r.id FOR UPDATE", corp, in.DeliveryIDs).Scan(&lines).Error; e != nil {
			return e
		}
		if len(lines) != len(in.DeliveryIDs) {
			return errors.New("部分送货记录已变更或不存在，请刷新")
		}
		for _, l := range lines {
			if l.CustomerID != in.CustomerID || l.PriceReady != 1 || l.ReconciliationID != nil || l.Quantity <= 0 {
				return errors.New("只能核对同一客户、已补价且尚未对账的有效送货记录")
			}
		}
		out = Statement{CorpID: corp, CustomerID: in.CustomerID, CustomerName: lines[0].CustomerName, StatementNo: "DZ-" + in.RequestID, BusinessDate: in.Date, Amount: "0", Status: "confirmed", Notes: in.Notes, RequestID: in.RequestID, RequestHash: hash, CreatedBy: actor, CreatedAt: time.Now()}
		if e = tx.Create(&out).Error; e != nil {
			return e
		}
		for _, l := range lines {
			e = tx.Exec(`INSERT INTO delivery_reconciliation_lines(corp_id,reconciliation_id,delivery_record_id,active_delivery_id,order_id,order_item_id,order_number,part_name,part_number,business_date,quantity,unit_price,amount) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,ROUND(?*CAST(? AS DECIMAL(15,4)),2))`, corp, out.ID, l.ID, l.ID, l.OrderID, l.OrderItemID, l.OrderNumber, l.PartName, l.PartNumber, l.BusinessDate, l.Quantity, l.UnitPrice, l.Quantity, l.UnitPrice).Error
			if e != nil {
				return e
			}
		}
		if e = tx.Exec("UPDATE delivery_reconciliations SET amount=(SELECT SUM(amount) FROM delivery_reconciliation_lines WHERE corp_id=? AND reconciliation_id=?) WHERE corp_id=? AND id=?", corp, out.ID, corp, out.ID).Error; e != nil {
			return e
		}
		return tx.Where("corp_id=? AND id=?", corp, out.ID).Take(&out).Error
	})
	return out, err
}
func RequireUnreconciled(tx *gorm.DB, corp, delivery int64) error {
	var n int64
	if e := tx.Table("delivery_reconciliation_lines").Where("corp_id=? AND active_delivery_id=?", corp, delivery).Count(&n).Error; e != nil {
		return e
	}
	if n > 0 {
		return errors.New("本次送货已对账，请先撤销对应对账单，再修改或撤销交货记录")
	}
	return nil
}
func ReverseStatement(db *gorm.DB, corp, actor, id int64, reason string) error {
	if id <= 0 || reason == "" || !productionflow.Text(reason, 500) {
		return errors.New("撤销原因必填，最多500字")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var row Statement
		if e := tx.Raw("SELECT * FROM delivery_reconciliations WHERE corp_id=? AND id=? FOR UPDATE", corp, id).Scan(&row).Error; e != nil {
			return e
		}
		if row.ID == 0 {
			return errors.New("对账单不存在")
		}
		if row.Status == "reversed" {
			return nil
		}
		if e := tx.Exec("UPDATE delivery_reconciliation_lines SET active_delivery_id=NULL WHERE corp_id=? AND reconciliation_id=?", corp, id).Error; e != nil {
			return e
		}
		return tx.Exec("UPDATE delivery_reconciliations SET status='reversed',reversed_by=?,reversed_at=NOW(3),reversal_reason=? WHERE corp_id=? AND id=?", actor, reason, corp, id).Error
	})
}
