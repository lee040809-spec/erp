package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/internal/productionflow"
	"machining-erp/internal/repository"
	"machining-erp/pkg/response"
	"machining-erp/pkg/utils"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type ProcessTimelineHandler struct{ db *gorm.DB }

func NewProcessTimelineHandler(db *gorm.DB) *ProcessTimelineHandler {
	return &ProcessTimelineHandler{db}
}
func (h *ProcessTimelineHandler) canWrite(c *gin.Context) bool {
	return hasModuleAccess(h.db, c, "orders", "production_orders", "production_dashboard")
}
func (h *ProcessTimelineHandler) List(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "orders", "production_orders", "dashboard", "production_dashboard", "overdue", "warning_orders", "imminent_orders", "production_overdue", "production_warning", "production_imminent", "production_delivery", "reconciliation") {
		return
	}
	itemID, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	corp := middleware.GetCorpID(c)
	var item models.OrderItem
	if err != nil || itemID <= 0 || h.db.Table("order_items i").Joins("JOIN orders o ON o.id=i.order_id AND o.corp_id=i.corp_id").Select("i.id,i.order_id,i.part_name,i.due_date,i.quantity,COALESCE(i.start_date,o.start_date) AS start_date").Where("i.id=? AND i.corp_id=?", itemID, corp).Take(&item).Error != nil {
		response.ErrorWithCode(c, 404, response.CodeNotFound, "零件不存在")
		return
	}
	processes := []models.OrderProcess{}
	if err = h.db.Omit("outsourcing_fee").Where("order_item_id=? AND corp_id=?", itemID, corp).Order("sort_order,id").Find(&processes).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取工序时间失败")
		return
	}
	archived := []models.OrderProcess{}
	if err = h.db.Unscoped().Omit("outsourcing_fee").Where("order_item_id=? AND corp_id=? AND deleted_at IS NOT NULL", itemID, corp).Order("deleted_at DESC,id DESC").Limit(100).Find(&archived).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取历史工序失败")
		return
	}
	events := []models.ProcessTimeEvent{}
	if err = h.db.Where("order_item_id=? AND corp_id=?", itemID, corp).Order("id DESC").Limit(50).Find(&events).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取时间修改记录失败")
		return
	}
	deliveries, err := repository.LoadDeliverySummaries(h.db, corp, []int64{itemID})
	if err != nil {
		response.InternalError(c, response.CodeInternal, "读取送货节点失败")
		return
	}
	response.Success(c, gin.H{"archived_processes": withoutAmounts(archived), "processes": withoutAmounts(processes), "events": events, "delivery": deliveries[itemID], "start_date": item.StartDate, "due_date": item.DueDate, "quantity": item.Quantity, "can_manage": h.canWrite(c), "timezone": "Asia/Shanghai"})
}

type timelineRequest struct {
	PlannedStart *string `json:"planned_start_at"`
	PlannedEnd   *string `json:"planned_end_at"`
	ActualStart  *string `json:"actual_start_at"`
	ActualEnd    *string `json:"actual_end_at"`
	Reason       string  `json:"reason"`
}

var timelineZone = time.FixedZone("Asia/Shanghai", 8*3600)

