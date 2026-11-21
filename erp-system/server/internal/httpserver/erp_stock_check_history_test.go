// 有问题欢迎联系 hot_kun@hotmail.com
package httpserver

import (
	"net/http"
	"strconv"
	"testing"
)

// TestApprovedStockCheckVisibleInAuditHistory 审核中心的状态筛选对盘点单必须同样生效。
//
// 此前盘点 CTE 写死了 status = submitted，把筛选切到「已通过」时，
// 采购/销售/调拨都有历史记录，盘点却凭空消失。
func TestApprovedStockCheckVisibleInAuditHistory(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	_, warehouseID, goodsID, _ := setupMaster(t, r, admin)
	approver := approveUser(t, r, admin)

	// 账面 0、实盘 2 -> 净差异为正，归入库侧
	body := `{"warehouse_id":` + strconv.FormatInt(warehouseID, 10) +
		`,"items":[{"goods_id":` + strconv.FormatInt(goodsID, 10) + `,"actual_qty":2}]}`
	w := call(t, r, http.MethodPost, "/api/v1/stock-checks", body, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("建盘点单: %d %s", w.Code, w.Body.String())
	}
	var sc struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &sc)
	id := strconv.FormatInt(sc.ID, 10)
	if w := call(t, r, http.MethodPost, "/api/v1/stock-checks/"+id+"/submit", "", admin); w.Code != http.StatusOK {
		t.Fatalf("提交盘点: %d %s", w.Code, w.Body.String())
	}
	if w := call(t, r, http.MethodPost, "/api/v1/audits/stock-check/"+id, `{"pass":true}`, approver); w.Code != http.StatusOK {
		t.Fatalf("审核盘点: %d %s", w.Code, w.Body.String())
	}

	w = call(t, r, http.MethodGet, "/api/v1/audits?direction=in&status=approved", "", admin)
	if w.Code != http.StatusOK {
		t.Fatalf("读取已通过审核记录: %d %s", w.Code, w.Body.String())
	}
	var history struct {
		List []struct {
			BillType string `json:"bill_type"`
			BillID   int64  `json:"bill_id"`
		} `json:"list"`
	}
	decodeData(t, w.Body.Bytes(), &history)
	for _, bill := range history.List {
		if bill.BillType == "stock-check" && bill.BillID == sc.ID {
			return
		}
	}
	t.Fatalf("已通过的盘点单未出现在审核中心历史里: %+v", history.List)
}
