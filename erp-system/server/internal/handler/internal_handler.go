// 有问题欢迎联系 hot_kun@hotmail.com
package handler

import (
	"crypto/subtle"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"practical-training/erp-system/internal/domain"
	"practical-training/erp-system/internal/httpx"
	"practical-training/erp-system/internal/service"
)

// InternalHandler 只给平台调用，不经过用户会话。
type InternalHandler struct {
	switches *service.SwitchService
}

func NewInternalHandler(switches *service.SwitchService) *InternalHandler {
	return &InternalHandler{switches: switches}
}

// RequireInternalToken 校验内部共享令牌。
//
// 令牌为空时只允许本机访问：本地开发不必配令牌，但绝不能让「没配令牌」
// 等价于「谁都能改开关」。常量时间比较避免时序侧信道。
func RequireInternalToken(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token == "" {
			ip := c.ClientIP()
			if ip != "127.0.0.1" && ip != "::1" {
				httpx.Fail(c, http.StatusUnauthorized, httpx.KeyUnauthorized)
				c.Abort()
				return
			}
			c.Next()
			return
		}
		got := c.GetHeader("X-Internal-Token")
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			httpx.Fail(c, http.StatusUnauthorized, httpx.KeyUnauthorized)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *InternalHandler) GetSwitches(c *gin.Context) {
	scope := c.DefaultQuery("scope", "all")
	var scopeID int64
	if v := c.Query("scope_id"); v != "" {
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			httpx.Fail(c, http.StatusBadRequest, httpx.KeyInvalidID)
			return
		}
		scopeID = parsed
	}
	cfg, err := h.switches.Config(c.Request.Context(), scope, scopeID)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"scope": scope, "scope_id": scopeID, "config": cfg})
}

type internalSwitchRequest struct {
	Scope   string              `json:"scope" binding:"required,oneof=all group personal"`
	ScopeID int64               `json:"scope_id" binding:"gte=0"`
	Config  domain.SwitchConfig `json:"config" binding:"required"`
}

func (h *InternalHandler) PutSwitches(c *gin.Context) {
	var req internalSwitchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	if err := h.switches.Set(c.Request.Context(), req.Scope, req.ScopeID, req.Config); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"scope": req.Scope, "scope_id": req.ScopeID})
}

// Bugs 返回缺陷清单，供平台侧渲染开关面板。
// Menus 返回可关闭的菜单清单，平台据此渲染开关面板。
func (h *InternalHandler) Menus(c *gin.Context) {
	httpx.Success(c, gin.H{"list": domain.MenuSwitches})
}

func (h *InternalHandler) Bugs(c *gin.Context) {
	httpx.Success(c, gin.H{"list": []gin.H{
		{"code": domain.BugAmountError, "name": "金额少算 1 分", "area": "purchase/sales"},
		{"code": domain.BugNegativeStock, "name": "出库允许负库存", "area": "sales"},
		{"code": domain.BugApproveSelf, "name": "允许自审自批", "area": "purchase/sales"},
		{"code": domain.BugEditCompleted, "name": "已完成单据仍可编辑", "area": "purchase/sales"},
	}})
}

func (h *InternalHandler) Health(c *gin.Context) {
	httpx.Success(c, gin.H{"status": "up"})
}
