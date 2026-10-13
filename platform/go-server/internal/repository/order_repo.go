package repository

import (
	"errors"
	"machining-erp/internal/drawings"
	"machining-erp/internal/productionflow"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"machining-erp/internal/models"
	"machining-erp/pkg/utils"
)

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(order *models.Order, items []models.OrderItem, actors ...int64) (int64, error) {
	var actor int64
	if len(actors) > 0 {
		actor = actors[0]
	}
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// 计算每个订单项的状态
		for i := range items {
			items[i].ScrapQuantity = 0
			items[i].DeliveredQty = nil
			items[i].CompletionDate = nil
			if items[i].ID == 0 {
				if items[i].ProductionQuantity == 0 {
					items[i].ProductionQuantity = items[i].Quantity
				}
				if err := validateProductionQuantities(items[i].Quantity, items[i].ProductionQuantity, items[i].ScrapQuantity, 0, 0); err != nil {
					return err
				}
			}
			if err := protectDeliveryProcesses(tx, &items[i]); err != nil {
				return err
			}
			items[i].Status = models.OrderStatus(utils.CalculateOrderItemStatus(items[i].Processes))
			if items[i].Status == models.OrderStatusCompleted {
				items[i].Status = models.OrderStatusProcessing
			}
		}

		// 插入订单
		if err := tx.Create(order).Error; err != nil {
			return err
		}

		// 插入订单项
		for i := range items {
			items[i].OrderID = order.ID
			items[i].CorpID = order.CorpID
			// 先保存工序，然后清空避免 GORM 自动插入关联
			processes := items[i].Processes
			items[i].Processes = nil
			if err := tx.Omit("Processes").Create(&items[i]).Error; err != nil {
				return err
			}

			// 插入工序
			for j := range processes {
				if err := models.NormalizeCuttingSpec(&processes[j]); err != nil {
					return err
				}
				processes[j].OrderItemID = items[i].ID
				processes[j].CorpID = order.CorpID
				processes[j].SortOrder = j
				if err := tx.Create(&processes[j]).Error; err != nil {
					return err
				}
			}
			items[i].Processes = processes
			if e := drawings.Bind(tx, items[i], items[i].DrawingIDs, actor, "新建订单关联图纸"); e != nil {
				return e
			}
			if len(items[i].DeliveryPlan) > 0 {
				if e := productionflow.ConfigureBatches(tx, items[i], items[i].DeliveryPlan, nil, actor); e != nil {
					return e
				}
			}
		}

		if err := recalculateOrderAmounts(tx, order); err != nil {
			return err
		}
		// 计算订单状态
		var statuses []string
		for _, item := range items {
			statuses = append(statuses, string(item.Status))
		}
		order.Status = models.OrderStatus(utils.CalculateStatus(statuses))
		return tx.Model(order).Omit("Items").Updates(map[string]interface{}{"status": order.Status, "total_amount": order.TotalAmount}).Error
	})

	if err != nil {
		return 0, err
	}
	return int64(order.ID), nil
}

