package handlers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"machining-erp/internal/middleware"
	"machining-erp/pkg/response"
	"strconv"
	"strings"
	"unicode/utf8"
)

type EvidencePolicy struct {
	SourceRef       bool `json:"source_ref"`
	ReviewNotes     bool `json:"review_notes"`
	AllocationBasis bool `json:"allocation_basis"`
	ClosingNotes    bool `json:"closing_notes"`
	OperationNotes  bool `json:"operation_notes"`
}
type BusinessPolicyRow struct {
	CorpID    int64 `gorm:"primaryKey;autoIncrement:false"`
	Policy    string
	RoleModes string
}

func (BusinessPolicyRow) TableName() string { return "business_entry_policies" }
func effectiveEvidencePolicy(db *gorm.DB, c *gin.Context) (EvidencePolicy, error) {
	var row BusinessPolicyRow
	var policy EvidencePolicy
	err := db.Where("corp_id=?", middleware.GetCorpID(c)).Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return policy, nil
	}
	if err != nil {
		return policy, err
	}
	if err = json.Unmarshal([]byte(row.Policy), &policy); err != nil {
		return policy, err
	}
	modes := map[string]string{}
	if err = json.Unmarshal([]byte(row.RoleModes), &modes); err != nil {
		return policy, err
	}
	if len(modes) > 0 {
		var roleID int64
		if err = db.Table("users").Where("id=? AND corp_id=?", middleware.GetUserID(c), middleware.GetCorpID(c)).Select("role_id").Scan(&roleID).Error; err != nil {
			return policy, err
		}
		if mode := modes[strconv.FormatInt(roleID, 10)]; mode == "basic" {
			policy = EvidencePolicy{}
		} else if mode == "detailed" {
			policy = EvidencePolicy{true, true, true, true, true}
		}
	}
	return policy, nil
}
func checkEvidence(db *gorm.DB, c *gin.Context, key, value string, maxLen int) bool {
	policy, err := effectiveEvidencePolicy(db, c)
	if err != nil {
		response.InternalError(c, response.CodeInternal, "读取填写要求失败，请联系管理员")
		return false
	}
	required := map[string]bool{"source_ref": policy.SourceRef, "review_notes": policy.ReviewNotes, "allocation_basis": policy.AllocationBasis, "closing_notes": policy.ClosingNotes, "operation_notes": policy.OperationNotes}[key]
	if utf8.RuneCountInString(value) > maxLen || (required && strings.TrimSpace(value) == "") {
		response.BadRequest(c, response.CodeInvalidRequest, "凭证说明超过长度限制，或管理员要求填写该项凭证信息")
		return false
	}
	return true
}

type BusinessPolicyHandler struct{ db *gorm.DB }

func NewBusinessPolicyHandler(db *gorm.DB) *BusinessPolicyHandler { return &BusinessPolicyHandler{db} }
func (h *BusinessPolicyHandler) Get(c *gin.Context) {
	policy, err := effectiveEvidencePolicy(h.db, c)
	if err != nil {
		response.InternalError(c, response.CodeInternal, "读取填写要求失败")
		return
	}
	data := gin.H{"effective": policy, "can_configure": middleware.GetRoleType(c) == "admin"}
	if middleware.GetRoleType(c) == "admin" {
		row := BusinessPolicyRow{Policy: "{}", RoleModes: "{}"}
		err = h.db.Where("corp_id=?", middleware.GetCorpID(c)).Take(&row).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			response.InternalError(c, response.CodeInternal, "读取配置失败")
			return
		}
		var defaults EvidencePolicy
		var modes map[string]string
		json.Unmarshal([]byte(row.Policy), &defaults)
		json.Unmarshal([]byte(row.RoleModes), &modes)
		roles := []map[string]interface{}{}
		if err = h.db.Table("roles").Select("id,name").Where("corp_id=?", middleware.GetCorpID(c)).Find(&roles).Error; err != nil {
			response.InternalError(c, response.CodeInternal, "读取角色失败")
			return
		}
		data["defaults"] = defaults
		data["role_modes"] = modes
		data["roles"] = roles
	}
	response.Success(c, data)
}
func (h *BusinessPolicyHandler) Save(c *gin.Context) {
	if middleware.GetRoleType(c) != "admin" {
		response.Forbidden(c, response.CodeForbidden, "仅系统管理员可修改填写要求")
		return
	}
	var req struct {
		Defaults  EvidencePolicy    `json:"defaults"`
		RoleModes map[string]string `json:"role_modes"`
	}
	if c.ShouldBindJSON(&req) != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "配置格式错误")
		return
	}
	for key, mode := range req.RoleModes {
		id, err := strconv.ParseInt(key, 10, 64)
		var count int64
		if err != nil || id <= 0 || (mode != "basic" && mode != "detailed") {
			response.BadRequest(c, response.CodeInvalidRequest, "角色配置错误")
			return
		}
		if h.db.Table("roles").Where("id=? AND corp_id=?", id, middleware.GetCorpID(c)).Count(&count).Error != nil || count != 1 {
			response.BadRequest(c, response.CodeInvalidRequest, "角色不属于当前公司")
			return
		}
	}
	policy, _ := json.Marshal(req.Defaults)
	modes := []byte("{}")
	if req.RoleModes != nil {
		modes, _ = json.Marshal(req.RoleModes)
	}
	row := BusinessPolicyRow{middleware.GetCorpID(c), string(policy), string(modes)}
	if h.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "corp_id"}}, DoUpdates: clause.AssignmentColumns([]string{"policy", "role_modes"})}).Create(&row).Error != nil {
		response.InternalError(c, response.CodeInternal, "保存配置失败")
		return
	}
	response.SuccessWithMessage(c, "填写要求已保存；不改变角色原有功能权限", nil)
}
