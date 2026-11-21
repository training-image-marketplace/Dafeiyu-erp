// 有问题欢迎联系 hot_kun@hotmail.com
package httpserver

import (
	"net/http"
	"strconv"
	"testing"
)

// TestDeleteDraftInventoryBills 覆盖库存调整/调拨/盘点的「删除草稿」。
//
// 这三类单据的明细表外键名各不相同（调拨用 transfer_id、盘点用 check_id，
// 调整则没有明细表），此前共用一处写死 order_id 的 DELETE，导致接口 500。
func TestDeleteDraftInventoryBills(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	_, warehouseID, goodsID, _ := setupMaster(t, r, admin)

	// 第二个仓库，供调拨使用
	w := call(t, r, http.MethodPost, "/api/v1/warehouses", `{"code":"WH2","name":"二号仓"}`, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("建二号仓: %d %s", w.Code, w.Body.String())
	}
	var wh2 struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &wh2)

	wh := strconv.FormatInt(warehouseID, 10)
	goods := strconv.FormatInt(goodsID, 10)

	// 建草稿 -> 删除 -> 再查应当 404
	assertDraftDeletable := func(t *testing.T, collection, createBody string) {
		t.Helper()
		w := call(t, r, http.MethodPost, "/api/v1/"+collection, createBody, admin)
		if w.Code != http.StatusOK {
			t.Fatalf("建草稿 %s: %d %s", collection, w.Code, w.Body.String())
		}
		var item struct {
			ID int64 `json:"id"`
		}
		decodeData(t, w.Body.Bytes(), &item)
		path := "/api/v1/" + collection + "/" + strconv.FormatInt(item.ID, 10)

		if w := call(t, r, http.MethodDelete, path, "", admin); w.Code != http.StatusOK {
			t.Fatalf("删除草稿 %s 返回 %d，期望 200: %s", path, w.Code, w.Body.String())
		}
		if w := call(t, r, http.MethodGet, path, "", admin); w.Code != http.StatusNotFound {
			t.Fatalf("删除后 %s 返回 %d，期望 404", path, w.Code)
		}
	}

	t.Run("stock adjustment", func(t *testing.T) {
		assertDraftDeletable(t, "stock-adjustments",
			`{"warehouse_id":`+wh+`,"goods_id":`+goods+`,"direction":"in","qty":1}`)
	})
	t.Run("transfer", func(t *testing.T) {
		assertDraftDeletable(t, "transfers",
			`{"from_warehouse_id":`+wh+`,"to_warehouse_id":`+strconv.FormatInt(wh2.ID, 10)+
				`,"items":[{"goods_id":`+goods+`,"qty":1}]}`)
	})
	t.Run("stock check", func(t *testing.T) {
		assertDraftDeletable(t, "stock-checks",
			`{"warehouse_id":`+wh+`,"items":[{"goods_id":`+goods+`,"actual_qty":1}]}`)
	})
}
