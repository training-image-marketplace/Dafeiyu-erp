// 有问题欢迎联系 hot_kun@hotmail.com
package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// setupMaster 建一套最小主数据，返回 id。
func approvedStockAdjustment(t *testing.T, r *gin.Engine, creator, approver sessionInfo, warehouseID, goodsID int64, direction string, qty float64) int64 {
	t.Helper()
	body := `{"warehouse_id":` + strconv.FormatInt(warehouseID, 10) + `,"goods_id":` +
		strconv.FormatInt(goodsID, 10) + `,"direction":"` + direction + `","qty":` +
		strconv.FormatFloat(qty, 'f', -1, 64) + `,"reason":"test setup"}`
	w := call(t, r, http.MethodPost, "/api/v1/stock-adjustments", body, creator)
	if w.Code != http.StatusOK {
		t.Fatalf("create stock adjustment: %d %s", w.Code, w.Body.String())
	}
	var item struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &item)
	id := strconv.FormatInt(item.ID, 10)
	if w := call(t, r, http.MethodPost, "/api/v1/stock-adjustments/"+id+"/submit", "", creator); w.Code != http.StatusOK {
		t.Fatalf("submit stock adjustment: %d %s", w.Code, w.Body.String())
	}
	if w := call(t, r, http.MethodPost, "/api/v1/audits/stock-adjustment/"+id, `{"pass":true}`, approver); w.Code != http.StatusOK {
		t.Fatalf("approve stock adjustment: %d %s", w.Code, w.Body.String())
	}
	return item.ID
}

func setupMaster(t *testing.T, r *gin.Engine, admin sessionInfo) (supplierID, warehouseID, goodsID, customerID int64) {
	t.Helper()

	post := func(path, body string, dst any) {
		t.Helper()
		w := call(t, r, http.MethodPost, path, body, admin)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		if dst != nil {
			decodeData(t, w.Body.Bytes(), dst)
		}
	}

	var s struct {
		ID int64 `json:"id"`
	}
	post("/api/v1/suppliers", `{"code":"SUP1","name":"供应商A"}`, &s)
	var wh struct {
		ID int64 `json:"id"`
	}
	post("/api/v1/warehouses", `{"code":"WH1","name":"主仓","location":"一楼"}`, &wh)
	var g struct {
		ID int64 `json:"id"`
	}
	post("/api/v1/goods", `{"code":"G1","name":"螺丝","unit":"个","price":5,"cost":3}`, &g)
	var c struct {
		ID int64 `json:"id"`
	}
	post("/api/v1/customers", `{"code":"CUS1","name":"客户A"}`, &c)
	return s.ID, wh.ID, g.ID, c.ID
}

