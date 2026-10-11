package productionflow

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"machining-erp/internal/models"
	"strings"
	"time"
	"unicode/utf8"
)

func guardRouteChange(tx *gorm.DB, corp, item int64, incoming []models.OrderProcess, allowArchive bool) error {
	var old []models.OrderProcess
	if err := tx.Where("corp_id=? AND order_item_id=?", corp, item).Order("sort_order,id").Find(&old).Error; err != nil {
		return err
	}
	oldByID := map[int64]models.OrderProcess{}
	for _, p := range old {
		oldByID[p.ID] = p
	}
	next := map[int64]models.OrderProcess{}
	positions := map[int64]int{}
	for i, p := range incoming {
		if strings.TrimSpace(p.Name) == "" || utf8.RuneCountInString(p.Name) > 100 {
			return errors.New("工序名称须填写，最多100个字")
		}
		if p.ID == 0 {
			if p.Status != "pending" {
				return errors.New("新增工序应为待加工；实际进度请通过报工记录")
			}
			continue
		}
		if _, ok := oldByID[p.ID]; !ok {
			return errors.New("工序不属于当前零件")
		}
		if _, ok := next[p.ID]; ok {
			return errors.New("同一工序不能重复提交")
		}
		next[p.ID] = p
		positions[p.ID] = i
	}
	var totals []models.ProcessQuantity
	if err := tx.Where("corp_id=? AND order_item_id=?", corp, item).Find(&totals).Error; err != nil {
		return err
	}
	tracked := len(totals) > 0
	quantities := map[int64]models.ProcessQuantity{}
	for _, q := range totals {
		quantities[q.OrderProcessID] = q
	}
	var eventIDs []int64
	if err := tx.Model(&models.ProcessQuantityEvent{}).Where("corp_id=? AND order_item_id=?", corp, item).Distinct("order_process_id").Pluck("order_process_id", &eventIDs).Error; err != nil {
		return err
	}
	history := map[int64]bool{}
	for _, id := range eventIDs {
		history[id] = true
	}
	var lines []models.OutsourcingOrderLine
	if err := tx.Where("corp_id=? AND order_item_id=?", corp, item).Find(&lines).Error; err != nil {
		return err
	}
	linked := map[int64]bool{}
	for _, line := range lines {
		linked[line.OrderProcessID] = true
	}
	var dispatches []models.OutsourcingDispatch
	if err := tx.Where("corp_id=? AND order_item_id=?", corp, item).Find(&dispatches).Error; err != nil {
		return err
	}
	var dispatchIDs []int64
	for _, d := range dispatches {
		linked[d.OrderProcessID] = true
		dispatchIDs = append(dispatchIDs, d.ID)
	}
	if len(dispatchIDs) > 0 {
		var covered []models.OutsourcingDispatchProcess
		if err := tx.Where("corp_id=? AND dispatch_id IN ?", corp, dispatchIDs).Find(&covered).Error; err != nil {
			return err
		}
		for _, p := range covered {
			linked[p.OrderProcessID] = true
		}
	}
	// Completed external transactions retain historical references. Only outstanding supply blocks removal.
	busyExternal := map[int64]bool{}
	var reserved []int64
	if err := tx.Table("outsourcing_order_lines l").Joins("JOIN outsourcing_orders o ON o.corp_id=l.corp_id AND o.id=l.outsourcing_order_id").Where("l.corp_id=? AND l.order_item_id=? AND o.status NOT IN ('cancelled','deleted') AND l.quantity>l.started_quantity", corp, item).Pluck("l.order_process_id", &reserved).Error; err != nil {
		return err
	}
	for _, id := range reserved {
		busyExternal[id] = true
	}
	for _, d := range dispatches {
		if d.OpenQuantity+d.ReworkWaitingQuantity+d.ReworkOutQuantity > 0 {
			busyExternal[d.OrderProcessID] = true
			var ids []int64
			if err := tx.Model(&models.OutsourcingDispatchProcess{}).Where("corp_id=? AND dispatch_id=?", corp, d.ID).Pluck("order_process_id", &ids).Error; err != nil {
				return err
			}
			for _, id := range ids {
				busyExternal[id] = true
			}
		}
	}
	for _, p := range old {
		n, exists := next[p.ID]
		q := quantities[p.ID]
		used := history[p.ID] || q.StartedQuantity > 0 || q.CompletedQuantity > 0 || q.ScrapQuantity > 0 || p.ActualStartAt != nil || p.ActualEndAt != nil || p.Status != "pending"
		if !exists {
			if allowArchive {
				if busyExternal[p.ID] {
					return fmt.Errorf("%s仍有未发出的外协数量、待回厂或返工，请先完成或取消对应外协业务", p.Name)
				}
				continue
			}
			if linked[p.ID] {
				return fmt.Errorf("%s已有外协关联，不能删除；请先处理对应外协订单或发出记录", p.Name)
			}
			// Actual time protection is enforced by the repository and retains its typed error.
			if used && p.ActualStartAt == nil && p.ActualEndAt == nil {
				return fmt.Errorf("%s已有生产数量或进度记录，不能删除；可修改名称、计划或新增后续工序", p.Name)
			}
			if history[p.ID] || q.StartedQuantity > 0 || q.CompletedQuantity > 0 || q.ScrapQuantity > 0 {
				return fmt.Errorf("%s已有报工台账，不能删除", p.Name)
			}
			continue
		}
		if (p.Name == "送货") != (n.Name == "送货") {
			return errors.New("送货节点不能改为生产工序，生产工序也不能改为送货节点")
		}
		if tracked && n.Status != p.Status {
			return fmt.Errorf("%s状态由报工数量计算，请在分批生产中记录；工艺修改不应改变实际进度", p.Name)
		}
		if n.IsOutsourced != p.IsOutsourced && allowArchive && (q.StartedQuantity > q.CompletedQuantity+q.ScrapQuantity || !tracked && p.Status == models.ProcessStatusProcessing) {
			return fmt.Errorf("%s仍有在制件，不能切换加工方式；可先完成原报工，或移除后新增替代工序", p.Name)
		}
		if n.IsOutsourced != p.IsOutsourced && (busyExternal[p.ID] || !allowArchive && (used || linked[p.ID])) {
			return fmt.Errorf("%s已有生产或外协记录，不能切换内制/外协；可新增工序安排后续加工", p.Name)
		}
	}

	// Posted qualified output must keep its original terminal operation.
	var confirmed int64
	if err := tx.Table("production_completions").Where("corp_id=? AND order_item_id=? AND status='confirmed'", corp, item).Count(&confirmed).Error; err != nil {
		return err
	}
	if confirmed > 0 {
		var previousLast, nextLast int64
		for _, p := range old {
			if p.Name != "送货" {
				previousLast = p.ID
			}
		}
		for _, p := range incoming {
			if p.Name != "送货" {
				nextLast = p.ID
			}
		}
		if previousLast != nextLast && (!allowArchive || next[previousLast].ID != 0) {
			return errors.New("已有合格完工确认，须保留原末道工序；可以改名、调整计划及未开工步骤，新增末道需先处理原完工记录")
		}
	}
	// A full-route purchase is one physical shipment; its covered sequence must remain intact.
	var full []models.OutsourcingOrder
	if err := tx.Where("corp_id=? AND order_item_id=? AND scope='full' AND status NOT IN ('cancelled','deleted')", corp, item).Find(&full).Error; err != nil {
		return err
	}
	for _, order := range full {
		covered := []models.OutsourcingOrderLine{}
		if err := tx.Where("corp_id=? AND outsourcing_order_id=?", corp, order.ID).Order("sequence,id").Find(&covered).Error; err != nil {
			return err
		}
		busy := false
		for _, line := range covered {
			if busyExternal[line.OrderProcessID] {
				busy = true
			}
		}
		if allowArchive && !busy {
			continue
		}
		for i, line := range covered {
			pos, ok := positions[line.OrderProcessID]
			if !ok || i > 0 && pos != positions[covered[i-1].OrderProcessID]+1 {
				return errors.New("整套外协覆盖的工序须保持连续和原有顺序；可调整名称、日期，范围变更请先处理整套外协单")
			}
		}
	}
	return nil
}

