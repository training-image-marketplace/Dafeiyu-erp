// 有问题欢迎联系 hot_kun@hotmail.com
package httpserver

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

// approveUser 建一个有内置角色的用户，用作「审核人」，避免默认禁止自审。
func approveUser(t *testing.T, r *gin.Engine, admin sessionInfo) sessionInfo {
	t.Helper()
	id := createUser(t, r, admin, "approver", "approver-pw")
	if w := call(t, r, http.MethodPut, "/api/v1/users/"+strconv.FormatInt(id, 10)+"/roles",
		`{"role_ids":[1]}`, admin); w.Code != http.StatusOK {
		t.Fatalf("给审核人授权: %d %s", w.Code, w.Body.String())
	}
	return login(t, r, "approver", "approver-pw")
}

// completedPurchase 建一张采购单并走到 completed，返回 id 与单号。
func completedPurchase(t *testing.T, r *gin.Engine, admin, approver sessionInfo,
	supplierID, warehouseID, goodsID int64, qty, price float64,
) (int64, string) {
	t.Helper()
	body := `{"supplier_id":` + strconv.FormatInt(supplierID, 10) + `,"warehouse_id":` +
		strconv.FormatInt(warehouseID, 10) + `,"items":[{"goods_id":` +
		strconv.FormatInt(goodsID, 10) + `,"qty":` + strconv.FormatFloat(qty, 'f', -1, 64) +
		`,"price":` + strconv.FormatFloat(price, 'f', -1, 64) + `}]}`
	w := call(t, r, http.MethodPost, "/api/v1/purchases", body, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("建采购单: %d %s", w.Code, w.Body.String())
	}
	var po struct {
		ID      int64  `json:"id"`
		OrderNo string `json:"order_no"`
	}
	decodeData(t, w.Body.Bytes(), &po)
	id := strconv.FormatInt(po.ID, 10)
	if w := call(t, r, http.MethodPost, "/api/v1/purchases/"+id+"/submit", "", admin); w.Code != http.StatusOK {
		t.Fatalf("提交采购: %d %s", w.Code, w.Body.String())
	}
	if w := call(t, r, http.MethodPost, "/api/v1/purchases/"+id+"/approve", "", approver); w.Code != http.StatusOK {
		t.Fatalf("审批采购: %d %s", w.Code, w.Body.String())
	}
	if w := call(t, r, http.MethodPost, "/api/v1/purchases/"+id+"/receive", "", admin); w.Code != http.StatusOK {
		t.Fatalf("入库: %d %s", w.Code, w.Body.String())
	}
	return po.ID, po.OrderNo
}

// completedSales 建一张销售单并走到 completed，返回 id 与单号。
func completedSales(t *testing.T, r *gin.Engine, admin, approver sessionInfo,
	customerID, warehouseID, goodsID int64, qty, price float64,
) (int64, string) {
	t.Helper()
	body := `{"customer_id":` + strconv.FormatInt(customerID, 10) + `,"warehouse_id":` +
		strconv.FormatInt(warehouseID, 10) + `,"items":[{"goods_id":` +
		strconv.FormatInt(goodsID, 10) + `,"qty":` + strconv.FormatFloat(qty, 'f', -1, 64) +
		`,"price":` + strconv.FormatFloat(price, 'f', -1, 64) + `}]}`
	w := call(t, r, http.MethodPost, "/api/v1/sales", body, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("建销售单: %d %s", w.Code, w.Body.String())
	}
	var so struct {
		ID      int64  `json:"id"`
		OrderNo string `json:"order_no"`
	}
	decodeData(t, w.Body.Bytes(), &so)
	id := strconv.FormatInt(so.ID, 10)
	if w := call(t, r, http.MethodPost, "/api/v1/sales/"+id+"/submit", "", admin); w.Code != http.StatusOK {
		t.Fatalf("提交销售: %d %s", w.Code, w.Body.String())
	}
	if w := call(t, r, http.MethodPost, "/api/v1/sales/"+id+"/approve", "", approver); w.Code != http.StatusOK {
		t.Fatalf("审批销售: %d %s", w.Code, w.Body.String())
	}
	if w := call(t, r, http.MethodPost, "/api/v1/sales/"+id+"/ship", "", admin); w.Code != http.StatusOK {
		t.Fatalf("出库: %d %s", w.Code, w.Body.String())
	}
	return so.ID, so.OrderNo
}