// TestStockAdjustmentRequiresApprovalAndUsesExplicitDirection 覆盖库存调整的审核闭环：
// 入库/出库必须明确选择且数量为正，审核通过前不改库存，审核后才入账。
func TestStockAdjustmentRequiresApprovalAndUsesExplicitDirection(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	_, warehouseID, goodsID, _ := setupMaster(t, r, admin)

	// 建立一名独立审核人，避免自己审自己的单据。
	approverID := createUser(t, r, admin, "stock-auditor", "stock-auditor-pw")
	if w := call(t, r, http.MethodPut, "/api/v1/users/"+strconv.FormatInt(approverID, 10)+"/roles",
		`{"role_ids":[1]}`, admin); w.Code != http.StatusOK {
		t.Fatalf("给审核人授权: %d %s", w.Code, w.Body.String())
	}
	approver := login(t, r, "stock-auditor", "stock-auditor-pw")

	// 方向独立于数量：入库以正数量创建草稿，创建和提交均不得立即改库存。
	w := call(t, r, http.MethodPost, "/api/v1/stock-adjustments",
		`{"warehouse_id":`+strconv.FormatInt(warehouseID, 10)+`,"goods_id":`+
			strconv.FormatInt(goodsID, 10)+`,"direction":"in","qty":12,"reason":"盘盈"}`, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("创建入库调整: %d %s", w.Code, w.Body.String())
	}
	var adjustment struct {
		ID        int64   `json:"id"`
		Status    string  `json:"status"`
		Direction string  `json:"direction"`
		Qty       float64 `json:"qty"`
	}
	decodeData(t, w.Body.Bytes(), &adjustment)
	if adjustment.ID == 0 || adjustment.Status != "draft" || adjustment.Direction != "in" || adjustment.Qty != 12 {
		t.Fatalf("调整单返回错误: %+v", adjustment)
	}
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 0 {
		t.Fatalf("草稿不应改库存，got %v", got)
	}

	id := strconv.FormatInt(adjustment.ID, 10)
	if w := call(t, r, http.MethodPost, "/api/v1/stock-adjustments/"+id+"/submit", "", admin); w.Code != http.StatusOK {
		t.Fatalf("提交入库调整: %d %s", w.Code, w.Body.String())
	}
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 0 {
		t.Fatalf("待审核调整不应改库存，got %v", got)
	}

	w = call(t, r, http.MethodGet, "/api/v1/audits?direction=in&status=submitted", "", approver)
	if w.Code != http.StatusOK {
		t.Fatalf("查看入库审核: %d %s", w.Code, w.Body.String())
	}
	var pending struct {
		List []struct {
			BillType string `json:"bill_type"`
			BillID   int64  `json:"bill_id"`
		} `json:"list"`
	}
	decodeData(t, w.Body.Bytes(), &pending)
	found := false
	for _, bill := range pending.List {
		if bill.BillType == "stock-adjustment" && bill.BillID == adjustment.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("审核中心未列出库存调整单: %+v", pending.List)
	}

	w = call(t, r, http.MethodPost, "/api/v1/audits/stock-adjustment/"+id,
		`{"pass":true}`, approver)
	if w.Code != http.StatusOK {
		t.Fatalf("审核入库调整: %d %s", w.Code, w.Body.String())
	}
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 12 {
		t.Fatalf("审核通过后应入库 12，got %v", got)
	}

	w = call(t, r, http.MethodPost, "/api/v1/stock-adjustments",
		`{"warehouse_id":`+strconv.FormatInt(warehouseID, 10)+`,"goods_id":`+
			strconv.FormatInt(goodsID, 10)+`,"direction":"out","qty":5,"reason":"盘亏"}`, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("创建出库调整: %d %s", w.Code, w.Body.String())
	}
	var outbound struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &outbound)
	outID := strconv.FormatInt(outbound.ID, 10)
	if w := call(t, r, http.MethodPost, "/api/v1/stock-adjustments/"+outID+"/submit", "", admin); w.Code != http.StatusOK {
		t.Fatalf("提交出库调整: %d %s", w.Code, w.Body.String())
	}
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 12 {
		t.Fatalf("待审出库不得提前减库存，got %v", got)
	}
	if w := call(t, r, http.MethodPost, "/api/v1/audits/stock-adjustment/"+outID,
		`{"pass":false,"comment":"数量需复核"}`, approver); w.Code != http.StatusOK {
		t.Fatalf("驳回出库调整: %d %s", w.Code, w.Body.String())
	}
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 12 {
		t.Fatalf("驳回后库存不应变化，got %v", got)
	}

	// 库存扣减调整同样要明确选择 out、输入正数量；不能用负数方向偷偷反转。
	w = call(t, r, http.MethodPost, "/api/v1/stock-adjustments",
		`{"warehouse_id":`+strconv.FormatInt(warehouseID, 10)+`,"goods_id":`+
			strconv.FormatInt(goodsID, 10)+`,"direction":"out","qty":-2,"reason":"非法负数"}`, admin)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("负数量出库应拒绝，got %d %s", w.Code, w.Body.String())
	}

	// 入库数量不能因自审例外开关之外的任何原因由提交人本人审核。
	w = call(t, r, http.MethodPost, "/api/v1/stock-adjustments",
		`{"warehouse_id":`+strconv.FormatInt(warehouseID, 10)+`,"goods_id":`+
			strconv.FormatInt(goodsID, 10)+`,"direction":"in","qty":1}`, admin)
	decodeData(t, w.Body.Bytes(), &outbound)
	selfID := strconv.FormatInt(outbound.ID, 10)
	call(t, r, http.MethodPost, "/api/v1/stock-adjustments/"+selfID+"/submit", "", admin)
	if w := call(t, r, http.MethodPost, "/api/v1/audits/stock-adjustment/"+selfID,
		`{"pass":true}`, admin); w.Code != http.StatusForbidden {
		t.Fatalf("提交人自审应拒绝，got %d %s", w.Code, w.Body.String())
	}
}