// handleProcesses 处理已存在订单项的工序更新逻辑
func handleProcesses(tx *gorm.DB, itemID, corpID int64, processes []models.OrderProcess, actor int64, planProvided map[int64]bool, changes ...RouteChangeOptions) error {
	var change RouteChangeOptions
	if len(changes) > 0 {
		change = changes[0]
	}
	tracked, err := productionflow.Enabled(tx, corpID, itemID)
	if err != nil {
		return err
	}
	// 获取旧的工序ID
	var oldProcessIDs []int64
	if err := tx.Model(&models.OrderProcess{}).Where("order_item_id = ? AND corp_id=?", itemID, corpID).Order("sort_order,id").Pluck("id", &oldProcessIDs).Error; err != nil {
		return err
	}
	oldSet := map[int64]bool{}
	for _, id := range oldProcessIDs {
		oldSet[id] = true
	}

	// 收集前端传来的工序ID
	newProcessIDs := make(map[int64]bool)
	newProcessNames := make(map[int64]string)
	for _, p := range processes {
		if p.ID != 0 {
			if !oldSet[p.ID] {
				return ErrOrderScope
			}
			newProcessIDs[p.ID] = true
			newProcessNames[p.ID] = p.Name
		}
	}

	// Validate deletions before route changes, retaining the original actual-time error.
	for _, oldPID := range oldProcessIDs {
		if !newProcessIDs[oldPID] {
			if err := protectTimedProcess(tx, corpID, oldPID); err != nil && (!change.AllowProcessedChanges || !errors.Is(err, ErrTimedProcess)) {
				return err
			}
		}
	}
	if err := productionflow.GuardRoute(tx, corpID, itemID, processes, change.AllowProcessedChanges); err != nil {
		return err
	}
	// 删除不再存在的工序
	for _, oldPID := range oldProcessIDs {
		if !newProcessIDs[oldPID] {
			var previous models.OrderProcess
			if err := tx.Where("id=? AND corp_id=? AND order_item_id=?", oldPID, corpID, itemID).Take(&previous).Error; err != nil {
				return err
			}
			after := previous
			for i, id := range oldProcessIDs {
				if id != oldPID {
					continue
				}
				for _, nextID := range oldProcessIDs[i+1:] {
					if newProcessIDs[nextID] && newProcessNames[nextID] != "送货" {
						anchor := nextID
						after.RetiredBeforeID = &anchor
						break
					}
				}
				break
			}
			after.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
			reason := change.Reason
			if reason == "" {
				reason = "订单编辑移除工序；保留原有工序历史"
			}
			if err := RecordProcessTimeEvent(tx, previous, after, actor, "route", reason); err != nil {
				return err
			}
			if err := tx.Model(&previous).Update("deleted_at", after.DeletedAt.Time).Error; err != nil {
				return err
			}
		}
	}

	// 更新或插入工序
	for j := range processes {
		if err := models.NormalizeCuttingSpec(&processes[j]); err != nil {
			return err
		}
		processes[j].OrderItemID = itemID
		processes[j].CorpID = corpID
		processes[j].SortOrder = j

		if processes[j].ID != 0 {
			var previous models.OrderProcess
			if err := tx.Where("id=? AND corp_id=? AND order_item_id=?", processes[j].ID, corpID, itemID).Take(&previous).Error; err != nil {
				return err
			}
			if processes[j].CuttingSpec == nil {
				processes[j].CuttingSpec = previous.CuttingSpec
			}
			timeFields, err := ApplyProcessTimeTransition(tx, previous, processes[j].Status, actor)
			if err != nil {
				return err
			}
			fields := map[string]interface{}{"name": processes[j].Name, "is_outsourced": processes[j].IsOutsourced, "outsourcing_fee": processes[j].OutsourcingFee, "status": processes[j].Status, "sort_order": j}
			if processes[j].CuttingSpec != nil {
				fields["cutting_spec"] = *processes[j].CuttingSpec
			}
			for key, value := range timeFields {
				fields[key] = value
			}
			if planProvided[processes[j].ID] {
				beforePlan := previous
				beforePlan.Status = processes[j].Status
				if v, ok := timeFields["actual_start_at"].(time.Time); ok {
					beforePlan.ActualStartAt = &v
				}
				if v, ok := timeFields["actual_end_at"].(time.Time); ok {
					beforePlan.ActualEndAt = &v
				}
				afterPlan := beforePlan
				afterPlan.PlannedStartAt, afterPlan.PlannedEndAt = processes[j].PlannedStartAt, processes[j].PlannedEndAt
				if err := utils.ValidateProcessPlan(afterPlan.PlannedStartAt, afterPlan.PlannedEndAt); err != nil {
					return err
				}
				fields["planned_start_at"], fields["planned_end_at"] = afterPlan.PlannedStartAt, afterPlan.PlannedEndAt
				if err := RecordProcessTimeEvent(tx, beforePlan, afterPlan, actor, "order", "订单编辑调整工序计划日期"); err != nil {
					return err
				}
			}
			if !models.SameCuttingSpec(previous.CuttingSpec, processes[j].CuttingSpec) || previous.Name != processes[j].Name || previous.SortOrder != j || previous.IsOutsourced != processes[j].IsOutsourced {
				after := previous
				after.CuttingSpec = processes[j].CuttingSpec
				after.Name, after.SortOrder, after.IsOutsourced = processes[j].Name, j, processes[j].IsOutsourced
				if err := RecordProcessTimeEvent(tx, previous, after, actor, "route", routeChangeReason(change)); err != nil {
					return err
				}
			}
			// 有ID，执行更新
			if err := tx.Model(&models.OrderProcess{}).Where("id = ? AND order_item_id = ? AND corp_id=?", processes[j].ID, itemID, corpID).Updates(fields).Error; err != nil {
				return err
			}
		} else {
			// 没有ID，插入新工序
			if err := tx.Create(&processes[j]).Error; err != nil {
				return err
			}
		}
	}

	if tracked {
		return productionflow.ReconcileRoute(tx, corpID, itemID)
	}
	return nil
}