// Initialize only newly added steps. Existing IDs, quantities, events and activation date survive.
func ReconcileRoute(tx *gorm.DB, corp, itemID int64) error {
	var item models.OrderItem
	if err := tx.Where("corp_id=? AND id=?", corp, itemID).Take(&item).Error; err != nil {
		return err
	}
	if item.ProductionQuantity == 0 {
		item.ProductionQuantity = item.Quantity
	}
	ps, err := route(tx, corp, itemID)
	if err != nil {
		return err
	}
	if len(ps) == 0 {
		return errors.New("已启用数量跟踪的零件至少保留一道生产工序")
	}
	var totals []models.ProcessQuantity
	if err = tx.Where("corp_id=? AND order_item_id=?", corp, itemID).Find(&totals).Error; err != nil {
		return err
	}
	byID := map[int64]models.ProcessQuantity{}
	activated := time.Now()
	for _, q := range totals {
		byID[q.OrderProcessID] = q
		if q.ActivatedAt.Before(activated) {
			activated = q.ActivatedAt
		}
	}
	retained := map[int64]bool{}
	for _, p := range ps {
		retained[p.ID] = true
		if _, ok := byID[p.ID]; !ok {
			q := models.ProcessQuantity{CorpID: corp, OrderItemID: itemID, OrderProcessID: p.ID, ActivatedAt: activated}
			if err = tx.Create(&q).Error; err != nil {
				return err
			}
			byID[p.ID] = q
		}
	}
	for _, q := range totals {
		if !retained[q.OrderProcessID] {
			var archived models.OrderProcess
			if err = tx.Unscoped().Where("id=? AND corp_id=? AND order_item_id=? AND deleted_at IS NOT NULL", q.OrderProcessID, corp, itemID).Take(&archived).Error; err == nil {
				continue
			} else if err != gorm.ErrRecordNotFound {
				return err
			}
			if q.StartedQuantity != 0 || q.CompletedQuantity != 0 || q.ScrapQuantity != 0 {
				return errors.New("不能删除有数量的工序")
			}
			if err = tx.Where("id=? AND corp_id=?", q.ID, corp).Delete(&models.ProcessQuantity{}).Error; err != nil {
				return err
			}
		}
	}
	if err = validateChronology(tx, item, ps); err != nil {
		return fmt.Errorf("工艺调整与原报工不一致：%w；请保留已开工工序的前道供给关系", err)
	}
	var confirmed int
	if err = tx.Table("production_completions").Where("corp_id=? AND order_item_id=? AND status='confirmed'", corp, itemID).Select("CAST(COALESCE(SUM(qualified_quantity),0) AS SIGNED)").Scan(&confirmed).Error; err != nil {
		return err
	}
	losses, e := archivedLosses(tx, corp, itemID)
	if e != nil {
		return e
	}
	last := byID[ps[len(ps)-1].ID].CompletedQuantity - terminalLoss(losses)
	if last < confirmed || item.DeliveredQty != nil && last < *item.DeliveredQty {
		return errors.New("调整后的末道完工数量不能少于已确认合格或已交货数量")
	}
	return nil
}