// TestERPPurchaseSalesLifecycle 覆盖采购入库与销售出库的库存流转。
func TestERPPurchaseSalesLifecycle(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	supplierID, warehouseID, goodsID, customerID := setupMaster(t, r, admin)

	// 审批人：避免「自己审自己」被默认规则拦住
	approverID := createUser(t, r, admin, "approver", "approver-pw")
	if w := call(t, r, http.MethodPut, "/api/v1/users/"+strconv.FormatInt(approverID, 10)+"/roles",
		`{"role_ids":[1]}`, admin); w.Code != http.StatusOK {
		t.Fatalf("给审批人授权: %d %s", w.Code, w.Body.String())
	}
	approver := login(t, r, "approver", "approver-pw")

	// 初始库存 100，通过审核的盘点入库调整建立。
	approvedStockAdjustment(t, r, admin, approver, warehouseID, goodsID, "in", 100)

	// 采购：10 × 5 = 50
	body := `{"supplier_id":` + strconv.FormatInt(supplierID, 10) + `,"warehouse_id":` +
		strconv.FormatInt(warehouseID, 10) + `,"items":[{"goods_id":` +
		strconv.FormatInt(goodsID, 10) + `,"qty":10,"price":5}]}`

	w := call(t, r, http.MethodPost, "/api/v1/purchases", body, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("建采购单: %d %s", w.Code, w.Body.String())
	}
	var po struct {
		ID          int64   `json:"id"`
		OrderNo     string  `json:"order_no"`
		Status      string  `json:"status"`
		TotalAmount float64 `json:"total_amount"`
	}
	decodeData(t, w.Body.Bytes(), &po)
	if po.Status != "draft" || po.TotalAmount != 50 || po.OrderNo == "" {
		t.Fatalf("采购单初始状态不符: %+v", po)
	}
	poID := strconv.FormatInt(po.ID, 10)

	if w := call(t, r, http.MethodPost, "/api/v1/purchases/"+poID+"/submit", "", admin); w.Code != http.StatusOK {
		t.Fatalf("提交采购: %d %s", w.Code, w.Body.String())
	}
	if w := call(t, r, http.MethodPost, "/api/v1/purchases/"+poID+"/approve", "", admin); w.Code != http.StatusForbidden {
		t.Errorf("自己审批自己的单据返回 %d，期望 403", w.Code)
	}
	if w := call(t, r, http.MethodPost, "/api/v1/purchases/"+poID+"/approve", "", approver); w.Code != http.StatusOK {
		t.Fatalf("审批采购: %d %s", w.Code, w.Body.String())
	}
	if w := call(t, r, http.MethodPost, "/api/v1/purchases/"+poID+"/receive", "", admin); w.Code != http.StatusOK {
		t.Fatalf("入库: %d %s", w.Code, w.Body.String())
	}

	// 入库后 110
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 110 {
		t.Errorf("入库后库存 = %v，期望 110", got)
	}

	// 销售：3 × 8 = 24
	salesBody := `{"customer_id":` + strconv.FormatInt(customerID, 10) + `,"warehouse_id":` +
		strconv.FormatInt(warehouseID, 10) + `,"items":[{"goods_id":` +
		strconv.FormatInt(goodsID, 10) + `,"qty":3,"price":8}]}`
	w = call(t, r, http.MethodPost, "/api/v1/sales", salesBody, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("建销售单: %d %s", w.Code, w.Body.String())
	}
	var so struct {
		ID      int64   `json:"id"`
		OrderNo string  `json:"order_no"`
		Total   float64 `json:"total_amount"`
	}
	decodeData(t, w.Body.Bytes(), &so)
	if so.Total != 24 {
		t.Errorf("销售单总额 = %v，期望 24", so.Total)
	}
	soID := strconv.FormatInt(so.ID, 10)
	call(t, r, http.MethodPost, "/api/v1/sales/"+soID+"/submit", "", admin)
	if w := call(t, r, http.MethodPost, "/api/v1/sales/"+soID+"/approve", "", approver); w.Code != http.StatusOK {
		t.Fatalf("审批销售: %d %s", w.Code, w.Body.String())
	}
	if w := call(t, r, http.MethodPost, "/api/v1/sales/"+soID+"/ship", "", admin); w.Code != http.StatusOK {
		t.Fatalf("出库: %d %s", w.Code, w.Body.String())
	}
	if got := stockQty(t, r, admin, warehouseID, goodsID); got != 107 {
		t.Errorf("出库后库存 = %v，期望 107", got)
	}

	// 操作记录：创建/提交/审批/入库
	w = call(t, r, http.MethodGet, "/api/v1/bills/purchase/"+poID+"/operations", "", admin)
	var ops struct {
		List []struct {
			Action string `json:"action"`
		} `json:"list"`
	}
	decodeData(t, w.Body.Bytes(), &ops)
	if len(ops.List) != 4 {
		t.Errorf("采购单操作记录 = %d 条，期望 4（create/submit/approve/receive）", len(ops.List))
	}
}

