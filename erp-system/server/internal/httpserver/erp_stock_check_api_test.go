// 有问题欢迎联系 hot_kun@hotmail.com
package httpserver

import (
	"net/http"
	"strconv"
	"testing"
)

// TestStockCheckGeneratesAdjustment 覆盖盘点：账面数由服务端读取，
// 审核通过后按差异入账，并留下可追溯的库存流水。
func TestStockCheckGeneratesAdjustment(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	_, warehouseID, goodsID, _ := setupMaster(t, r, admin)
	approver := approveUser(t, r, admin)

	// 账面 20
	approvedStockAdjustment(t, r, admin, approver, warehouseID, goodsID, "in", 20)

	body := `{"warehouse_id":` + strconv.FormatInt(warehouseID, 10) +
		`,"items":[{"goods_id":` + strconv.FormatInt(goodsID, 10) + `,"actual_qty":17}]}`
	w := call(t, r, http.MethodPost, "/api/v1/stock-checks", body, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("建盘点单: %d %s", w.Code, w.Body.String())
	}
	var sc struct {
		ID    int64 `json:"id"`
		Items []struct {
			BookQty   float64 `json:"book_qty"`
			ActualQty float64 `json:"actual_qty"`
			DiffQty   float64 `json:"diff_qty"`
		} `json:"items"`
	}
	decodeData(t, w.Body.Bytes(), &sc)
	if len(sc.Items) != 1 || sc.Items[0].BookQty != 20 || sc.Items[0].DiffQty != -3 {
		t.Fatalf("盘点明细不符: %+v", sc.Items)
	}
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 20 {
		t.Fatalf("建盘点单不应改库存，got %v", got)
	}

	id := strconv.FormatInt(sc.ID, 10)
	if w := call(t, r, http.MethodPost, "/api/v1/stock-checks/"+id+"/submit", "", admin); w.Code != http.StatusOK {
		t.Fatalf("提交盘点: %d %s", w.Code, w.Body.String())
	}
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 20 {
		t.Fatalf("待审核盘点不应改库存，got %v", got)
	}

	w = call(t, r, http.MethodGet, "/api/v1/audits?direction=out&status=submitted", "", approver)
	if w.Code != http.StatusOK {
		t.Fatalf("读取入库审核待办: %d %s", w.Code, w.Body.String())
	}
	var pending struct {
		List []struct {
			BillType string `json:"bill_type"`
			BillID   int64  `json:"bill_id"`
		} `json:"list"`
	}
	decodeData(t, w.Body.Bytes(), &pending)
	foundPending := false
	for _, bill := range pending.List {
		if bill.BillType == "stock-check" && bill.BillID == sc.ID {
			foundPending = true
		}
	}
	if !foundPending {
		t.Fatalf("已提交盘点单未出现在审核中心出库待办: %+v", pending.List)
	}

	if w := call(t, r, http.MethodPost, "/api/v1/audits/stock-check/"+id, `{"pass":true}`, approver); w.Code != http.StatusOK {
		t.Fatalf("审核盘点: %d %s", w.Code, w.Body.String())
	}
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 17 {
		t.Fatalf("盘点审核后库存 = %v，期望 17", got)
	}

	// 库存流水应有一条盘点产生的调整记录
	w = call(t, r, http.MethodGet, "/api/v1/stocks?logs=1&warehouse_id="+strconv.FormatInt(warehouseID, 10), "", admin)
	var logs struct {
		List []struct {
			ChangeQty float64 `json:"change_qty"`
			BizType   string  `json:"biz_type"`
		} `json:"list"`
	}
	if w.Code != http.StatusOK {
		t.Fatalf("查流水: %d %s", w.Code, w.Body.String())
	}
	decodeData(t, w.Body.Bytes(), &logs)
	found := false
	for _, l := range logs.List {
		if l.BizType == "adjust" && l.ChangeQty == -3 {
			found = true
		}
	}
	if !found {
		t.Fatalf("盘点差异未写入库存流水: %+v", logs.List)
	}
}
