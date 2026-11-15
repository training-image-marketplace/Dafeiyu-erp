// 有问题欢迎联系 hot_kun@hotmail.com
package httpserver

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// TestMenuExposesButtonPermissions 钉住按钮级权限的数据来源：
// 菜单接口要返回每个页面的按钮权限码，授权页据此渲染。
func TestMenuExposesButtonPermissions(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")

	w := call(t, r, http.MethodGet, "/api/v1/menus", "", admin)
	if w.Code != http.StatusOK {
		t.Fatalf("菜单接口: %d %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"buttons", "erp:purchase:edit", "erp:audit:approve"} {
		if !strings.Contains(body, want) {
			t.Errorf("菜单响应缺少 %q", want)
		}
	}
}

// TestApproveRequiresAuditPermission 确认审批只认 erp:audit:approve：
// 只有采购建单权限的人能提交，但审批必须 403；审核中心的人可以审。
func TestApproveRequiresAuditPermission(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	supplierID, warehouseID, goodsID, _ := setupMaster(t, r, admin)

	// 只有建单权限的角色
	var role struct {
		ID int64 `json:"id"`
	}
	w := call(t, r, http.MethodPost, "/api/v1/roles", `{"code":"buyer-only","name":"采购员"}`, admin)
	decodeData(t, w.Body.Bytes(), &role)
	if w := call(t, r, http.MethodPut,
		"/api/v1/roles/"+strconv.FormatInt(role.ID, 10)+"/grants",
		`{"menu_ids":[],"perm_codes":["erp:purchase:list","erp:purchase:edit"]}`, admin); w.Code != http.StatusOK {
		t.Fatalf("授权: %d %s", w.Code, w.Body.String())
	}
	buyerID := createUser(t, r, admin, "buyer-only", "buyer-only-pw")
	if w := call(t, r, http.MethodPut, "/api/v1/users/"+strconv.FormatInt(buyerID, 10)+"/roles",
		`{"role_ids":[`+strconv.FormatInt(role.ID, 10)+`]}`, admin); w.Code != http.StatusOK {
		t.Fatalf("给用户授权: %d %s", w.Code, w.Body.String())
	}
	buyer := login(t, r, "buyer-only", "buyer-only-pw")

	body := `{"supplier_id":` + strconv.FormatInt(supplierID, 10) + `,"warehouse_id":` +
		strconv.FormatInt(warehouseID, 10) + `,"items":[{"goods_id":` +
		strconv.FormatInt(goodsID, 10) + `,"qty":1,"price":5}]}`
	w = call(t, r, http.MethodPost, "/api/v1/purchases", body, buyer)
	if w.Code != http.StatusOK {
		t.Fatalf("建采购单: %d %s", w.Code, w.Body.String())
	}
	var po struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &po)
	id := strconv.FormatInt(po.ID, 10)

	if w := call(t, r, http.MethodPost, "/api/v1/purchases/"+id+"/submit", "", buyer); w.Code != http.StatusOK {
		t.Fatalf("提交采购: %d %s", w.Code, w.Body.String())
	}
	if w := call(t, r, http.MethodPost, "/api/v1/purchases/"+id+"/approve", "", buyer); w.Code != http.StatusForbidden {
		t.Fatalf("无审核权限审批应 403，got %d %s", w.Code, w.Body.String())
	}
	if w := call(t, r, http.MethodPost, "/api/v1/audits/purchase/"+id, `{"pass":true}`, admin); w.Code != http.StatusOK {
		t.Fatalf("审核中心审批: %d %s", w.Code, w.Body.String())
	}
}