// TestERPBugSwitches 验证三个缺陷开关真的改变行为。
func TestERPBugSwitches(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	supplierID, warehouseID, goodsID, customerID := setupMaster(t, r, admin)

	setBug := func(code string, on bool) {
		t.Helper()
		body := `{"bug_code":"` + code + `","enabled":` + strconv.FormatBool(on) + `}`
		if w := call(t, r, http.MethodPost, "/api/v1/switches/bug", body, admin); w.Code != http.StatusOK {
			t.Fatalf("设置 %s: %d %s", code, w.Code, w.Body.String())
		}
	}

	poBody := `{"supplier_id":` + strconv.FormatInt(supplierID, 10) + `,"warehouse_id":` +
		strconv.FormatInt(warehouseID, 10) + `,"items":[{"goods_id":` +
		strconv.FormatInt(goodsID, 10) + `,"qty":10,"price":5}]}`

	// BUG-001：总额少 1 分
	setBug("BUG-001", true)
	w := call(t, r, http.MethodPost, "/api/v1/purchases", poBody, admin)
	var po struct {
		ID          int64   `json:"id"`
		TotalAmount float64 `json:"total_amount"`
	}
	decodeData(t, w.Body.Bytes(), &po)
	if po.TotalAmount != 49.99 {
		t.Errorf("开启 BUG-001 后总额 = %v，期望 49.99", po.TotalAmount)
	}
	setBug("BUG-001", false)

	// BUG-003：允许自己审批
	setBug("BUG-003", true)
	poID := strconv.FormatInt(po.ID, 10)
	call(t, r, http.MethodPost, "/api/v1/purchases/"+poID+"/submit", "", admin)
	if w := call(t, r, http.MethodPost, "/api/v1/purchases/"+poID+"/approve", "", admin); w.Code != http.StatusOK {
		t.Errorf("开启 BUG-003 后自审返回 %d，期望 200", w.Code)
	}
	setBug("BUG-003", false)

	// BUG-002：允许负库存；先用已审核入库调整准备库存。
	// 这段用例只有 admin 一个账号，临时打开 BUG-003 才能自审准备库存，随后立即关闭。
	setBug("BUG-003", true)
	approvedStockAdjustment(t, r, admin, admin, warehouseID, goodsID, "in", 5)
	setBug("BUG-003", false)
	soBody := `{"customer_id":` + strconv.FormatInt(customerID, 10) + `,"warehouse_id":` +
		strconv.FormatInt(warehouseID, 10) + `,"items":[{"goods_id":` +
		strconv.FormatInt(goodsID, 10) + `,"qty":100,"price":1}]}`
	w = call(t, r, http.MethodPost, "/api/v1/sales", soBody, admin)
	var so struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &so)
	soID := strconv.FormatInt(so.ID, 10)
	call(t, r, http.MethodPost, "/api/v1/sales/"+soID+"/submit", "", admin)
	setBug("BUG-003", true) // 自审放行，简化准备步骤
	call(t, r, http.MethodPost, "/api/v1/sales/"+soID+"/approve", "", admin)

	// 关闭 BUG-002 时必须拒绝
	if w := call(t, r, http.MethodPost, "/api/v1/sales/"+soID+"/ship", "", admin); w.Code != http.StatusConflict {
		t.Errorf("库存不足时出库返回 %d，期望 409", w.Code)
	}
	setBug("BUG-002", true)
	if w := call(t, r, http.MethodPost, "/api/v1/sales/"+soID+"/ship", "", admin); w.Code != http.StatusOK {
		t.Errorf("开启 BUG-002 后出库返回 %d，期望 200", w.Code)
	}
	if got := stockQty(t, r, admin, warehouseID, goodsID); got >= 0 {
		t.Errorf("开启 BUG-002 后库存 = %v，期望为负", got)
	}
}