func returnBody(t *testing.T, sourceOrderID, goodsID int64, qty float64) string {
	t.Helper()
	return `{"source_order_id":` + strconv.FormatInt(sourceOrderID, 10) + `,"items":[{"goods_id":` +
		strconv.FormatInt(goodsID, 10) + `,"qty":` + strconv.FormatFloat(qty, 'f', -1, 64) + `}]}`
}

// TestOrderRejectWithdrawDelete 覆盖采购/销售单新增的三个动作与非法状态。
func TestOrderRejectWithdrawDelete(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	supplierID, warehouseID, goodsID, customerID := setupMaster(t, r, admin)
	approver := approveUser(t, r, admin)

	poBody := `{"supplier_id":` + strconv.FormatInt(supplierID, 10) + `,"warehouse_id":` +
		strconv.FormatInt(warehouseID, 10) + `,"items":[{"goods_id":` +
		strconv.FormatInt(goodsID, 10) + `,"qty":10,"price":5}]}`

	// 1) 驳回：submitted → rejected，原因进操作记录
	w := call(t, r, http.MethodPost, "/api/v1/purchases", poBody, admin)
	var po struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &po)
	poID := strconv.FormatInt(po.ID, 10)
	call(t, r, http.MethodPost, "/api/v1/purchases/"+poID+"/submit", "", admin)

	if w := call(t, r, http.MethodPost, "/api/v1/purchases/"+poID+"/reject",
		`{"comment":"价格不对，重新谈"}`, approver); w.Code != http.StatusOK {
		t.Fatalf("驳回采购: %d %s", w.Code, w.Body.String())
	}
	w = call(t, r, http.MethodGet, "/api/v1/purchases/"+poID, "", admin)
	var got struct {
		Status string `json:"status"`
	}
	decodeData(t, w.Body.Bytes(), &got)
	if got.Status != "rejected" {
		t.Errorf("驳回后状态 = %q，期望 rejected", got.Status)
	}

	w = call(t, r, http.MethodGet, "/api/v1/bills/purchase/"+poID+"/operations", "", admin)
	var ops struct {
		List []struct {
			Action  string `json:"action"`
			Comment string `json:"comment"`
		} `json:"list"`
	}
	decodeData(t, w.Body.Bytes(), &ops)
	last := ops.List[len(ops.List)-1]
	if last.Action != "reject" || last.Comment != "价格不对，重新谈" {
		t.Errorf("最后一条操作 = %+v，期望 reject + 驳回原因", last)
	}
	if w := call(t, r, http.MethodPost, "/api/v1/purchases/"+poID+"/reject",
		`{"comment":"重复驳回"}`, approver); w.Code != http.StatusConflict {
		t.Errorf("重复驳回返回 %d，期望 409", w.Code)
	}

	// 2) 撤销提交：submitted → draft，清空 submitted_at，之后可编辑
	w = call(t, r, http.MethodPost, "/api/v1/purchases", poBody, admin)
	decodeData(t, w.Body.Bytes(), &po)
	poID = strconv.FormatInt(po.ID, 10)
	call(t, r, http.MethodPost, "/api/v1/purchases/"+poID+"/submit", "", admin)
	if w := call(t, r, http.MethodPost, "/api/v1/purchases/"+poID+"/withdraw", "", admin); w.Code != http.StatusOK {
		t.Fatalf("撤销提交: %d %s", w.Code, w.Body.String())
	}
	w = call(t, r, http.MethodGet, "/api/v1/purchases/"+poID, "", admin)
	var detail struct {
		Status      string  `json:"status"`
		SubmittedAt *string `json:"submitted_at"`
	}
	decodeData(t, w.Body.Bytes(), &detail)
	if detail.Status != "draft" || detail.SubmittedAt != nil {
		t.Errorf("撤销后 = %+v，期望 draft 且 submitted_at 为空", detail)
	}
	if w := call(t, r, http.MethodPut, "/api/v1/purchases/"+poID, poBody, admin); w.Code != http.StatusOK {
		t.Errorf("撤销后编辑返回 %d，期望 200", w.Code)
	}

	// 3) 删除草稿；已提交的单不能删
	if w := call(t, r, http.MethodDelete, "/api/v1/purchases/"+poID, "", admin); w.Code != http.StatusOK {
		t.Fatalf("删除草稿: %d %s", w.Code, w.Body.String())
	}
	if w := call(t, r, http.MethodGet, "/api/v1/purchases/"+poID, "", admin); w.Code != http.StatusNotFound {
		t.Errorf("删除后查询返回 %d，期望 404", w.Code)
	}
	if w := call(t, r, http.MethodDelete, "/api/v1/purchases/"+poID, "", admin); w.Code != http.StatusNotFound {
		t.Errorf("重复删除返回 %d，期望 404", w.Code)
	}

	w = call(t, r, http.MethodPost, "/api/v1/purchases", poBody, admin)
	decodeData(t, w.Body.Bytes(), &po)
	poID = strconv.FormatInt(po.ID, 10)
	call(t, r, http.MethodPost, "/api/v1/purchases/"+poID+"/submit", "", admin)
	if w := call(t, r, http.MethodDelete, "/api/v1/purchases/"+poID, "", admin); w.Code != http.StatusConflict {
		t.Errorf("删除已提交单返回 %d，期望 409", w.Code)
	}

	// 4) 销售单同样支持驳回/撤销/删除（抽一条最短路径验证接线）
	soBody := `{"customer_id":` + strconv.FormatInt(customerID, 10) + `,"warehouse_id":` +
		strconv.FormatInt(warehouseID, 10) + `,"items":[{"goods_id":` +
		strconv.FormatInt(goodsID, 10) + `,"qty":1,"price":8}]}`
	w = call(t, r, http.MethodPost, "/api/v1/sales", soBody, admin)
	var so struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &so)
	soID := strconv.FormatInt(so.ID, 10)
	call(t, r, http.MethodPost, "/api/v1/sales/"+soID+"/submit", "", admin)
	if w := call(t, r, http.MethodPost, "/api/v1/sales/"+soID+"/withdraw", "", admin); w.Code != http.StatusOK {
		t.Errorf("撤销销售: %d", w.Code)
	}
	if w := call(t, r, http.MethodDelete, "/api/v1/sales/"+soID, "", admin); w.Code != http.StatusOK {
		t.Errorf("删除销售草稿: %d", w.Code)
	}
}

