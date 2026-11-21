// 有问题欢迎联系 hot_kun@hotmail.com
package httpserver

import (
	"net/http"
	"strconv"
	"testing"
)

// TestDashboardPendingHasNoDuplicateBills 仪表盘待办不能出现重复单据。
//
// 调拨单按设计同时属于入库与出库两条待办队列，合并时必须去重：重复的
// bill_type+bill_id 会让前端 keyed each 抛错（each_key_duplicate），
// 整个仪表盘停在加载态。
func TestDashboardPendingHasNoDuplicateBills(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	_, warehouseID, goodsID, _ := setupMaster(t, r, admin)

	w := call(t, r, http.MethodPost, "/api/v1/warehouses", `{"code":"WH2","name":"二号仓"}`, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("建二号仓: %d %s", w.Code, w.Body.String())
	}
	var wh2 struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &wh2)

	// 建并提交一张调拨单：它会同时进入 in 与 out 两侧待办
	body := `{"from_warehouse_id":` + strconv.FormatInt(warehouseID, 10) +
		`,"to_warehouse_id":` + strconv.FormatInt(wh2.ID, 10) +
		`,"items":[{"goods_id":` + strconv.FormatInt(goodsID, 10) + `,"qty":1}]}`
	w = call(t, r, http.MethodPost, "/api/v1/transfers", body, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("建调拨单: %d %s", w.Code, w.Body.String())
	}
	var tr struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &tr)
	if w := call(t, r, http.MethodPost,
		"/api/v1/transfers/"+strconv.FormatInt(tr.ID, 10)+"/submit", "", admin); w.Code != http.StatusOK {
		t.Fatalf("提交调拨单: %d %s", w.Code, w.Body.String())
	}

	w = call(t, r, http.MethodGet, "/api/v1/dashboard", "", admin)
	if w.Code != http.StatusOK {
		t.Fatalf("仪表盘: %d %s", w.Code, w.Body.String())
	}
	var stats struct {
		Pending []struct {
			BillType string `json:"bill_type"`
			BillID   int64  `json:"bill_id"`
		} `json:"pending"`
	}
	decodeData(t, w.Body.Bytes(), &stats)

	counts := map[string]int{}
	for _, b := range stats.Pending {
		counts[b.BillType+"-"+strconv.FormatInt(b.BillID, 10)]++
	}
	for key, n := range counts {
		if n > 1 {
			t.Errorf("待办 %s 出现了 %d 次，前端会因重复 key 崩掉", key, n)
		}
	}
	if got := counts["transfer-"+strconv.FormatInt(tr.ID, 10)]; got != 1 {
		t.Errorf("调拨单应恰好出现一次，实际 %d 次", got)
	}
}
