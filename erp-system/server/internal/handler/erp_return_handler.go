// 有问题欢迎联系 hot_kun@hotmail.com
package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"practical-training/erp-system/internal/domain"
	"practical-training/erp-system/internal/httpx"
	"practical-training/erp-system/internal/repository"
)

// ---------- 退货单请求体 ----------

type returnItemRequest struct {
	GoodsID int64   `json:"goods_id" binding:"required,gt=0"`
	Qty     float64 `json:"qty" binding:"required,gt=0"`
}

type returnCreateRequest struct {
	SourceOrderID int64               `json:"source_order_id" binding:"required,gt=0"`
	Items         []returnItemRequest `json:"items" binding:"required,min=1,max=100,dive"`
	Remark        string              `json:"remark" binding:"max=255"`
}

type returnUpdateRequest struct {
	Items  []returnItemRequest `json:"items" binding:"required,min=1,max=100,dive"`
	Remark string              `json:"remark" binding:"max=255"`
}

func toReturnItems(in []returnItemRequest) []domain.OrderItem {
	out := make([]domain.OrderItem, 0, len(in))
	for _, it := range in {
		out = append(out, domain.OrderItem{GoodsID: it.GoodsID, Qty: it.Qty})
	}
	return out
}

func returnFilter(c *gin.Context) repository.ReturnFilter {
	return repository.ReturnFilter{
		Status:        c.Query("status"),
		Keyword:       c.Query("keyword"),
		SourceOrderID: queryInt64(c, "source_order_id"),
	}
}

// ---------- 采购退货 ----------

func (h *ERPHandler) ListPurchaseReturns(c *gin.Context) {
	if !h.menuEnabled(c, "purchase_return") {
		return
	}
	page, pageSize := pageParams(c)
	res, err := h.svc.ListPurchaseReturns(c.Request.Context(), returnFilter(c), page, pageSize)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, res)
}

func (h *ERPHandler) GetPurchaseReturn(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	item, err := h.svc.GetPurchaseReturn(c.Request.Context(), id)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, item)
}

func (h *ERPHandler) CreatePurchaseReturn(c *gin.Context) {
	uid, uname, ok := currentOperator(c)
	if !ok {
		return
	}
	var req returnCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	item, err := h.svc.CreatePurchaseReturn(c.Request.Context(),
		req.SourceOrderID, toReturnItems(req.Items), req.Remark, uid, uname)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, item)
}

func (h *ERPHandler) UpdatePurchaseReturn(c *gin.Context) {
	uid, uname, ok := currentOperator(c)
	if !ok {
		return
	}
	id, err := pathID(c)
	if err != nil {
		return
	}
	var req returnUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	if err := h.svc.UpdatePurchaseReturn(c.Request.Context(),
		id, toReturnItems(req.Items), req.Remark, uid, uname); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

func (h *ERPHandler) SubmitPurchaseReturn(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.SubmitPurchaseReturn(c.Request.Context(), id, uid, uname)
	})
}

func (h *ERPHandler) ApprovePurchaseReturn(c *gin.Context) {
	var req auditActionRequest
	if !bindOptionalJSON(c, &req) {
		return
	}
	scope, scopeID := scopeFrom(c)
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.ApprovePurchaseReturn(c.Request.Context(), id, uid, uname, req.Comment, scope, scopeID)
	})
}

func (h *ERPHandler) RejectPurchaseReturn(c *gin.Context) {
	h.reject(c, func(id, uid int64, uname, comment string) error {
		scope, scopeID := scopeFrom(c)
		return h.svc.RejectPurchaseReturn(c.Request.Context(), id, uid, uname, comment, scope, scopeID)
	})
}

func (h *ERPHandler) WithdrawPurchaseReturn(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.WithdrawPurchaseReturn(c.Request.Context(), id, uid, uname)
	})
}

func (h *ERPHandler) CompletePurchaseReturn(c *gin.Context) {
	scope, scopeID := scopeFrom(c)
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.CompletePurchaseReturn(c.Request.Context(), id, uid, uname, scope, scopeID)
	})
}

func (h *ERPHandler) DeleteDraftPurchaseReturn(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.DeleteDraftPurchaseReturn(c.Request.Context(), id, uid, uname)
	})
}