func parseProcessNode(s *string, current *time.Time) (*time.Time, error) {
	if s == nil {
		return current, nil
	}
	if strings.TrimSpace(*s) == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		t, err = time.ParseInLocation("2006-01-02T15:04", *s, timelineZone)
	}
	if err != nil {
		t, err = time.ParseInLocation("2006-01-02", *s, timelineZone)
	}
	if err != nil || t.Year() < 2000 || t.Year() > 2100 {
		return nil, errors.New("时间格式无效，请填写2000至2100年的日期与时间")
	}
	t = t.In(time.Local)
	return &t, nil
}
func validateProcessNodes(p models.OrderProcess, now time.Time) error {
	if err := utils.ValidateProcessPlan(p.PlannedStartAt, p.PlannedEndAt); err != nil {
		return err
	}
	if p.ActualStartAt != nil && utils.ProcessDay(*p.ActualStartAt).After(utils.ProcessDay(now)) {
		return errors.New("实际开工日期不能晚于今天")
	}
	if p.ActualEndAt != nil && utils.ProcessDay(*p.ActualEndAt).After(utils.ProcessDay(now)) {
		return errors.New("实际完工日期不能晚于今天")
	}
	if p.ActualStartAt != nil && p.ActualEndAt != nil && utils.ProcessDay(*p.ActualEndAt).Before(utils.ProcessDay(*p.ActualStartAt)) {
		return errors.New("实际完工日期不能早于实际开工日期")
	}
	return nil
}
func (h *ProcessTimelineHandler) Save(c *gin.Context) {
	if !h.canWrite(c) {
		response.Forbidden(c, response.CodeForbidden, "当前账号无工序时间修改权限")
		return
	}
	itemID, e1 := strconv.ParseInt(c.Param("itemId"), 10, 64)
	processID, e2 := strconv.ParseInt(c.Param("processId"), 10, 64)
	if e1 != nil || e2 != nil || itemID <= 0 || processID <= 0 {
		response.BadRequest(c, response.CodeInvalidRequest, "零件或工序编号无效")
		return
	}
	var req timelineRequest
	if c.ShouldBindJSON(&req) != nil || utf8.RuneCountInString(req.Reason) > 500 {
		response.BadRequest(c, response.CodeInvalidRequest, "时间节点数据无效")
		return
	}
	if req.PlannedStart == nil && req.PlannedEnd == nil && req.ActualStart == nil && req.ActualEnd == nil {
		response.BadRequest(c, response.CodeInvalidRequest, "请填写需要更新的时间节点")
		return
	}
	corp, actor := middleware.GetCorpID(c), middleware.GetUserID(c)
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var item models.OrderItem
		if err := tx.Where("id=? AND corp_id=?", itemID, corp).Take(&item).Error; err != nil {
			return err
		}
		var order models.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", item.OrderID, corp).Take(&order).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", itemID, corp).Take(&item).Error; err != nil {
			return err
		}
		var before models.OrderProcess
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND order_item_id=? AND corp_id=?", processID, itemID, corp).Take(&before).Error; err != nil {
			return err
		}
		if before.Name == "送货" {
			return errors.New("送货请在交货记录中登记实际日期")
		}
		next := before
		var err error
		if next.PlannedStartAt, err = parseProcessNode(req.PlannedStart, before.PlannedStartAt); err != nil {
			return err
		}
		if next.PlannedEndAt, err = parseProcessNode(req.PlannedEnd, before.PlannedEndAt); err != nil {
			return err
		}
		if next.ActualStartAt, err = parseProcessNode(req.ActualStart, before.ActualStartAt); err != nil {
			return err
		}
		if next.ActualEndAt, err = parseProcessNode(req.ActualEnd, before.ActualEndAt); err != nil {
			return err
		}
		if err = validateProcessNodes(next, time.Now()); err != nil {
			return err
		}
		actualChanged := req.ActualStart != nil || req.ActualEnd != nil
		if actualChanged {
			on, e := productionflow.Enabled(tx, corp, itemID)
			if e != nil {
				return e
			}
			if on {
				return errors.New("数量跟踪已启用，实际日期由工序数量报工自动记录，请在分批生产中登记")
			}
		}
		if actualChanged {
			if strings.TrimSpace(req.Reason) == "" {
				return errors.New("补录或更正实际时间须填写原因")
			}
			switch {
			case next.ActualEndAt != nil:
				next.Status = models.ProcessStatusCompleted
			case next.ActualStartAt != nil:
				next.Status = models.ProcessStatusProcessing
			default:
				next.Status = models.ProcessStatusPending
			}
		}
		if err = tx.Model(&before).Updates(map[string]interface{}{"planned_start_at": next.PlannedStartAt, "planned_end_at": next.PlannedEndAt, "actual_start_at": next.ActualStartAt, "actual_end_at": next.ActualEndAt, "status": next.Status}).Error; err != nil {
			return err
		}
		// Use the existing transaction and status/qualified-count rules; this never creates a completion or changes output value.
		if next.Status != before.Status {
			if _, _, err = repository.NewOrderRepository(tx).UpdateProcessForCorp(corp, itemID, processID, nil, nil, nil); err != nil {
				return err
			}
		}
		return repository.RecordProcessTimeEvent(tx, before, next, actor, "manual", strings.TrimSpace(req.Reason))
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.ErrorWithCode(c, 404, response.CodeNotFound, "零件或工序不存在")
		return
	}
	if err != nil {
		response.BadRequest(c, response.CodeInvalidRequest, err.Error())
		return
	}
	response.SuccessWithMessage(c, "工序时间节点已保存", nil)
}