// TestPurchaseReturnLifecycle 覆盖采购退货的数量校验、审核、库存回冲与审核中心。
func TestPurchaseReturnLifecycle(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	supplierID, warehouseID, goodsID, _ := setupMaster(t, r, admin)
	approver := approveUser(t, r, admin)

	// 入库 10 @5 → 库存 10
	sourceID, sourceNo := completedPurchase(t, r, admin, approver, supplierID, warehouseID, goodsID, 10, 5)
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 10 {
		t.Fatalf("入库后库存 = %v，期望 10", got)
	}

	// 退 4 件：单价取来源单 5，总额 20
	w := call(t, r, http.MethodPost, "/api/v1/purchase-returns", returnBody(t, sourceID, goodsID, 4), admin)
	if w.Code != http.StatusOK {
		t.Fatalf("建采购退货: %d %s", w.Code, w.Body.String())
	}
	var ret struct {
		ID            int64   `json:"id"`
		ReturnNo      string  `json:"return_no"`
		Status        string  `json:"status"`
		TotalAmount   float64 `json:"total_amount"`
		SourceOrderNo string  `json:"source_order_no"`
		SupplierName  string  `json:"supplier_name"`
		WarehouseName string  `json:"warehouse_name"`
		Items         []struct {
			Price float64 `json:"price"`
			Qty   float64 `json:"qty"`
		} `json:"items"`
	}
	decodeData(t, w.Body.Bytes(), &ret)
	if ret.Status != "draft" || ret.TotalAmount != 20 || ret.SourceOrderNo != sourceNo {
		t.Fatalf("退货单初始值不符: %+v", ret)
	}
	if len(ret.Items) != 1 || ret.Items[0].Price != 5 || ret.Items[0].Qty != 4 {
		t.Errorf("退货明细 = %+v，期望按来源单单价 5 带出", ret.Items)
	}
	retID := strconv.FormatInt(ret.ID, 10)

	// 提交 → 审核 → 完成，库存回冲 4
	if w := call(t, r, http.MethodPost, "/api/v1/purchase-returns/"+retID+"/submit", "", admin); w.Code != http.StatusOK {
		t.Fatalf("提交退货: %d %s", w.Code, w.Body.String())
	}
	if w := call(t, r, http.MethodPost, "/api/v1/purchase-returns/"+retID+"/approve", "", approver); w.Code != http.StatusOK {
		t.Fatalf("审核退货: %d %s", w.Code, w.Body.String())
	}
	if w := call(t, r, http.MethodPost, "/api/v1/purchase-returns/"+retID+"/complete", "", admin); w.Code != http.StatusOK {
		t.Fatalf("完成退货: %d %s", w.Code, w.Body.String())
	}
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 6 {
		t.Errorf("退货后库存 = %v，期望 6", got)
	}

	// 剩余可退 6：3 件 + 4 件的草稿都能建（草稿不占额度），但先后提交时第二张必须被拦
	qtys := []float64{3, 4}
	var rets []int64
	for i, qty := range qtys {
		w = call(t, r, http.MethodPost, "/api/v1/purchase-returns", returnBody(t, sourceID, goodsID, qty), admin)
		if w.Code != http.StatusOK {
			t.Fatalf("建第 %d 张退货: %d %s", i+1, w.Code, w.Body.String())
		}
		var one struct {
			ID int64 `json:"id"`
		}
		decodeData(t, w.Body.Bytes(), &one)
		rets = append(rets, one.ID)
	}
	first := strconv.FormatInt(rets[0], 10)
	second := strconv.FormatInt(rets[1], 10)
	if w := call(t, r, http.MethodPost, "/api/v1/purchase-returns/"+first+"/submit", "", admin); w.Code != http.StatusOK {
		t.Fatalf("提交第一张退货: %d %s", w.Code, w.Body.String())
	}
	w = call(t, r, http.MethodPost, "/api/v1/purchase-returns/"+second+"/submit", "", admin)
	if w.Code != http.StatusConflict {
		t.Fatalf("超退提交返回 %d，期望 409", w.Code)
	}
	if e := decode(t, w); e.Msg != "error.returnQtyExceeded" {
		t.Errorf("超退错误键 = %q，期望 error.returnQtyExceeded", e.Msg)
	}

	// 第一张已提交（占 3 件），剩余 3：建 1 件成功，建 4 件失败
	if w := call(t, r, http.MethodPost, "/api/v1/purchase-returns", returnBody(t, sourceID, goodsID, 1), admin); w.Code != http.StatusOK {
		t.Errorf("剩余额度内建单返回 %d，期望 200", w.Code)
	}
	if w := call(t, r, http.MethodPost, "/api/v1/purchase-returns", returnBody(t, sourceID, goodsID, 4), admin); w.Code != http.StatusConflict {
		t.Errorf("超出剩余额度建单返回 %d，期望 409", w.Code)
	}

	// 审核中心：出库方向的待办里应能看到第一张退货
	w = call(t, r, http.MethodGet, "/api/v1/audits?direction=out&status=submitted", "", admin)
	if w.Code != http.StatusOK {
		t.Fatalf("审核中心列表: %d %s", w.Code, w.Body.String())
	}
	var audit struct {
		List []struct {
			BillType string `json:"bill_type"`
			BillNo   string `json:"bill_no"`
		} `json:"list"`
		Total int64 `json:"total"`
	}
	decodeData(t, w.Body.Bytes(), &audit)
	if audit.Total != 1 || audit.List[0].BillType != "purchase-return" {
		t.Fatalf("审核中心出库待办 = %+v，期望 1 条 purchase-return", audit)
	}

	// 通过审核中心驳回：释放额度，且驳回原因落库
	auditID := strconv.FormatInt(rets[0], 10)
	if w := call(t, r, http.MethodPost, "/api/v1/audits/purchase-return/"+auditID,
		`{"pass":false,"comment":"货已拆封"}`, approver); w.Code != http.StatusOK {
		t.Fatalf("审核中心驳回: %d %s", w.Code, w.Body.String())
	}
	w = call(t, r, http.MethodGet, "/api/v1/purchase-returns/"+first, "", admin)
	var after struct {
		Status string `json:"status"`
	}
	decodeData(t, w.Body.Bytes(), &after)
	if after.Status != "rejected" {
		t.Errorf("驳回后状态 = %q，期望 rejected", after.Status)
	}
}