// ---------- 销售退货 ----------

func (h *ERPHandler) ListSalesReturns(c *gin.Context) {
	if !h.menuEnabled(c, "sales_return") {
		return
	}
	page, pageSize := pageParams(c)
	res, err := h.svc.ListSalesReturns(c.Request.Context(), returnFilter(c), page, pageSize)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, res)
}

func (h *ERPHandler) GetSalesReturn(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	item, err := h.svc.GetSalesReturn(c.Request.Context(), id)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, item)
}

func (h *ERPHandler) CreateSalesReturn(c *gin.Context) {
	uid, uname, ok := currentOperator(c)
	if !ok {
		return
	}
	var req returnCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	item, err := h.svc.CreateSalesReturn(c.Request.Context(),
		req.SourceOrderID, toReturnItems(req.Items), req.Remark, uid, uname)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, item)
}

func (h *ERPHandler) UpdateSalesReturn(c *gin.Context) {
	uid, uname, ok := currentOperator(c)
	if !ok {
		return
	}
	id, err := pathID(c)
	if err != nil {
		return
	}
	var req returnUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	if err := h.svc.UpdateSalesReturn(c.Request.Context(),
		id, toReturnItems(req.Items), req.Remark, uid, uname); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

func (h *ERPHandler) SubmitSalesReturn(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.SubmitSalesReturn(c.Request.Context(), id, uid, uname)
	})
}

func (h *ERPHandler) ApproveSalesReturn(c *gin.Context) {
	var req auditActionRequest
	if !bindOptionalJSON(c, &req) {
		return
	}
	scope, scopeID := scopeFrom(c)
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.ApproveSalesReturn(c.Request.Context(), id, uid, uname, req.Comment, scope, scopeID)
	})
}

func (h *ERPHandler) RejectSalesReturn(c *gin.Context) {
	h.reject(c, func(id, uid int64, uname, comment string) error {
		scope, scopeID := scopeFrom(c)
		return h.svc.RejectSalesReturn(c.Request.Context(), id, uid, uname, comment, scope, scopeID)
	})
}

func (h *ERPHandler) WithdrawSalesReturn(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.WithdrawSalesReturn(c.Request.Context(), id, uid, uname)
	})
}

func (h *ERPHandler) CompleteSalesReturn(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.CompleteSalesReturn(c.Request.Context(), id, uid, uname)
	})
}

func (h *ERPHandler) DeleteDraftSalesReturn(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.DeleteDraftSalesReturn(c.Request.Context(), id, uid, uname)
	})
}

// ---------- 审核中心 ----------

func (h *ERPHandler) ListAudits(c *gin.Context) {
	if !h.menuEnabled(c, "audit") {
		return
	}
	page, pageSize := pageParams(c)
	res, err := h.svc.ListAuditBills(c.Request.Context(),
		c.DefaultQuery("direction", "in"),
		repository.AuditFilter{Status: c.Query("status"), Keyword: c.Query("keyword")},
		page, pageSize)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, res)
}

type auditBillRequest struct {
	// Pass 用指针：false 是合法值，不能用 binding:"required" 判。
	Pass    *bool  `json:"pass"`
	Comment string `json:"comment" binding:"max=500"`
}

func (h *ERPHandler) AuditBill(c *gin.Context) {
	billType := c.Param("type")
	if !validBillType(billType) {
		httpx.Fail(c, http.StatusBadRequest, httpx.KeyInvalidID)
		return
	}
	uid, uname, ok := currentOperator(c)
	if !ok {
		return
	}
	id, err := pathID(c)
	if err != nil {
		return
	}
	var req auditBillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	if req.Pass == nil {
		httpx.Fail(c, http.StatusBadRequest, httpx.KeyValidationFailed)
		return
	}
	if !*req.Pass && strings.TrimSpace(req.Comment) == "" {
		httpx.Fail(c, http.StatusBadRequest, httpx.KeyValidationFailed)
		return
	}
	scope, scopeID := scopeFrom(c)
	if err := h.svc.AuditBill(c.Request.Context(), billType, id, uid, uname,
		req.Comment, scope, scopeID, *req.Pass); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id, "pass": *req.Pass})
}
