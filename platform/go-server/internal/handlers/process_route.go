package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/internal/repository"
	"machining-erp/pkg/response"
	"machining-erp/pkg/utils"
	"strconv"
	"strings"
	"unicode/utf8"
)

func (h *ProcessTimelineHandler) SaveRoute(c *gin.Context) {
	if !h.canWrite(c) {
		response.Forbidden(c, response.CodeForbidden, "当前账号无加工工艺修改权限")
		return
	}
	itemID, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil || itemID <= 0 {
		response.BadRequest(c, response.CodeInvalidRequest, "零件编号无效")
		return
	}
	var req struct {
		AllowProcessedChanges bool   `json:"allow_processed_changes"`
		Reason                string `json:"reason"`
		Processes             []struct {
			ID           int64   `json:"id"`
			Name         string  `json:"name"`
			Outsourced   bool    `json:"is_outsourced"`
			PlannedStart *string `json:"planned_start_at"`
			PlannedEnd   *string `json:"planned_end_at"`
		} `json:"processes"`
	}
	if c.ShouldBindJSON(&req) != nil || len(req.Processes) == 0 || len(req.Processes) > 100 {
		response.BadRequest(c, response.CodeInvalidRequest, "请保留1至100道工序")
		return
	}
	incoming := []models.OrderProcess{}
	plans := map[int64]bool{}
	delivery := 0
	for _, v := range req.Processes {
		name := strings.TrimSpace(v.Name)
		if name == "" || utf8.RuneCountInString(name) > 100 || v.ID < 0 {
			response.BadRequest(c, response.CodeInvalidRequest, "工序名称须为1至100个字，编号须有效")
			return
		}
		if name == "送货" {
			delivery++
		}
		if delivery > 1 {
			response.BadRequest(c, response.CodeInvalidRequest, "送货节点只能有一个")
			return
		}
		if (v.PlannedStart == nil) != (v.PlannedEnd == nil) {
			response.BadRequest(c, response.CodeInvalidRequest, "计划开始和结束日期须一起提供")
			return
		}
		p := models.OrderProcess{ID: v.ID, Name: name, IsOutsourced: v.Outsourced, Status: models.ProcessStatusPending}
		if p.PlannedStartAt, err = parseProcessNode(v.PlannedStart, nil); err != nil {
			response.BadRequest(c, response.CodeInvalidRequest, err.Error())
			return
		}
		if p.PlannedEndAt, err = parseProcessNode(v.PlannedEnd, nil); err != nil {
			response.BadRequest(c, response.CodeInvalidRequest, err.Error())
			return
		}
		if err = utils.ValidateProcessPlan(p.PlannedStartAt, p.PlannedEndAt); err != nil {
			response.BadRequest(c, response.CodeInvalidRequest, err.Error())
			return
		}
		plans[p.ID] = v.PlannedStart != nil
		incoming = append(incoming, p)
	}
	err = repository.NewOrderRepository(h.db).UpdateRouteForCorp(middleware.GetCorpID(c), itemID, middleware.GetUserID(c), incoming, plans, repository.RouteChangeOptions{AllowProcessedChanges: req.AllowProcessedChanges, Reason: req.Reason})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.ErrorWithCode(c, 404, response.CodeNotFound, "零件或工序不存在")
		return
	}
	if err != nil {
		response.BadRequest(c, response.CodeInvalidRequest, err.Error())
		return
	}
	response.SuccessWithMessage(c, "加工工艺已保存，移除的工序已归档，原有生产和费用记录保持不变", nil)
}