type OrderUpdateOptions struct {
	ActorID              int64
	TaxProvided          map[int64]bool
	UnitPriceProvided    map[int64]bool
	ProtectAmounts       bool
	ProcessPlansProvided map[int64]bool
}

func (r *OrderRepository) Update(order *models.Order, items []models.OrderItem, options ...OrderUpdateOptions) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var existing models.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", order.ID, order.CorpID).Take(&existing).Error; err != nil {
			return err
		}
		var oldItems []models.OrderItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_id=? AND corp_id=?", order.ID, order.CorpID).Order("id").Find(&oldItems).Error; err != nil {
			return err
		}
		oldByID := map[int64]models.OrderItem{}
		for _, it := range oldItems {
			oldByID[it.ID] = it
		}
		// 更新订单基本信息
		if err := tx.Model(order).Updates(map[string]interface{}{
			"customer_id":         order.CustomerID,
			"customer_name":       order.CustomerName,
			"customer_short_name": order.CustomerShortName,
			"order_number":        order.OrderNumber,
			"order_name":          order.OrderName,
			"contact_name":        order.ContactName,
			"priority":            order.Priority,
			"start_date":          order.StartDate,
			"due_date":            order.DueDate,
			"notes":               order.Notes,
		}).Error; err != nil {
			return err
		}

		if items != nil {
			// 获取旧的订单项ID
			var oldItemIDs []int64
			for _, it := range oldItems {
				oldItemIDs = append(oldItemIDs, it.ID)
			}

			// 收集前端传来的订单项ID
			newItemIDs := make(map[int64]bool)
			for _, item := range items {
				if item.ID != 0 {
					if _, ok := oldByID[item.ID]; !ok {
						return ErrOrderScope
					}
					newItemIDs[item.ID] = true
				}
			}

			// 删除不再存在的订单项和其工序
			for _, oldID := range oldItemIDs {
				if !newItemIDs[oldID] {
					if err := protectPostedItem(tx, order.CorpID, oldID); err != nil {
						return err
					}
					if err := tx.Exec("DELETE FROM production_plans WHERE corp_id=? AND order_item_id=?", order.CorpID, oldID).Error; err != nil {
						return err
					}
					if err := tx.Delete(&models.OrderProcess{}, "order_item_id = ? AND corp_id=?", oldID, order.CorpID).Error; err != nil {
						return err
					}
					if err := tx.Where("corp_id=?", order.CorpID).Delete(&models.OrderItem{}, oldID).Error; err != nil {
						return err
					}
				}
			}

			// 更新或插入订单项和工序
			for i := range items {
				items[i].OrderID = order.ID
				items[i].CorpID = order.CorpID
				if items[i].ID == 0 {
					items[i].ScrapQuantity = 0
					items[i].DeliveredQty = nil
					items[i].CompletionDate = nil
					if items[i].ProductionQuantity == 0 {
						items[i].ProductionQuantity = items[i].Quantity
					}
					if err := validateProductionQuantities(items[i].Quantity, items[i].ProductionQuantity, items[i].ScrapQuantity, 0, 0); err != nil {
						return err
					}
				}
				if err := protectDeliveryProcesses(tx, &items[i]); err != nil {
					return err
				}
				items[i].Status = models.OrderStatus(utils.CalculateOrderItemStatus(items[i].Processes))
				if items[i].Status == models.OrderStatusCompleted {
					items[i].Status = models.OrderStatusProcessing
				}

				if items[i].ID != 0 {
					old := oldByID[items[i].ID]
					items[i].ScrapQuantity = old.ScrapQuantity
					items[i].DeliveredQty = old.DeliveredQty
					items[i].CompletionDate = old.CompletionDate
					if old.DeliveredQty != nil && *old.DeliveredQty > items[i].Quantity {
						return ErrPostedQuantity
					}
					if items[i].ProductionQuantity == 0 {
						items[i].ProductionQuantity = old.ProductionQuantity
						if items[i].ProductionQuantity == 0 {
							items[i].ProductionQuantity = old.Quantity
						}
						if items[i].ProductionQuantity < items[i].Quantity {
							items[i].ProductionQuantity = items[i].Quantity
						}
					}
					if old.Status == models.OrderStatusDelivered {
						items[i].Status = old.Status
					}
					if len(options) > 0 && !options[0].TaxProvided[items[i].ID] {
						items[i].TaxInclusive = old.TaxInclusive
					}
					if items[i].SupplyMode == "" {
						items[i].SupplyMode = old.SupplyMode
					}
					if items[i].Currency == "" {
						items[i].Currency = old.Currency
					}
					var posted struct {
						Count    int64
						Quantity float64
						Billable float64
					}
					if err := tx.Raw("SELECT COUNT(*) AS count,COALESCE(SUM(CASE WHEN status='confirmed' THEN qualified_quantity ELSE 0 END),0) AS quantity,COALESCE(SUM(CASE WHEN status='confirmed' THEN billable_quantity ELSE 0 END),0) AS billable FROM production_completions WHERE corp_id=? AND order_item_id=?", order.CorpID, items[i].ID).Scan(&posted).Error; err != nil {
						return err
					}
					if e := productionflow.GuardQuantities(tx, order.CorpID, items[i].ID, items[i].Quantity, items[i].ProductionQuantity); e != nil {
						return e
					}
					if e := productionflow.GuardDates(tx, items[i]); e != nil {
						return e
					}
					if err := validateProductionQuantities(items[i].Quantity, items[i].ProductionQuantity, items[i].ScrapQuantity, posted.Quantity, posted.Billable); err != nil {
						return err
					}
					if posted.Count == 0 && old.Status == models.OrderStatusCompleted {
						items[i].Status = old.Status
					}
					if posted.Count > 0 {
						if old.Status != models.OrderStatusDelivered {
							if posted.Quantity >= float64(items[i].Quantity) {
								items[i].Status = models.OrderStatusCompleted
							} else {
								items[i].Status = models.OrderStatusProcessing
							}
						}
						items[i].CompletionDate = old.CompletionDate
						if items[i].SupplyMode != old.SupplyMode || items[i].Currency != old.Currency || items[i].TaxInclusive != old.TaxInclusive {
							return ErrPostedBasis
						}
					}
					if len(options) > 0 && options[0].ProtectAmounts {
						items[i].UnitPrice = old.UnitPrice
						items[i].ToolCost, items[i].FixtureCost = nil, nil
						items[i].MaterialCost, items[i].OtherCost = nil, nil
						var oldProcesses []models.OrderProcess
						if err := tx.Where("order_item_id=? AND corp_id=?", old.ID, order.CorpID).Find(&oldProcesses).Error; err != nil {
							return err
						}
						fees := map[int64]float64{}
						for _, p := range oldProcesses {
							fees[p.ID] = p.OutsourcingFee
						}
						for j := range items[i].Processes {
							items[i].Processes[j].OutsourcingFee = fees[items[i].Processes[j].ID]
						}
					}
					// 更新已存在的订单项
					updateFields := map[string]interface{}{
						"part_name":           items[i].PartName,
						"part_number":         items[i].PartNumber,
						"quantity":            items[i].Quantity,
						"production_quantity": items[i].ProductionQuantity,
						"scrap_quantity":      items[i].ScrapQuantity,
						"status":              items[i].Status,
						"notes":               items[i].Notes,
						"start_date":          items[i].StartDate,
						"due_date":            items[i].DueDate,
						"delivered_quantity":  items[i].DeliveredQty,
						"item_notes":          items[i].ItemNotes,
						"completion_date":     items[i].CompletionDate,
						"supply_mode":         items[i].SupplyMode,
						"currency":            items[i].Currency,
						"tax_inclusive":       items[i].TaxInclusive,
					}
					// Lists intentionally omit drawing contents. Missing drawing data must never erase it.
					if items[i].DrawingData != nil {
						updateFields["drawing_data"] = items[i].DrawingData
					}
					// 只有当费用字段有值时才更新（避免清空原有值）
					if (len(options) == 0 && items[i].UnitPrice != 0) || (len(options) > 0 && options[0].UnitPriceProvided[items[i].ID]) {
						updateFields["unit_price"] = items[i].UnitPrice
					}

					if items[i].ToolCost != nil {
						updateFields["tool_cost"] = items[i].ToolCost
					}
					if items[i].FixtureCost != nil {
						updateFields["fixture_cost"] = items[i].FixtureCost
					}
					if items[i].MaterialCost != nil {
						updateFields["material_cost"] = items[i].MaterialCost
					}
					if items[i].OtherCost != nil {
						updateFields["other_cost"] = items[i].OtherCost
					}
					if err := tx.Model(&items[i]).Omit("Processes").Updates(updateFields).Error; err != nil {
						return err
					}

					// 处理已存在订单项的工序
					var actor int64
					var processPlans map[int64]bool
					if len(options) > 0 {
						actor = options[0].ActorID
						processPlans = options[0].ProcessPlansProvided
					}
					if err := handleProcesses(tx, items[i].ID, order.CorpID, items[i].Processes, actor, processPlans); err != nil {
						return err
					}
				} else {
					if len(options) > 0 && options[0].ProtectAmounts {
						items[i].UnitPrice = 0
						items[i].ToolCost, items[i].FixtureCost = nil, nil
						items[i].MaterialCost, items[i].OtherCost = nil, nil
						for j := range items[i].Processes {
							items[i].Processes[j].OutsourcingFee = 0
						}
					}
					items[i].DrawingChangeReason = "新增订单零件关联图纸"
					// 插入新的订单项：清空所有工序ID，确保全新插入
					for j := range items[i].Processes {
						items[i].Processes[j].ID = 0
						items[i].Processes[j].OrderItemID = 0
					}
					if err := tx.Omit("Processes").Create(&items[i]).Error; err != nil {
						return err
					}

					// 插入工序（此时items[i].ID已赋值）
					for j := range items[i].Processes {
						items[i].Processes[j].OrderItemID = items[i].ID
						items[i].Processes[j].CorpID = order.CorpID
						items[i].Processes[j].SortOrder = j
						if err := tx.Omit("id").Create(&items[i].Processes[j]).Error; err != nil {
							return err
						}
					}
					if len(items[i].DeliveryPlan) > 0 {
						var actor int64
						if len(options) > 0 {
							actor = options[0].ActorID
						}
						if e := productionflow.ConfigureBatches(tx, items[i], items[i].DeliveryPlan, nil, actor); e != nil {
							return e
						}
					}
				}
			}
		}

		if err := recalculateOrderAmounts(tx, order); err != nil {
			return err
		}
		var drawingActor int64
		if len(options) > 0 {
			drawingActor = options[0].ActorID
		}
		for _, item := range items {
			if e := drawings.Bind(tx, item, item.DrawingIDs, drawingActor, item.DrawingChangeReason); e != nil {
				return e
			}
		}
		// 重新计算订单状态
		var statuses []string
		if err := tx.Model(&models.OrderItem{}).Where("order_id = ? AND corp_id=?", order.ID, order.CorpID).Pluck("status", &statuses).Error; err != nil {
			return err
		}
		order.Status = models.OrderStatus(utils.CalculateStatus(statuses))
		return tx.Model(order).Omit("Items").Updates(map[string]interface{}{"status": order.Status, "total_amount": order.TotalAmount}).Error
	})
}