// TestInternalEndpointsRequireToken 确认内部接口不会被匿名访问。
func TestInternalEndpointsRequireToken(t *testing.T) {
	r, _ := newTestRouter(t)
	if w := call(t, r, http.MethodGet, "/internal/health", "", sessionInfo{}); w.Code != http.StatusUnauthorized {
		t.Errorf("无令牌访问 /internal/health 返回 %d，期望 401", w.Code)
	}
}

func stockQty(t *testing.T, r *gin.Engine, s sessionInfo, warehouseID, goodsID int64) float64 {
	t.Helper()
	w := call(t, r, http.MethodGet, "/api/v1/stocks", "", s)
	if w.Code != http.StatusOK {
		t.Fatalf("查库存: %d %s", w.Code, w.Body.String())
	}
	var res struct {
		List []struct {
			WarehouseID int64   `json:"warehouse_id"`
			GoodsID     int64   `json:"goods_id"`
			Qty         float64 `json:"qty"`
		} `json:"list"`
	}
	decodeData(t, w.Body.Bytes(), &res)
	for _, row := range res.List {
		if row.WarehouseID == warehouseID && row.GoodsID == goodsID {
			return row.Qty
		}
	}
	return 0
}

func decodeData(t *testing.T, body []byte, dst any) {
	t.Helper()
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("解析响应: %v\n%s", err, body)
	}
	if err := json.Unmarshal(env.Data, dst); err != nil {
		t.Fatalf("解析 data: %v\n%s", err, env.Data)
	}
}

// TestMenuSwitchHidesModule 验证菜单开关能真正拦住接口，而不只是隐藏入口。
func TestMenuSwitchHidesModule(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")

	setMenus := func(goods bool) {
		t.Helper()
		body := `{"scope":"all","scope_id":0,"config":{"bugs":{},"menus":{"goods":` +
			strconv.FormatBool(goods) + `}}}`
		req := callWithHeader(t, r, http.MethodPut, "/internal/switches", body, sessionInfo{},
			"X-Internal-Token", "test-internal-token")
		if req.Code != http.StatusOK {
			t.Fatalf("设置菜单开关: %d %s", req.Code, req.Body.String())
		}
	}

	setMenus(false)
	if w := call(t, r, http.MethodGet, "/api/v1/goods", "", admin); w.Code != http.StatusForbidden {
		t.Errorf("菜单关闭后访问商品接口返回 %d，期望 403", w.Code)
	}
	if w := call(t, r, http.MethodPost, "/api/v1/goods",
		`{"code":"GW1","name":"写接口","unit":"个"}`, admin); w.Code != http.StatusForbidden {
		t.Errorf("菜单关闭后写接口返回 %d，期望 403", w.Code)
	}
	setMenus(true)
	if w := call(t, r, http.MethodGet, "/api/v1/goods", "", admin); w.Code != http.StatusOK {
		t.Errorf("菜单开启后访问商品接口返回 %d，期望 200", w.Code)
	}
}

