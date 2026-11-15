// 有问题欢迎联系 hot_kun@hotmail.com
package httpserver

import (
	"net/http"
	"strconv"
	"testing"
)

// TestDashboardAccessibleToAllRoles 验证仪表盘对所有登录角色开放：
// 即使一个角色没有任何权限码，也能看到聚合概览，但业务接口仍然 403。
func TestDashboardAccessibleToAllRoles(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	supplierID, warehouseID, goodsID, _ := setupMaster(t, r, admin)

	// 造一张待审核采购单，保证待办不是 0。
	body := `{"supplier_id":` + strconv.FormatInt(supplierID, 10) +
		`,"warehouse_id":` + strconv.FormatInt(warehouseID, 10) +
		`,"items":[{"goods_id":` + strconv.FormatInt(goodsID, 10) + `,"qty":4,"price":6}]}`
	w := call(t, r, http.MethodPost, "/api/v1/purchases", body, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("建采购单: %d %s", w.Code, w.Body.String())
	}
	var po struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &po)
	if w := call(t, r, http.MethodPost,
		"/api/v1/purchases/"+strconv.FormatInt(po.ID, 10)+"/submit", "", admin); w.Code != http.StatusOK {
		t.Fatalf("提交采购单: %d %s", w.Code, w.Body.String())
	}

	// 一个完全没有角色授权的用户：能登录、能看仪表盘，但不能看业务列表。
	createUser(t, r, admin, "viewer", "viewer-pw")
	viewer := login(t, r, "viewer", "viewer-pw")

	w = call(t, r, http.MethodGet, "/api/v1/dashboard", "", viewer)
	if w.Code != http.StatusOK {
		t.Fatalf("无权限用户访问仪表盘: %d %s", w.Code, w.Body.String())
	}
	var stats struct {
		GoodsCount    int64 `json:"goods_count"`
		CustomerCount int64 `json:"customer_count"`
		PendingTotal  int64 `json:"pending_total"`
		Pending       []struct {
			BillType string `json:"bill_type"`
			BillNo   string `json:"bill_no"`
		} `json:"pending"`
	}
	decodeData(t, w.Body.Bytes(), &stats)
	if stats.GoodsCount < 1 || stats.CustomerCount < 1 {
		t.Errorf("仪表盘主数据计数偏低: %+v", stats)
	}
	if stats.PendingTotal < 1 || len(stats.Pending) == 0 {
		t.Errorf("仪表盘待办应当包含刚提交的采购单: %+v", stats)
	}
	if stats.Pending[0].BillType != "purchase" || stats.Pending[0].BillNo == "" {
		t.Errorf("待办首条不是采购单: %+v", stats.Pending[0])
	}

	if w := call(t, r, http.MethodGet, "/api/v1/purchases", "", viewer); w.Code != http.StatusForbidden {
		t.Errorf("无权限用户访问采购列表返回 %d，期望 403", w.Code)
	}
}