func recalculateOrderAmounts(tx *gorm.DB, order *models.Order) error {
	if err := tx.Exec("UPDATE order_items SET total_price=ROUND(quantity*unit_price,2) WHERE order_id=? AND corp_id=?", order.ID, order.CorpID).Error; err != nil {
		return err
	}
	return tx.Table("order_items").Where("order_id=? AND corp_id=?", order.ID, order.CorpID).Select("COALESCE(SUM(total_price),0)").Scan(&order.TotalAmount).Error
}

func (r *OrderRepository) UpdateStatus(id int64, status string) error {
	return r.db.Model(&models.Order{}).Where("id = ?", id).Update("status", status).Error
}

func (r *OrderRepository) Delete(id, corpID int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var order models.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", id, corpID).Take(&order).Error; err != nil {
			return err
		}
		var businessCount int64
		if err := tx.Table("order_business_records").Where("corp_id=? AND order_id=?", corpID, id).Count(&businessCount).Error; err != nil {
			return err
		}
		if businessCount > 0 {
			return ErrPostedOrder
		}
		var items []models.OrderItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_id=? AND corp_id=?", id, corpID).Order("id").Find(&items).Error; err != nil {
			return err
		}
		var costs int64
		if err := tx.Table("cost_entries").Where("corp_id=? AND order_id=?", corpID, id).Count(&costs).Error; err != nil {
			return err
		}
		if costs > 0 {
			return ErrPostedOrder
		}
		if err := tx.Table("production_completions").Where("corp_id=? AND order_id=?", corpID, id).Count(&costs).Error; err != nil {
			return err
		}
		if costs > 0 {
			return ErrPostedOrder
		}
		for _, it := range items {
			if err := protectPostedItem(tx, corpID, it.ID); err != nil {
				return err
			}
		}
		var itemIDs []int
		for _, it := range items {
			itemIDs = append(itemIDs, int(it.ID))
		}
		if len(itemIDs) > 0 {
			if err := tx.Exec("DELETE FROM production_plans WHERE corp_id=? AND order_item_id IN ?", corpID, itemIDs).Error; err != nil {
				return err
			}
			if err := tx.Delete(&models.OrderProcess{}, "order_item_id IN ? AND corp_id=?", itemIDs, corpID).Error; err != nil {
				return err
			}
		}
		if err := tx.Delete(&models.OrderItem{}, "order_id = ? AND corp_id=?", id, corpID).Error; err != nil {
			return err
		}
		return tx.Where("corp_id=?", corpID).Delete(&models.Order{}, id).Error
	})
}

