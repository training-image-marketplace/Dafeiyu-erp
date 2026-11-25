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
	Scope   string                     `json:"scope" binding:"required,oneof=all group personal"`
	ScopeID int64                      `json:"scope_id" binding:"gte=0"`
	Config  map[string]map[string]bool `json:"config" binding:"required"`
}

func (h *InternalHandler) PutSwitches(c *gin.Context) {
	var req internalSwitchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	// Schema group names are a presentation contract. Persist the canonical bugs/menus
	// structure used by business checks, while accepting legacy bugs payloads.
	cfg := domain.SwitchConfig{Bugs: map[string]bool{}, Menus: req.Config["menus"]}
	for code, on := range req.Config["bugs"] {
		cfg.Bugs[code] = on
	}
	for _, group := range domain.SwitchSchema() {
		if group.Code == "menus" {
			continue
		}
		for _, item := range group.Items {
			if on, ok := req.Config[group.Code][item.Code]; ok {
				cfg.Bugs[item.Code] = on
			}
		}
	}
	if err := h.switches.Set(c.Request.Context(), req.Scope, req.ScopeID, cfg); err != nil {
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
	// 只下发已接线的缺陷：平台面板据此渲染勾选项，列出「打开也没反应」的码
	// 会让老师以为平台下发失败。
	httpx.Success(c, gin.H{"list": domain.WiredBugSwitches()})
}

// SwitchSchema 自报开关组与项。
//
// 平台只按「组 → code」透传与下发，不内置任何组名：所以这里加一组、改个名字，
// 平台侧一行代码都不用动。分三组是给教师看的（单据 / 审批 / 库存 / 主数据 / 菜单），
// 不是按实现方式分的。
func (h *InternalHandler) SwitchSchema(c *gin.Context) {
	httpx.Success(c, gin.H{"list": domain.SwitchSchema()})
}

func (h *InternalHandler) Health(c *gin.Context) {
	httpx.Success(c, gin.H{"status": "up"})
}