// TestSalesReturnLifecycle 覆盖销售退货入库与超退拦截。
func TestSalesReturnLifecycle(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	_, warehouseID, goodsID, customerID := setupMaster(t, r, admin)
	approver := approveUser(t, r, admin)

	// 先审核入库 20，再卖 10
	approvedStockAdjustment(t, r, admin, approver, warehouseID, goodsID, "in", 20)
	sourceID, sourceNo := completedSales(t, r, admin, approver, customerID, warehouseID, goodsID, 10, 8)
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 10 {
		t.Fatalf("出库后库存 = %v，期望 10", got)
	}

	// 退 4 件，总额 32
	w := call(t, r, http.MethodPost, "/api/v1/sales-returns", returnBody(t, sourceID, goodsID, 4), admin)
	if w.Code != http.StatusOK {
		t.Fatalf("建销售退货: %d %s", w.Code, w.Body.String())
	}
	var ret struct {
		ID          int64   `json:"id"`
		TotalAmount float64 `json:"total_amount"`
		SourceOrder string  `json:"source_order_no"`
	}
	decodeData(t, w.Body.Bytes(), &ret)
	if ret.TotalAmount != 32 || ret.SourceOrder != sourceNo {
		t.Fatalf("销售退货初始值不符: %+v", ret)
	}
	id := strconv.FormatInt(ret.ID, 10)
	call(t, r, http.MethodPost, "/api/v1/sales-returns/"+id+"/submit", "", admin)
	call(t, r, http.MethodPost, "/api/v1/sales-returns/"+id+"/approve", "", approver)
	if w := call(t, r, http.MethodPost, "/api/v1/sales-returns/"+id+"/complete", "", admin); w.Code != http.StatusOK {
		t.Fatalf("完成销售退货: %d %s", w.Code, w.Body.String())
	}
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 14 {
		t.Errorf("退货入库后库存 = %v，期望 14", got)
	}

	// 已退 4，剩余 6：退 7 必须 409
	if w := call(t, r, http.MethodPost, "/api/v1/sales-returns", returnBody(t, sourceID, goodsID, 7), admin); w.Code != http.StatusConflict {
		t.Errorf("超退建单返回 %d，期望 409", w.Code)
	}

	// 来源单未完成时不允许退货
	soBody := `{"customer_id":` + strconv.FormatInt(customerID, 10) + `,"warehouse_id":` +
		strconv.FormatInt(warehouseID, 10) + `,"items":[{"goods_id":` +
		strconv.FormatInt(goodsID, 10) + `,"qty":1,"price":8}]}`
	w = call(t, r, http.MethodPost, "/api/v1/sales", soBody, admin)
	var draft struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &draft)
	if w := call(t, r, http.MethodPost, "/api/v1/sales-returns", returnBody(t, draft.ID, goodsID, 1), admin); w.Code != http.StatusConflict {
		t.Errorf("对草稿销售单退货返回 %d，期望 409", w.Code)
	}
}