// GetByID 根据订单ID获取完整订单信息（包含零件和工序）
func (r *OrderRepository) GetByID(orderID int64) (*models.Order, error) {
	var order models.Order
	if err := r.db.First(&order, orderID).Error; err != nil {
		return nil, err
	}

	// 查询零件信息
	var items []models.OrderItem
	if err := r.db.Where("order_id = ?", orderID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}

	// 收集零件ID
	itemIDs := make([]int64, len(items))
	for i, it := range items {
		itemIDs[i] = it.ID
	}

	// 查询工序信息
	var processes []models.OrderProcess
	if len(itemIDs) > 0 {
		if err := r.db.Where("order_item_id IN ?", itemIDs).Order("sort_order ASC").Find(&processes).Error; err != nil {
			return nil, err
		}
	}

	// 构建工序映射
	processMap := make(map[int64][]models.OrderProcess)
	for _, p := range processes {
		processMap[p.OrderItemID] = append(processMap[p.OrderItemID], p)
	}

	// 组装零件的工序
	for i := range items {
		items[i].Processes = processMap[items[i].ID]
	}

	order.Items = items
	return &order, nil
}

// DashboardItem 工作看板零件数据
type DashboardItem struct {
	OrderID           int64                 `json:"order_id"`
	OrderNumber       string                `json:"order_number"`
	CustomerShortName string                `json:"customer_short_name"`
	Priority          string                `json:"priority"`
	ItemID            int64                 `json:"item_id"`
	PartName          string                `json:"part_name"`
	PartNumber        string                `json:"part_number"`
	Quantity          int                   `json:"quantity"`
	Status            string                `json:"status"`
	StartDate         *time.Time            `json:"start_date"`
	DueDate           *time.Time            `json:"due_date"`
	DrawingData       string                `json:"drawing_data"`
	Processes         []models.OrderProcess `json:"processes"`
}

