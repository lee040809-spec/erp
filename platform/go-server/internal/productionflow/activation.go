package productionflow

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"machining-erp/internal/models"
	"time"
)

// Activate preserves legacy timestamps. Unknown historical quantities must be entered by a person.
func Activate(tx *gorm.DB, item models.OrderItem, baseline []models.ProcessBaseline, actor int64) error {
	on, err := Enabled(tx, item.CorpID, item.ID)
	if err != nil {
		return err
	}
	if on {
		if len(baseline) > 0 {
			return errors.New("数量跟踪已启用，不能覆盖历史基线")
		}
		return nil
	}
	ps, err := route(tx, item.CorpID, item.ID)
	if err != nil {
		return err
	}
	if len(ps) == 0 {
		return errors.New("请先设置至少一道生产工序")
	}
	var confirmed int
	if err = tx.Table("production_completions").Where("corp_id=? AND order_item_id=? AND status='confirmed'", item.CorpID, item.ID).Select("CAST(COALESCE(SUM(qualified_quantity),0) AS SIGNED)").Scan(&confirmed).Error; err != nil {
		return err
	}
	historical := confirmed > 0 || item.ScrapQuantity > 0 || item.DeliveredQty != nil && *item.DeliveredQty > 0
	for _, p := range ps {
		historical = historical || p.Status != "pending" || p.ActualStartAt != nil || p.ActualEndAt != nil
	}
	if historical && len(baseline) != len(ps) {
		return errors.New("原零件已有生产记录，请逐工序核实已开工、已完工和报废数量后启用；不会按状态猜测数量")
	}
	inputs := map[int64]models.ProcessBaseline{}
	for _, v := range baseline {
		if _, ok := inputs[v.ProcessID]; ok {
			return errors.New("历史工序重复")
		}
		inputs[v.ProcessID] = v
	}
	if len(baseline) > 0 && len(baseline) != len(ps) {
		return errors.New("历史数量须覆盖全部生产工序")
	}
	capacity, allScrap := item.ProductionQuantity, 0
	for _, p := range ps {
		v := inputs[p.ID]
		if err = GuardReservedCapacity(tx, item, p.ID, v.Started); err != nil {
			return err
		}
		if len(baseline) > 0 && v.ProcessID != p.ID {
			return errors.New("历史工序不属于当前零件")
		}
		if v.Started < 0 || v.Completed < 0 || v.Scrap < 0 || v.Started > capacity || v.Completed > v.Started || v.Scrap > v.Started-v.Completed {
			return fmt.Errorf("%s历史数量不守恒：后道开工不能超过前道完工，完工加报废不能超过开工", p.Name)
		}
		capacity = v.Completed
		allScrap += v.Scrap
	}
	if allScrap != item.ScrapQuantity || capacity < confirmed {
		return errors.New("工序报废合计须等于原累计报废，末道完工不得少于已确认合格数量")
	}
	if item.DeliveredQty != nil && capacity < *item.DeliveredQty {
		return errors.New("末道历史完工数量不得少于累计已交货数量")
	}
	now := time.Now()
	for _, p := range ps {
		v := inputs[p.ID]
		t := models.ProcessQuantity{CorpID: item.CorpID, OrderItemID: item.ID, OrderProcessID: p.ID, StartedQuantity: v.Started, CompletedQuantity: v.Completed, ScrapQuantity: v.Scrap, ActivatedAt: now}
		if err = tx.Create(&t).Error; err != nil {
			return err
		}
		for kind, qty := range map[string]int{"start": v.Started, "complete": v.Completed, "scrap": v.Scrap} {
			if qty == 0 {
				continue
			}
			e := models.ProcessQuantityEvent{CorpID: item.CorpID, OrderItemID: item.ID, OrderProcessID: p.ID, Kind: kind, Quantity: qty, SourceType: "baseline", Notes: "启用数量跟踪时由用户核实的历史基线；实际日期未知，不新增产值或报废", RequestID: fmt.Sprintf("BASE-%d-%s", p.ID, kind), CreatedBy: actor, CreatedAt: now}
			if err = tx.Create(&e).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