func callWithHeader(t *testing.T, r *gin.Engine, method, path, body string, s sessionInfo, key, val string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(key, val)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestDraftPurchaseOrderEdit 覆盖草稿改单：金额重算、明细替换、提交后拒绝。
func TestDraftPurchaseOrderEdit(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	supplierID, warehouseID, goodsID, _ := setupMaster(t, r, admin)

	body := `{"supplier_id":` + strconv.FormatInt(supplierID, 10) + `,"warehouse_id":` +
		strconv.FormatInt(warehouseID, 10) + `,"items":[{"goods_id":` +
		strconv.FormatInt(goodsID, 10) + `,"qty":10,"price":5}]}`
	w := call(t, r, http.MethodPost, "/api/v1/purchases", body, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("建单: %d %s", w.Code, w.Body.String())
	}
	var po struct {
		ID          int64   `json:"id"`
		TotalAmount float64 `json:"total_amount"`
	}
	decodeData(t, w.Body.Bytes(), &po)
	if po.TotalAmount != 50 {
		t.Fatalf("建单金额 = %v，期望 50", po.TotalAmount)
	}

	edit := `{"supplier_id":` + strconv.FormatInt(supplierID, 10) + `,"warehouse_id":` +
		strconv.FormatInt(warehouseID, 10) + `,"items":[{"goods_id":` +
		strconv.FormatInt(goodsID, 10) + `,"qty":12,"price":5}]}`
	w = call(t, r, http.MethodPut, "/api/v1/purchases/"+strconv.FormatInt(po.ID, 10), edit, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("改单: %d %s", w.Code, w.Body.String())
	}

	w = call(t, r, http.MethodGet, "/api/v1/purchases/"+strconv.FormatInt(po.ID, 10), "", admin)
	var got struct {
		TotalAmount float64 `json:"total_amount"`
		Items       []struct {
			Qty float64 `json:"qty"`
		} `json:"items"`
	}
	decodeData(t, w.Body.Bytes(), &got)
	if got.TotalAmount != 60 || len(got.Items) != 1 || got.Items[0].Qty != 12 {
		t.Errorf("改单后不符: %+v", got)
	}

	// 提交后不允许再改
	call(t, r, http.MethodPost, "/api/v1/purchases/"+strconv.FormatInt(po.ID, 10)+"/submit", "", admin)
	if w := call(t, r, http.MethodPut, "/api/v1/purchases/"+strconv.FormatInt(po.ID, 10), edit, admin); w.Code != http.StatusConflict {
		t.Errorf("已提交后改单返回 %d，期望 409", w.Code)
	}
}

// TestCategoryBrandCRUD 覆盖分类/品牌的改删与被商品引用时的删除保护。
func TestCategoryBrandCRUD(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")

	w := call(t, r, http.MethodPost, "/api/v1/categories", `{"name":"五金","sort":1}`, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("建分类: %d %s", w.Code, w.Body.String())
	}
	var cat struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &cat)

	w = call(t, r, http.MethodPost, "/api/v1/brands", `{"name":"品牌A"}`, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("建品牌: %d %s", w.Code, w.Body.String())
	}
	var brand struct {
		ID int64 `json:"id"`
	}
	decodeData(t, w.Body.Bytes(), &brand)

	// 改名
	if w := call(t, r, http.MethodPut, "/api/v1/categories/"+strconv.FormatInt(cat.ID, 10),
		`{"name":"五金件","sort":2}`, admin); w.Code != http.StatusOK {
		t.Fatalf("改分类: %d %s", w.Code, w.Body.String())
	}
	if w := call(t, r, http.MethodPut, "/api/v1/brands/"+strconv.FormatInt(brand.ID, 10),
		`{"name":"品牌B"}`, admin); w.Code != http.StatusOK {
		t.Fatalf("改品牌: %d %s", w.Code, w.Body.String())
	}

	// 商品引用后不能删
	w = call(t, r, http.MethodPost, "/api/v1/goods",
		`{"code":"G9","name":"螺丝刀","category_id":`+strconv.FormatInt(cat.ID, 10)+
			`,"brand_id":`+strconv.FormatInt(brand.ID, 10)+`,"price":10,"cost":5}`, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("建商品: %d %s", w.Code, w.Body.String())
	}
	if w := call(t, r, http.MethodDelete, "/api/v1/categories/"+strconv.FormatInt(cat.ID, 10), "", admin); w.Code != http.StatusConflict {
		t.Errorf("删被引用的分类返回 %d，期望 409", w.Code)
	}
	if w := call(t, r, http.MethodDelete, "/api/v1/brands/"+strconv.FormatInt(brand.ID, 10), "", admin); w.Code != http.StatusConflict {
		t.Errorf("删被引用的品牌返回 %d，期望 409", w.Code)
	}

	// 新建未被引用的可删
	w = call(t, r, http.MethodPost, "/api/v1/categories", `{"name":"临时分类","sort":9}`, admin)
	decodeData(t, w.Body.Bytes(), &cat)
	if w := call(t, r, http.MethodDelete, "/api/v1/categories/"+strconv.FormatInt(cat.ID, 10), "", admin); w.Code != http.StatusOK {
		t.Errorf("删空分类返回 %d，期望 200", w.Code)
	}
	w = call(t, r, http.MethodPost, "/api/v1/brands", `{"name":"临时品牌"}`, admin)
	decodeData(t, w.Body.Bytes(), &brand)
	if w := call(t, r, http.MethodDelete, "/api/v1/brands/"+strconv.FormatInt(brand.ID, 10), "", admin); w.Code != http.StatusOK {
		t.Errorf("删空品牌返回 %d，期望 200", w.Code)
	}
}