// DashboardResult 工作看板返回结果
type DashboardResult struct {
	Items    []DashboardItem `json:"items"`
	Orders   []OrderInfo     `json:"orders"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
}

// OrderInfo 订单简要信息
type OrderInfo struct {
	ID                int64  `json:"id"`
	OrderNumber       string `json:"order_number"`
	CustomerName      string `json:"customer_name"`
	CustomerShortName string `json:"customer_short_name"`
	Status            string `json:"status"`
	Priority          string `json:"priority"`
}

// DashboardStats 看板卡片统计数据
type DashboardStats struct {
	PendingCount    int64 `json:"pending_count"`
	ProcessingCount int64 `json:"processing_count"`
	CompletedCount  int64 `json:"completed_count"`
	OverdueCount    int64 `json:"overdue_count"`  // 逾期订单
	WarningCount    int64 `json:"warning_count"`  // 告警订单
	NearDueCount    int64 `json:"near_due_count"` // 临期订单
}

// GetDashboardStats 获取看板卡片统计数据
func (r *OrderRepository) GetDashboardStats(corpID int64) (*DashboardStats, error) {
	stats := &DashboardStats{}
	today := time.Now().Format("2006-01-02")

	// 统计各状态的零件数量
	query := r.db.Table("order_items as oi").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Where("oi.status NOT IN ?", []string{"delivered", "completed"})
	query = query.Where("o.corp_id = ? AND oi.corp_id=?", corpID, corpID)
	rows, err := query.Select("oi.status, COUNT(*) as count").Group("oi.status").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		switch status {
		case "pending":
			stats.PendingCount = count
		case "processing":
			stats.ProcessingCount = count
		}
	}

	// 已完成的统计
	completedQuery := r.db.Table("order_items as oi").Joins("JOIN orders o ON o.id = oi.order_id").Where("oi.status = ?", "completed")
	completedQuery = completedQuery.Where("o.corp_id = ? AND oi.corp_id=?", corpID, corpID)
	completedQuery.Count(&stats.CompletedCount)

	// 逾期订单：零件交期 < 今天 且订单状态未完成
	overdueQuery := r.db.Table("order_items as oi").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Where("oi.due_date IS NOT NULL AND oi.due_date < ?", today).
		Where("o.status NOT IN ?", []string{"delivered", "completed"})
	overdueQuery = overdueQuery.Where("o.corp_id = ?", corpID)
	overdueQuery.Count(&stats.OverdueCount)

	// 告警订单：根据规则统计
	warningIDs, err := r.getOrderIDsByRuleType(int(corpID), "warning")
	if err != nil {
		return nil, err
	}
	stats.WarningCount = int64(len(warningIDs))

	// 临期订单：根据规则统计
	imminentIDs, err := r.getOrderIDsByRuleType(int(corpID), "imminent")
	if err != nil {
		return nil, err
	}
	stats.NearDueCount = int64(len(imminentIDs))

	return stats, nil
}

// GetDashboardItems 获取工作看板零件数据（带分页）
func (r *OrderRepository) GetDashboardItems(corpID int64, page, pageSize int) (*DashboardResult, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	// 1. 查询总数
	var total int64
	countQuery := r.db.Table("order_items as oi").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Where("oi.status NOT IN ?", []string{"delivered", "completed"})
	countQuery = countQuery.Where("o.corp_id = ? AND oi.corp_id=?", corpID, corpID)
	countQuery.Count(&total)

	// 2. 分页查询零件
	var results []struct {
		OrderID           int64
		OrderNumber       string
		CustomerShortName string
		Priority          string
		ItemID            int64
		PartName          string
		PartNumber        string
		Quantity          int
		Status            string
		StartDate         *time.Time
		DueDate           *time.Time
		DrawingData       *string
	}

	dataQuery := r.db.Table("order_items as oi").
		Select(`o.id as order_id, o.order_number,
			COALESCE(o.customer_short_name, '') as customer_short_name,
			o.priority, oi.id as item_id, oi.part_name,
			COALESCE(oi.part_number, '') as part_number,
			oi.quantity, oi.status,
			oi.start_date, oi.due_date,
			'' as drawing_data`).
		Joins("JOIN orders o ON o.id = oi.order_id").
		Where("oi.status NOT IN ?", []string{"delivered", "completed"})
	dataQuery = dataQuery.Where("o.corp_id = ? AND oi.corp_id=?", corpID, corpID)
	err := dataQuery.Order("oi.start_date ASC, oi.due_date ASC, oi.status ASC").
		Limit(pageSize).Offset(offset).
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return &DashboardResult{
			Items:    []DashboardItem{},
			Orders:   []OrderInfo{},
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	// 3. 获取订单ID列表，查询订单信息
	orderIDs := make([]int64, 0)
	orderIDSet := make(map[int64]bool)
	for _, r := range results {
		if !orderIDSet[r.OrderID] {
			orderIDs = append(orderIDs, r.OrderID)
			orderIDSet[r.OrderID] = true
		}
	}

	var orderInfos []OrderInfo
	r.db.Table("orders").
		Select("id, order_number, customer_name, customer_short_name, status, priority").
		Where("id IN ?", orderIDs).
		Where("corp_id=?", corpID).
		Order("start_date ASC").
		Find(&orderInfos)

	// 4. 批量加载工序
	itemIDs := make([]int64, len(results))
	itemMap := make(map[int64]int)
	for i, r := range results {
		itemIDs[i] = r.ItemID
		itemMap[r.ItemID] = i
	}

	var processes []models.OrderProcess
	err = r.db.Where("order_item_id IN ? AND corp_id=?", itemIDs, corpID).Order("sort_order ASC").Find(&processes).Error
	if err != nil {
		return nil, err
	}

	// 4. 组装数据
	items := make([]DashboardItem, len(results))
	for i, r := range results {
		items[i] = DashboardItem{
			OrderID:           r.OrderID,
			OrderNumber:       r.OrderNumber,
			CustomerShortName: r.CustomerShortName,
			Priority:          r.Priority,
			ItemID:            r.ItemID,
			PartName:          r.PartName,
			PartNumber:        r.PartNumber,
			Quantity:          r.Quantity,
			Status:            r.Status,
			StartDate:         r.StartDate,
			DueDate:           r.DueDate,
		}
		if r.DrawingData != nil {
			items[i].DrawingData = *r.DrawingData
		}
	}

	for _, p := range processes {
		if idx, ok := itemMap[p.OrderItemID]; ok {
			items[idx].Processes = append(items[idx].Processes, p)
		}
	}

	return &DashboardResult{
		Items:    items,
		Orders:   orderInfos,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Include partial qualified production and delivery history; do not require the whole order to finish.
const deliveryReadyFilter = `status IN ('completed','delivered') OR EXISTS (SELECT 1 FROM order_items di JOIN production_completions dq ON dq.corp_id=di.corp_id AND dq.order_item_id=di.id AND dq.status='confirmed' WHERE di.order_id=orders.id AND di.corp_id=orders.corp_id) OR EXISTS (SELECT 1 FROM order_business_records dr WHERE dr.order_id=orders.id AND dr.corp_id=orders.corp_id AND dr.kind='delivery')`
