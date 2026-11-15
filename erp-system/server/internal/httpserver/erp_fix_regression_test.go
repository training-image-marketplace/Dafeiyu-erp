// 有问题欢迎联系 hot_kun@hotmail.com
package httpserver

import (
	"net/http"
	"strconv"
	"testing"
)

// TestStockAdjustmentRefusesNegativeStock 覆盖出库调整不能把库存减成负数。
func TestStockAdjustmentRefusesNegativeStock(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	_, warehouseID, goodsID, _ := setupMaster(t, r, admin)
	approver := approveUser(t, r, admin)
	approvedStockAdjustment(t, r, admin, approver, warehouseID, goodsID, "in", 5)

	w := call(t, r, http.MethodPost, "/api/v1/stock-adjustments",
		`{"warehouse_id":`+strconv.FormatInt(warehouseID, 10)+`,"goods_id":`+
			strconv.FormatInt(goodsID, 10)+`,"direction":"out","qty":10}`, admin)
	var adj struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &adj)
	id := strconv.FormatInt(adj.ID, 10)
	if w := call(t, r, http.MethodPost, "/api/v1/stock-adjustments/"+id+"/submit", "", admin); w.Code != http.StatusOK {
		t.Fatalf("提交: %d %s", w.Code, w.Body.String())
	}
	if w := call(t, r, http.MethodPost, "/api/v1/audits/stock-adjustment/"+id, `{"pass":true}`, approver); w.Code != http.StatusConflict {
		t.Fatalf("超库存出库调整返回 %d，期望 409", w.Code)
	}
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 5 {
		t.Fatalf("库存不应变化，got %v", got)
	}
}

// TestStockCheckRejectsDuplicateGoods 防止同一商品多行导致重复调整。
func TestStockCheckRejectsDuplicateGoods(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	_, warehouseID, goodsID, _ := setupMaster(t, r, admin)
	body := `{"warehouse_id":` + strconv.FormatInt(warehouseID, 10) +
		`,"items":[{"goods_id":` + strconv.FormatInt(goodsID, 10) + `,"actual_qty":8},` +
		`{"goods_id":` + strconv.FormatInt(goodsID, 10) + `,"actual_qty":7}]}`
	if w := call(t, r, http.MethodPost, "/api/v1/stock-checks", body, admin); w.Code != http.StatusBadRequest {
		t.Fatalf("重复商品盘点返回 %d，期望 400", w.Code)
	}
}

// TestMasterUpdatesPersistNewFields 覆盖此前被静默丢弃的字段。
func TestMasterUpdatesPersistNewFields(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	_, _, _, _ = setupMaster(t, r, admin)

	w := call(t, r, http.MethodPost, "/api/v1/goods",
		`{"code":"GFIX","name":"修复测试","spec":"旧规格","unit":"个","price":10,"cost":5,"purchase_price":5,"sale_price":10,"min_qty":3}`, admin)
	var g struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &g)
	id := strconv.FormatInt(g.ID, 10)
	if w := call(t, r, http.MethodPut, "/api/v1/goods/"+id,
		`{"code":"GFIX","name":"修复测试","spec":"新规格","unit":"个","price":10,"cost":5,"purchase_price":6,"sale_price":12,"min_qty":4}`, admin); w.Code != http.StatusOK {
		t.Fatalf("更新商品: %d %s", w.Code, w.Body.String())
	}
	w = call(t, r, http.MethodGet, "/api/v1/goods?page=1&page_size=50", "", admin)
	var list struct {
		List []struct {
			ID            int64   `json:"id"`
			Spec          string  `json:"spec"`
			PurchasePrice float64 `json:"purchase_price"`
			SalePrice     float64 `json:"sale_price"`
			MinQty        float64 `json:"min_qty"`
		} `json:"list"`
	}
	decodeData(t, w.Body.Bytes(), &list)
	updated := false
	for _, item := range list.List {
		if item.ID == g.ID {
			updated = true
			if item.Spec != "新规格" || item.PurchasePrice != 6 || item.SalePrice != 12 || item.MinQty != 4 {
				t.Fatalf("商品字段未持久化: %+v", item)
			}
		}
	}
	if !updated {
		t.Fatal("商品列表找不到更新后的记录")
	}

	w = call(t, r, http.MethodPost, "/api/v1/suppliers",
		`{"code":"SFIX","name":"供应商","contact":"李","phone":"1"}`, admin)
	var s struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &s)
	sid := strconv.FormatInt(s.ID, 10)
	call(t, r, http.MethodPut, "/api/v1/suppliers/"+sid,
		`{"code":"SFIX","name":"供应商","contact":"李","phone":"1","address":"新地址","payment_method":"月结"}`, admin)
	w = call(t, r, http.MethodGet, "/api/v1/suppliers?page=1&page_size=50", "", admin)
	var sList struct {
		List []struct {
			ID            int64  `json:"id"`
			Address       string `json:"address"`
			PaymentMethod string `json:"payment_method"`
		} `json:"list"`
	}
	decodeData(t, w.Body.Bytes(), &sList)
	found := false
	for _, item := range sList.List {
		if item.ID == s.ID {
			found = true
			if item.Address != "新地址" || item.PaymentMethod != "月结" {
				t.Fatalf("供应商字段未持久化: %+v", item)
			}
		}
	}
	if !found {
		t.Fatal("供应商列表找不到更新后的记录")
	}
}

