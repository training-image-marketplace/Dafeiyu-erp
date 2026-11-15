// 有问题欢迎联系 hot_kun@hotmail.com
package httpserver

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestTransferLifecycle 覆盖调拨单：草稿不改库存，审核后也不改，完成时才双边转移。
func TestTransferLifecycle(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	_, warehouseID, goodsID, _ := setupMaster(t, r, admin)
	approver := approveUser(t, r, admin)

	var second struct {
		ID int64 `json:"id"`
	}
	if w := call(t, r, http.MethodPost, "/api/v1/warehouses",
		`{"code":"WH2","name":"分仓"}`, admin); w.Code != http.StatusOK {
		t.Fatalf("建分仓: %d %s", w.Code, w.Body.String())
	} else {
		decodeData(t, w.Body.Bytes(), &second)
	}

	// 先入库 20 到主仓
	approvedStockAdjustment(t, r, admin, approver, warehouseID, goodsID, "in", 20)

	body := `{"from_warehouse_id":` + strconv.FormatInt(warehouseID, 10) +
		`,"to_warehouse_id":` + strconv.FormatInt(second.ID, 10) +
		`,"items":[{"goods_id":` + strconv.FormatInt(goodsID, 10) + `,"qty":8}]}`
	w := call(t, r, http.MethodPost, "/api/v1/transfers", body, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("建调拨单: %d %s", w.Code, w.Body.String())
	}
	var tr struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	}
	decodeData(t, w.Body.Bytes(), &tr)
	if tr.Status != "draft" {
		t.Fatalf("调拨单初始状态 = %q", tr.Status)
	}
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 20 {
		t.Fatalf("草稿不应改库存，got %v", got)
	}

	id := strconv.FormatInt(tr.ID, 10)
	if w := call(t, r, http.MethodPost, "/api/v1/transfers/"+id+"/submit", "", admin); w.Code != http.StatusOK {
		t.Fatalf("提交调拨: %d %s", w.Code, w.Body.String())
	}
	for _, direction := range []string{"in", "out"} {
		w := call(t, r, http.MethodGet, "/api/v1/audits?direction="+direction+"&status=submitted", "", approver)
		var queue struct {
			List []struct {
				BillType string `json:"bill_type"`
				BillID   int64  `json:"bill_id"`
			} `json:"list"`
		}
		decodeData(t, w.Body.Bytes(), &queue)
		found := false
		for _, bill := range queue.List {
			if bill.BillType == "transfer" && bill.BillID == tr.ID {
				found = true
			}
		}
		if !found {
			t.Fatalf("调拨应同时出现在 %s 待办（调出/调入两侧都要审）", direction)
		}
	}
	if w := call(t, r, http.MethodPost, "/api/v1/audits/transfer/"+id, `{"pass":true}`, approver); w.Code != http.StatusOK {
		t.Fatalf("审核调拨: %d %s", w.Code, w.Body.String())
	}
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 20 {
		t.Fatalf("审核后不应改库存，got %v", got)
	}

	if w := call(t, r, http.MethodPost, "/api/v1/transfers/"+id+"/complete", "", admin); w.Code != http.StatusOK {
		t.Fatalf("完成调拨: %d %s", w.Code, w.Body.String())
	}
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 12 {
		t.Fatalf("调出后主仓库存 = %v，期望 12", got)
	}
	if got := stockQty(t, r, admin, second.ID, goodsID); got != 8 {
		t.Fatalf("调入后分仓库存 = %v，期望 8", got)
	}

	// 库存不足时必须拒绝
	body = `{"from_warehouse_id":` + strconv.FormatInt(warehouseID, 10) +
		`,"to_warehouse_id":` + strconv.FormatInt(second.ID, 10) +
		`,"items":[{"goods_id":` + strconv.FormatInt(goodsID, 10) + `,"qty":100}]}`
	w = call(t, r, http.MethodPost, "/api/v1/transfers", body, admin)
	var over struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &over)
	overID := strconv.FormatInt(over.ID, 10)
	call(t, r, http.MethodPost, "/api/v1/transfers/"+overID+"/submit", "", admin)
	call(t, r, http.MethodPost, "/api/v1/audits/transfer/"+overID, `{"pass":true}`, approver)
	if w := call(t, r, http.MethodPost, "/api/v1/transfers/"+overID+"/complete", "", admin); w.Code != http.StatusConflict {
		t.Fatalf("库存不足调拨返回 %d，期望 409", w.Code)
	}
}

// TestTransferRejectsSameWarehouse 调出与调入必须是两个仓库。
func TestTransferRejectsSameWarehouse(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	_, warehouseID, goodsID, _ := setupMaster(t, r, admin)

	body := `{"from_warehouse_id":` + strconv.FormatInt(warehouseID, 10) +
		`,"to_warehouse_id":` + strconv.FormatInt(warehouseID, 10) +
		`,"items":[{"goods_id":` + strconv.FormatInt(goodsID, 10) + `,"qty":1}]}`
	if w := call(t, r, http.MethodPost, "/api/v1/transfers", body, admin); w.Code != http.StatusBadRequest {
		t.Fatalf("同仓调拨返回 %d，期望 400", w.Code)
	}
}

var _ = gin.H{}