// TestAuditCenter 覆盖聚合列表、审核分发与权限隔离。
func TestAuditCenter(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	supplierID, warehouseID, goodsID, customerID := setupMaster(t, r, admin)
	approver := approveUser(t, r, admin)

	poBody := `{"supplier_id":` + strconv.FormatInt(supplierID, 10) + `,"warehouse_id":` +
		strconv.FormatInt(warehouseID, 10) + `,"items":[{"goods_id":` +
		strconv.FormatInt(goodsID, 10) + `,"qty":2,"price":5}]}`
	w := call(t, r, http.MethodPost, "/api/v1/purchases", poBody, admin)
	var po struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &po)
	poID := strconv.FormatInt(po.ID, 10)
	call(t, r, http.MethodPost, "/api/v1/purchases/"+poID+"/submit", "", admin)

	soBody := `{"customer_id":` + strconv.FormatInt(customerID, 10) + `,"warehouse_id":` +
		strconv.FormatInt(warehouseID, 10) + `,"items":[{"goods_id":` +
		strconv.FormatInt(goodsID, 10) + `,"qty":1,"price":8}]}`
	w = call(t, r, http.MethodPost, "/api/v1/sales", soBody, admin)
	var so struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &so)
	soID := strconv.FormatInt(so.ID, 10)
	call(t, r, http.MethodPost, "/api/v1/sales/"+soID+"/submit", "", admin)

	// 入库方向的待办只有采购单
	w = call(t, r, http.MethodGet, "/api/v1/audits?direction=in&status=submitted", "", admin)
	var inList struct {
		List []struct {
			BillType string  `json:"bill_type"`
			BillID   int64   `json:"bill_id"`
			Creator  string  `json:"creator_name"`
			Partner  string  `json:"partner_name"`
			Approver string  `json:"approver_name"`
			Total    float64 `json:"total_amount"`
		} `json:"list"`
		Total int64 `json:"total"`
	}
	decodeData(t, w.Body.Bytes(), &inList)
	if inList.Total != 1 || inList.List[0].BillType != "purchase" || inList.List[0].BillID != po.ID {
		t.Fatalf("入库待办 = %+v，期望 1 条 purchase", inList)
	}
	if inList.List[0].Creator == "" || inList.List[0].Partner == "" || inList.List[0].Total != 10 {
		t.Errorf("审核列表展示字段不完整: %+v", inList.List[0])
	}

	w = call(t, r, http.MethodGet, "/api/v1/audits?direction=out&status=submitted", "", admin)
	var outList struct {
		List []struct {
			BillType string `json:"bill_type"`
		} `json:"list"`
		Total int64 `json:"total"`
	}
	decodeData(t, w.Body.Bytes(), &outList)
	if outList.Total != 1 || outList.List[0].BillType != "sales" {
		t.Fatalf("出库待办 = %+v，期望 1 条 sales", outList)
	}

	// 审核中心驳回采购单
	if w := call(t, r, http.MethodPost, "/api/v1/audits/purchase/"+poID,
		`{"pass":false,"comment":"供应商资质过期"}`, approver); w.Code != http.StatusOK {
		t.Fatalf("审核中心驳回采购: %d %s", w.Code, w.Body.String())
	}
	w = call(t, r, http.MethodGet, "/api/v1/purchases/"+poID, "", admin)
	var got struct {
		Status string `json:"status"`
	}
	decodeData(t, w.Body.Bytes(), &got)
	if got.Status != "rejected" {
		t.Errorf("驳回后状态 = %q，期望 rejected", got.Status)
	}

	// 审核中心通过销售单
	if w := call(t, r, http.MethodPost, "/api/v1/audits/sales/"+soID, `{"pass":true}`, approver); w.Code != http.StatusOK {
		t.Fatalf("审核中心通过销售: %d %s", w.Code, w.Body.String())
	}

	// 非法参数
	if w := call(t, r, http.MethodPost, "/api/v1/audits/bogus/1", `{"pass":true}`, admin); w.Code != http.StatusBadRequest {
		t.Errorf("非法单据类型返回 %d，期望 400", w.Code)
	}
	if w := call(t, r, http.MethodPost, "/api/v1/audits/sales/"+soID, `{}`, admin); w.Code != http.StatusBadRequest {
		t.Errorf("缺少 pass 返回 %d，期望 400", w.Code)
	}

	// 无审核权限的普通用户看不到审核中心
	createUser(t, r, admin, "nobody", "nobody-pw")
	nobody := login(t, r, "nobody", "nobody-pw")
	if w := call(t, r, http.MethodGet, "/api/v1/audits", "", nobody); w.Code != http.StatusForbidden {
		t.Errorf("无权限访问审核中心返回 %d，期望 403", w.Code)
	}
}