// TestUnitDeleteActuallyDeletes 覆盖删除单位此前是空操作。
func TestUnitDeleteActuallyDeletes(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	w := call(t, r, http.MethodPost, "/api/v1/units", `{"name":"临时单位"}`, admin)
	var u struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &u)
	id := strconv.FormatInt(u.ID, 10)
	if w := call(t, r, http.MethodDelete, "/api/v1/units/"+id, "", admin); w.Code != http.StatusOK {
		t.Fatalf("删除单位: %d %s", w.Code, w.Body.String())
	}
	w = call(t, r, http.MethodGet, "/api/v1/units", "", admin)
	var list struct {
		List []struct {
			ID int64 `json:"id"`
		} `json:"list"`
	}
	decodeData(t, w.Body.Bytes(), &list)
	for _, item := range list.List {
		if item.ID == u.ID {
			t.Fatal("单位删除后仍存在")
		}
	}
}

// TestRejectRequiresComment 驳回必须填写原因。
func TestRejectRequiresComment(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	supplierID, warehouseID, goodsID, _ := setupMaster(t, r, admin)
	approver := approveUser(t, r, admin)
	body := `{"supplier_id":` + strconv.FormatInt(supplierID, 10) + `,"warehouse_id":` +
		strconv.FormatInt(warehouseID, 10) + `,"items":[{"goods_id":` +
		strconv.FormatInt(goodsID, 10) + `,"qty":1,"price":5}]}`
	w := call(t, r, http.MethodPost, "/api/v1/purchases", body, admin)
	var po struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &po)
	id := strconv.FormatInt(po.ID, 10)
	call(t, r, http.MethodPost, "/api/v1/purchases/"+id+"/submit", "", admin)
	if w := call(t, r, http.MethodPost, "/api/v1/audits/purchase/"+id, `{"pass":false}`, approver); w.Code != http.StatusBadRequest {
		t.Fatalf("空原因驳回返回 %d，期望 400", w.Code)
	}
}

// TestInventoryKeywordFilters 盘点/调拨关键字过滤不能查不存在的 order_no。
func TestInventoryKeywordFilters(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	for _, path := range []string{"/api/v1/stock-checks?keyword=x", "/api/v1/transfers?keyword=x"} {
		if w := call(t, r, http.MethodGet, path, "", admin); w.Code != http.StatusOK {
			t.Errorf("%s 返回 %d: %s", path, w.Code, w.Body.String())
		}
	}
}
