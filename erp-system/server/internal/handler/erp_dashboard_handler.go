// 有问题欢迎联系 hot_kun@hotmail.com
package handler

import (
	"github.com/gin-gonic/gin"

	"practical-training/erp-system/internal/httpx"
)

// Dashboard 返回首页聚合指标。
//
// 挂在「仅需登录」分组：仪表盘对所有角色开放，不需要额外权限码。
// 它只读、只返回计数与待办的公开字段，因此不会绕过业务接口的授权边界。
func (h *ERPHandler) Dashboard(c *gin.Context) {
	stats, err := h.svc.Dashboard(c.Request.Context())
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, stats)
}
