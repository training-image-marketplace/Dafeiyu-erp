// 有问题欢迎联系 hot_kun@hotmail.com
package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"practical-training/erp-system/internal/domain"
	"practical-training/erp-system/internal/httpx"
	"practical-training/erp-system/internal/middleware"
	"practical-training/erp-system/internal/repository"
	"practical-training/erp-system/internal/service"
)

// ERPHandler 暴露 ERP 主数据、库存、采购与销售接口。
type ERPHandler struct {
	svc      *service.ERPService
	switches *service.SwitchService
}

func NewERPHandler(svc *service.ERPService, switches *service.SwitchService) *ERPHandler {
	return &ERPHandler{svc: svc, switches: switches}
}

// scopeFrom 返回本次请求应用的开关作用域。
//
// 历史实现从 X-Switch-Scope / X-Switch-Scope-Id 请求头取作用域，但这条链路
// 从来没有真正的发送方：平台只把配置 PUT 给 /internal/switches，学生是浏览器
// 直连 ERP 的。于是两个后果同时成立：
//  1. 老师配在「小组 / 个人」上的缺陷永远不会被学生触发；
//  2. 任何客户端自己带一个 X-Switch-Scope: all 就能把老师开的缺陷摘掉。
//
// 现在一律按 all 解析：行为确定，不再受客户端影响。小组/个人由平台在
// 开环境/保存配置时合并成 all 下推进学生容器（见 docs/02 §12.1），
// ERP 运行时不需要、也不应该再相信客户端给的作用域。
//
// 参数保留而不用：A/B/C 任一接线方案都要重新用到请求上下文，
// 留着它可以让 20 处调用点不必跟着改签名。
func scopeFrom(_ *gin.Context) (string, int64) {
	return "all", 0
}

// menuEnabled 检查菜单开关；关闭时返回 403。
func (h *ERPHandler) menuEnabled(c *gin.Context, code string) bool {
	scope, id := scopeFrom(c)
	if h.switches.MenuEnabled(c.Request.Context(), code, scope, id) {
		return true
	}
	httpx.Fail(c, http.StatusForbidden, httpx.KeyForbidden)
	return false
}

// MenuSwitchGate 在认证组统一拦业务菜单开关。
//
// 逐个 handler 补 menuEnabled 容易漏（历史上只拦了列表），
// 这里按路径前缀集中判断，写接口与读接口口径一致。
func (h *ERPHandler) MenuSwitchGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		code := menuCodeOfPath(c.Request.URL.Path)
		if code == "" {
			c.Next()
			return
		}
		scope, scopeID := scopeFrom(c)
		if !h.switches.MenuEnabled(c.Request.Context(), code, scope, scopeID) {
			httpx.Fail(c, http.StatusForbidden, httpx.KeyForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}

// menuCodeByResource 把 API 资源段映射到菜单开关的 key。
//
// 这些 key 与平台开关页的勾选项、docs/03 §4.4 的 config_json 必须一致。
var menuCodeByResource = map[string]string{
	"customers":         "customers",
	"suppliers":         "suppliers",
	"warehouses":        "warehouses",
	"goods":             "goods",
	"categories":        "goods",
	"brands":            "goods",
	"units":             "goods",
	"stocks":            "stock",
	"stock-adjustments": "stock",
	"transfers":         "stock",
	"stock-checks":      "stock",
	"reports":           "reports",
	"purchases":         "purchase",
	"sales":             "sales",
	"purchase-returns":  "purchase_return",
	"sales-returns":     "sales_return",
	"audits":            "audit",
}

// menuCodeOfPath 按资源段**精确匹配**，不做前缀匹配。
//
// 前缀匹配配 map 迭代顺序会出随机 bug：/api/v1/sales-returns 同时以
// /api/v1/sales 开头，于是「销售退货」的判定可能落到「销售」这个 key 上——
// 关掉一个，另一个时好时坏地 403。
func menuCodeOfPath(path string) string {
	// 调用方传的是 URL.Path（不含查询串）；这里顺手剥掉 "?..." ，
	// 免得以后有人改传 RequestURI 时把带查询的路径判成「不属于任何菜单」。
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	rest, ok := strings.CutPrefix(path, "/api/v1/")
	if !ok {
		return ""
	}
	resource, _, _ := strings.Cut(rest, "/")
	return menuCodeByResource[resource]
}

func currentOperator(c *gin.Context) (int64, string, bool) {
	u, ok := middleware.CurrentUser(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, httpx.KeyUnauthorized)
		return 0, "", false
	}
	return u.ID, u.Nickname, true
}

// ---------- 客户 ----------

type customerRequest struct {
	Code          string `json:"code" binding:"required,min=2,max=32,alphanum"`
	Name          string `json:"name" binding:"required,min=1,max=128"`
	Contact       string `json:"contact" binding:"max=64"`
	Phone         string `json:"phone" binding:"max=32"`
	Address       string `json:"address" binding:"max=255"`
	PaymentMethod string `json:"payment_method" binding:"max=32"`
	Status        string `json:"status" binding:"omitempty,oneof=active inactive"`
}

func (h *ERPHandler) ListCustomers(c *gin.Context) {
	if !h.menuEnabled(c, "customers") {
		return
	}
	page, pageSize := pageParams(c)
	res, err := h.svc.ListCustomers(c.Request.Context(), repository.MasterFilter{
		Keyword: c.Query("keyword"), Status: c.Query("status"),
	}, page, pageSize)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, res)
}

func (h *ERPHandler) CreateCustomer(c *gin.Context) {
	var req customerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	item, err := h.svc.CreateCustomer(c.Request.Context(), domain.Customer{
		Code: req.Code, Name: req.Name, Contact: req.Contact, Phone: req.Phone,
		Address: req.Address, PaymentMethod: req.PaymentMethod, Status: req.Status,
	})
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, item)
}

func (h *ERPHandler) UpdateCustomer(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	var req customerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	if err := h.svc.UpdateCustomer(c.Request.Context(), domain.Customer{
		ID: id, Name: req.Name, Contact: req.Contact, Phone: req.Phone,
		Address: req.Address, PaymentMethod: req.PaymentMethod, Status: req.Status,
	}); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

func (h *ERPHandler) DeleteCustomer(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	if err := h.svc.DeleteCustomer(c.Request.Context(), id); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

// ---------- 供应商 ----------

func (h *ERPHandler) ListSuppliers(c *gin.Context) {
	page, pageSize := pageParams(c)
	res, err := h.svc.ListSuppliers(c.Request.Context(), repository.MasterFilter{
		Keyword: c.Query("keyword"), Status: c.Query("status"),
	}, page, pageSize)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, res)
}

func (h *ERPHandler) CreateSupplier(c *gin.Context) {
	var req customerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	item, err := h.svc.CreateSupplier(c.Request.Context(), domain.Supplier{
		Code: req.Code, Name: req.Name, Contact: req.Contact, Phone: req.Phone,
		Address: req.Address, PaymentMethod: req.PaymentMethod, Status: req.Status,
	})
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, item)
}

func (h *ERPHandler) UpdateSupplier(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	var req customerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	if err := h.svc.UpdateSupplier(c.Request.Context(), domain.Supplier{
		ID: id, Name: req.Name, Contact: req.Contact, Phone: req.Phone,
		Address: req.Address, PaymentMethod: req.PaymentMethod, Status: req.Status,
	}); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

func (h *ERPHandler) DeleteSupplier(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	if err := h.svc.DeleteSupplier(c.Request.Context(), id); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

// ---------- 仓库 ----------

type warehouseRequest struct {
	Code     string `json:"code" binding:"required,min=2,max=32,alphanum"`
	Name     string `json:"name" binding:"required,min=1,max=128"`
	Location string `json:"location" binding:"max=255"`
	Keeper   string `json:"keeper" binding:"max=64"`
	Phone    string `json:"phone" binding:"max=32"`
	Address  string `json:"address" binding:"max=255"`
	Status   string `json:"status" binding:"omitempty,oneof=active inactive"`
}

func (h *ERPHandler) ListWarehouses(c *gin.Context) {
	page, pageSize := pageParams(c)
	res, err := h.svc.ListWarehouses(c.Request.Context(), repository.MasterFilter{
		Keyword: c.Query("keyword"), Status: c.Query("status"),
	}, page, pageSize)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, res)
}

func (h *ERPHandler) CreateWarehouse(c *gin.Context) {
	var req warehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	item, err := h.svc.CreateWarehouse(c.Request.Context(), domain.Warehouse{
		Code: req.Code, Name: req.Name, Location: req.Location,
		Keeper: req.Keeper, Phone: req.Phone, Address: req.Address, Status: req.Status,
	})
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, item)
}

func (h *ERPHandler) UpdateWarehouse(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	var req warehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	if err := h.svc.UpdateWarehouse(c.Request.Context(), domain.Warehouse{
		ID: id, Name: req.Name, Location: req.Location,
		Keeper: req.Keeper, Phone: req.Phone, Address: req.Address, Status: req.Status,
	}); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

func (h *ERPHandler) DeleteWarehouse(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	if err := h.svc.DeleteWarehouse(c.Request.Context(), id); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

// ---------- 商品 ----------

type goodsRequest struct {
	Code          string  `json:"code" binding:"required,min=2,max=32,alphanum"`
	Name          string  `json:"name" binding:"required,min=1,max=128"`
	Spec          string  `json:"spec" binding:"max=128"`
	CategoryID    *int64  `json:"category_id" binding:"omitempty,gt=0"`
	BrandID       *int64  `json:"brand_id" binding:"omitempty,gt=0"`
	Unit          string  `json:"unit" binding:"max=16"`
	UnitID        *int64  `json:"unit_id" binding:"omitempty,gt=0"`
	Price         float64 `json:"price" binding:"gte=0"`
	Cost          float64 `json:"cost" binding:"gte=0"`
	PurchasePrice float64 `json:"purchase_price" binding:"gte=0"`
	SalePrice     float64 `json:"sale_price" binding:"gte=0"`
	MinQty        float64 `json:"min_qty" binding:"gte=0"`
	Status        string  `json:"status" binding:"omitempty,oneof=active inactive"`
}

func (h *ERPHandler) ListGoods(c *gin.Context) {
	if !h.menuEnabled(c, "goods") {
		return
	}
	page, pageSize := pageParams(c)
	res, err := h.svc.ListGoods(c.Request.Context(), repository.MasterFilter{
		Keyword: c.Query("keyword"), Status: c.Query("status"),
	}, page, pageSize)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, res)
}

func (h *ERPHandler) CreateGoods(c *gin.Context) {
	var req goodsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	item, err := h.svc.CreateGoods(c.Request.Context(), domain.Goods{
		Code: req.Code, Name: req.Name, Spec: req.Spec, CategoryID: req.CategoryID, BrandID: req.BrandID,
		Unit: req.Unit, UnitID: req.UnitID, Price: req.Price, Cost: req.Cost,
		PurchasePrice: req.PurchasePrice, SalePrice: req.SalePrice, MinQty: req.MinQty, Status: req.Status,
	})
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, item)
}

func (h *ERPHandler) UpdateGoods(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	var req goodsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	if err := h.svc.UpdateGoods(c.Request.Context(), domain.Goods{
		ID: id, Name: req.Name, Spec: req.Spec, CategoryID: req.CategoryID, BrandID: req.BrandID,
		Unit: req.Unit, UnitID: req.UnitID, Price: req.Price, Cost: req.Cost,
		PurchasePrice: req.PurchasePrice, SalePrice: req.SalePrice, MinQty: req.MinQty, Status: req.Status,
	}); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

func (h *ERPHandler) DeleteGoods(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	if err := h.svc.DeleteGoods(c.Request.Context(), id); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

func (h *ERPHandler) ListCategories(c *gin.Context) {
	if !h.menuEnabled(c, "goods") {
		return
	}
	list, err := h.svc.ListCategories(c.Request.Context())
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"list": list})
}

type categoryRequest struct {
	ParentID *int64 `json:"parent_id" binding:"omitempty,gt=0"`
	Name     string `json:"name" binding:"required,min=1,max=128"`
	Sort     int    `json:"sort" binding:"gte=0,lte=9999"`
}

func (h *ERPHandler) CreateCategory(c *gin.Context) {
	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	item, err := h.svc.CreateCategory(c.Request.Context(), domain.Category{
		ParentID: req.ParentID, Name: req.Name, Sort: req.Sort,
	})
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, item)
}

func (h *ERPHandler) UpdateCategory(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	if err := h.svc.UpdateCategory(c.Request.Context(), domain.Category{
		ID: id, ParentID: req.ParentID, Name: req.Name, Sort: req.Sort,
	}); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

func (h *ERPHandler) DeleteCategory(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	if err := h.svc.DeleteCategory(c.Request.Context(), id); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

type unitRequest struct {
	Name   string `json:"name" binding:"required,min=1,max=16"`
	Status string `json:"status" binding:"omitempty,oneof=active inactive"`
}

func (h *ERPHandler) ListUnits(c *gin.Context) {
	list, err := h.svc.ListUnits(c.Request.Context(), c.Query("status"))
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"list": list})
}

func (h *ERPHandler) CreateUnit(c *gin.Context) {
	var req unitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	item, err := h.svc.CreateUnit(c.Request.Context(), domain.Unit{Name: req.Name, Status: req.Status})
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, item)
}

func (h *ERPHandler) UpdateUnit(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	var req unitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	if err := h.svc.UpdateUnit(c.Request.Context(), domain.Unit{ID: id, Name: req.Name, Status: req.Status}); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

func (h *ERPHandler) DeleteUnit(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	if err := h.svc.DeleteUnit(c.Request.Context(), id); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

func (h *ERPHandler) ListBrands(c *gin.Context) {
	if !h.menuEnabled(c, "goods") {
		return
	}
	list, err := h.svc.ListBrands(c.Request.Context())
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"list": list})
}

type brandRequest struct {
	Name string `json:"name" binding:"required,min=1,max=128"`
}

func (h *ERPHandler) UpdateBrand(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	var req brandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	if err := h.svc.UpdateBrand(c.Request.Context(), domain.Brand{ID: id, Name: req.Name}); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

func (h *ERPHandler) DeleteBrand(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	if err := h.svc.DeleteBrand(c.Request.Context(), id); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

func (h *ERPHandler) CreateBrand(c *gin.Context) {
	var req brandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	item, err := h.svc.CreateBrand(c.Request.Context(), domain.Brand{Name: req.Name})
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, item)
}

// ---------- 库存 ----------

func (h *ERPHandler) ListStocks(c *gin.Context) {
	if !h.menuEnabled(c, "stock") {
		return
	}
	page, pageSize := pageParams(c)
	f := repository.StockFilter{
		WarehouseID: queryInt64(c, "warehouse_id"),
		Keyword:     c.Query("keyword"),
	}
	if c.Query("logs") == "1" {
		res, err := h.svc.ListStockLogs(c.Request.Context(), f, page, pageSize)
		if err != nil {
			httpx.FailFromError(c, err)
			return
		}
		httpx.Success(c, res)
		return
	}
	res, err := h.svc.ListStocks(c.Request.Context(), f, page, pageSize)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, res)
}

type stockAdjustmentRequest struct {
	WarehouseID int64   `json:"warehouse_id" binding:"required,gt=0"`
	GoodsID     int64   `json:"goods_id" binding:"required,gt=0"`
	Direction   string  `json:"direction" binding:"required,oneof=in out"`
	Qty         float64 `json:"qty" binding:"required,gt=0"`
	Reason      string  `json:"reason" binding:"max=255"`
}

func (h *ERPHandler) CreateStockAdjustment(c *gin.Context) {
	uid, uname, ok := currentOperator(c)
	if !ok {
		return
	}
	var req stockAdjustmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	item, err := h.svc.CreateStockAdjustment(c.Request.Context(), req.WarehouseID, req.GoodsID,
		req.Direction, req.Qty, req.Reason, uid, uname)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, item)
}

func (h *ERPHandler) GetStockAdjustment(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	item, err := h.svc.GetStockAdjustment(c.Request.Context(), id)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, item)
}

func (h *ERPHandler) SubmitStockAdjustment(c *gin.Context) {
	uid, uname, ok := currentOperator(c)
	if !ok {
		return
	}
	id, err := pathID(c)
	if err != nil {
		return
	}
	if err := h.svc.SubmitStockAdjustment(c.Request.Context(), id, uid, uname); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

func (h *ERPHandler) DeleteDraftStockAdjustment(c *gin.Context) {
	uid, uname, ok := currentOperator(c)
	if !ok {
		return
	}
	id, err := pathID(c)
	if err != nil {
		return
	}
	if err := h.svc.DeleteDraftStockAdjustment(c.Request.Context(), id, uid, uname); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

// ---------- 报表 ----------

func (h *ERPHandler) StockSummaryReport(c *gin.Context) {
	page, pageSize := pageParams(c)
	res, err := h.svc.ListStocks(c.Request.Context(), repository.StockFilter{
		WarehouseID: queryInt64(c, "warehouse_id"), Keyword: c.Query("keyword"),
	}, page, pageSize)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, res)
}

func (h *ERPHandler) StockFlowReport(c *gin.Context) {
	rows, err := h.svc.StockFlowReport(c.Request.Context(),
		c.Query("from"), c.Query("to"), queryInt64(c, "warehouse_id"))
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"list": rows})
}

// ---------- 库存盘点 ----------

type stockCheckItemRequest struct {
	GoodsID   int64   `json:"goods_id" binding:"required,gt=0"`
	ActualQty float64 `json:"actual_qty" binding:"gte=0"`
}

type stockCheckRequest struct {
	WarehouseID int64                   `json:"warehouse_id" binding:"required,gt=0"`
	Remark      string                  `json:"remark" binding:"max=255"`
	Items       []stockCheckItemRequest `json:"items" binding:"required,min=1,max=200,dive"`
}

func (h *ERPHandler) ListStockChecks(c *gin.Context) {
	page, pageSize := pageParams(c)
	res, err := h.svc.ListStockChecks(c.Request.Context(), repository.OrderFilter{
		Status: c.Query("status"), Keyword: c.Query("keyword"),
	}, page, pageSize)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, res)
}

func (h *ERPHandler) GetStockCheck(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	item, err := h.svc.GetStockCheck(c.Request.Context(), id)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, item)
}

func (h *ERPHandler) CreateStockCheck(c *gin.Context) {
	uid, uname, ok := currentOperator(c)
	if !ok {
		return
	}
	var req stockCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	items := make([]domain.StockCheckItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, domain.StockCheckItem{GoodsID: it.GoodsID, ActualQty: it.ActualQty})
	}
	item, err := h.svc.CreateStockCheck(c.Request.Context(), domain.StockCheck{
		WarehouseID: req.WarehouseID, Remark: req.Remark, CreatedBy: uid, Items: items,
	}, uname)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, item)
}

func (h *ERPHandler) SubmitStockCheck(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.SubmitStockCheck(c.Request.Context(), id, uid, uname)
	})
}

func (h *ERPHandler) DeleteDraftStockCheck(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.DeleteDraftStockCheck(c.Request.Context(), id, uid, uname)
	})
}

// ---------- 库存调拨 ----------

type transferItemRequest struct {
	GoodsID int64   `json:"goods_id" binding:"required,gt=0"`
	Qty     float64 `json:"qty" binding:"required,gt=0"`
}

type transferRequest struct {
	FromWarehouseID int64                 `json:"from_warehouse_id" binding:"required,gt=0"`
	ToWarehouseID   int64                 `json:"to_warehouse_id" binding:"required,gt=0"`
	Remark          string                `json:"remark" binding:"max=255"`
	Items           []transferItemRequest `json:"items" binding:"required,min=1,max=100,dive"`
}

func (h *ERPHandler) ListTransfers(c *gin.Context) {
	page, pageSize := pageParams(c)
	res, err := h.svc.ListTransfers(c.Request.Context(), repository.OrderFilter{
		Status: c.Query("status"), Keyword: c.Query("keyword"),
	}, page, pageSize)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, res)
}

func (h *ERPHandler) GetTransfer(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	item, err := h.svc.GetTransfer(c.Request.Context(), id)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, item)
}

func (h *ERPHandler) CreateTransfer(c *gin.Context) {
	uid, uname, ok := currentOperator(c)
	if !ok {
		return
	}
	var req transferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	items := make([]domain.TransferItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, domain.TransferItem{GoodsID: it.GoodsID, Qty: it.Qty})
	}
	item, err := h.svc.CreateTransfer(c.Request.Context(), domain.StockTransfer{
		FromWarehouseID: req.FromWarehouseID, ToWarehouseID: req.ToWarehouseID,
		Remark: req.Remark, CreatedBy: uid, Items: items,
	}, uname)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, item)
}

func (h *ERPHandler) SubmitTransfer(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.SubmitTransfer(c.Request.Context(), id, uid, uname)
	})
}

func (h *ERPHandler) DeleteDraftTransfer(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.DeleteDraftTransfer(c.Request.Context(), id, uid, uname)
	})
}

func (h *ERPHandler) ApproveTransfer(c *gin.Context) {
	var req auditActionRequest
	if !bindOptionalJSON(c, &req) {
		return
	}
	scope, scopeID := scopeFrom(c)
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.ApproveTransfer(c.Request.Context(), id, uid, uname, req.Comment, scope, scopeID)
	})
}

func (h *ERPHandler) RejectTransfer(c *gin.Context) {
	var req auditActionRequest
	if !bindOptionalJSON(c, &req) {
		return
	}
	scope, scopeID := scopeFrom(c)
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.RejectTransfer(c.Request.Context(), id, uid, uname, req.Comment, scope, scopeID)
	})
}

func (h *ERPHandler) CompleteTransfer(c *gin.Context) {
	scope, scopeID := scopeFrom(c)
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.CompleteTransfer(c.Request.Context(), id, uid, uname, scope, scopeID)
	})
}

// ---------- 采购单 ----------

type orderItemRequest struct {
	GoodsID int64   `json:"goods_id" binding:"required,gt=0"`
	Qty     float64 `json:"qty" binding:"required,gt=0"`
	Price   float64 `json:"price" binding:"gte=0"`
}

type purchaseRequest struct {
	SupplierID  int64              `json:"supplier_id" binding:"required,gt=0"`
	WarehouseID int64              `json:"warehouse_id" binding:"required,gt=0"`
	Items       []orderItemRequest `json:"items" binding:"required,min=1,max=100,dive"`
}

func (h *ERPHandler) ListPurchases(c *gin.Context) {
	if !h.menuEnabled(c, "purchase") {
		return
	}
	page, pageSize := pageParams(c)
	res, err := h.svc.ListPurchases(c.Request.Context(), repository.OrderFilter{
		Status: c.Query("status"), Keyword: c.Query("keyword"),
	}, page, pageSize)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, res)
}

func (h *ERPHandler) GetPurchase(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	o, err := h.svc.GetPurchase(c.Request.Context(), id)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, o)
}

func (h *ERPHandler) CreatePurchase(c *gin.Context) {
	uid, uname, ok := currentOperator(c)
	if !ok {
		return
	}
	var req purchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	scope, scopeID := scopeFrom(c)
	o, err := h.svc.CreatePurchase(c.Request.Context(), domain.PurchaseOrder{
		SupplierID: req.SupplierID, WarehouseID: req.WarehouseID,
		CreatedBy: uid, Items: toItems(req.Items),
	}, uname, scope, scopeID)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, o)
}

func (h *ERPHandler) UpdatePurchase(c *gin.Context) {
	uid, uname, ok := currentOperator(c)
	if !ok {
		return
	}
	id, err := pathID(c)
	if err != nil {
		return
	}
	var req purchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	scope, scopeID := scopeFrom(c)
	if err := h.svc.UpdatePurchase(c.Request.Context(), domain.PurchaseOrder{
		ID: id, SupplierID: req.SupplierID, WarehouseID: req.WarehouseID,
		CreatedBy: uid, Items: toItems(req.Items),
	}, uname, scope, scopeID); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

func (h *ERPHandler) SubmitPurchase(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.SubmitPurchase(c.Request.Context(), id, uid, uname)
	})
}

func (h *ERPHandler) ApprovePurchase(c *gin.Context) {
	var req auditActionRequest
	if !bindOptionalJSON(c, &req) {
		return
	}
	scope, scopeID := scopeFrom(c)
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.ApprovePurchase(c.Request.Context(), id, uid, uname, req.Comment, scope, scopeID)
	})
}

func (h *ERPHandler) RejectPurchase(c *gin.Context) {
	h.reject(c, func(id, uid int64, uname, comment string) error {
		scope, scopeID := scopeFrom(c)
		return h.svc.RejectPurchase(c.Request.Context(), id, uid, uname, comment, scope, scopeID)
	})
}

func (h *ERPHandler) WithdrawPurchase(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.WithdrawPurchase(c.Request.Context(), id, uid, uname)
	})
}

func (h *ERPHandler) DeleteDraftPurchase(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.DeleteDraftPurchase(c.Request.Context(), id, uid, uname)
	})
}

func (h *ERPHandler) ReceivePurchase(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.ReceivePurchase(c.Request.Context(), id, uid, uname)
	})
}

// ---------- 销售单 ----------

type salesRequest struct {
	CustomerID  int64              `json:"customer_id" binding:"required,gt=0"`
	WarehouseID int64              `json:"warehouse_id" binding:"required,gt=0"`
	Items       []orderItemRequest `json:"items" binding:"required,min=1,max=100,dive"`
}

func (h *ERPHandler) ListSales(c *gin.Context) {
	if !h.menuEnabled(c, "sales") {
		return
	}
	page, pageSize := pageParams(c)
	res, err := h.svc.ListSales(c.Request.Context(), repository.OrderFilter{
		Status: c.Query("status"), Keyword: c.Query("keyword"),
	}, page, pageSize)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, res)
}

func (h *ERPHandler) GetSales(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		return
	}
	o, err := h.svc.GetSales(c.Request.Context(), id)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, o)
}

func (h *ERPHandler) CreateSales(c *gin.Context) {
	uid, uname, ok := currentOperator(c)
	if !ok {
		return
	}
	var req salesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	scope, scopeID := scopeFrom(c)
	o, err := h.svc.CreateSales(c.Request.Context(), domain.SalesOrder{
		CustomerID: req.CustomerID, WarehouseID: req.WarehouseID,
		CreatedBy: uid, Items: toItems(req.Items),
	}, uname, scope, scopeID)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, o)
}

func (h *ERPHandler) UpdateSales(c *gin.Context) {
	uid, uname, ok := currentOperator(c)
	if !ok {
		return
	}
	id, err := pathID(c)
	if err != nil {
		return
	}
	var req salesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	scope, scopeID := scopeFrom(c)
	if err := h.svc.UpdateSales(c.Request.Context(), domain.SalesOrder{
		ID: id, CustomerID: req.CustomerID, WarehouseID: req.WarehouseID,
		CreatedBy: uid, Items: toItems(req.Items),
	}, uname, scope, scopeID); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

func (h *ERPHandler) SubmitSales(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.SubmitSales(c.Request.Context(), id, uid, uname)
	})
}

func (h *ERPHandler) ApproveSales(c *gin.Context) {
	var req auditActionRequest
	if !bindOptionalJSON(c, &req) {
		return
	}
	scope, scopeID := scopeFrom(c)
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.ApproveSales(c.Request.Context(), id, uid, uname, req.Comment, scope, scopeID)
	})
}

func (h *ERPHandler) RejectSales(c *gin.Context) {
	h.reject(c, func(id, uid int64, uname, comment string) error {
		scope, scopeID := scopeFrom(c)
		return h.svc.RejectSales(c.Request.Context(), id, uid, uname, comment, scope, scopeID)
	})
}

func (h *ERPHandler) WithdrawSales(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.WithdrawSales(c.Request.Context(), id, uid, uname)
	})
}

func (h *ERPHandler) DeleteDraftSales(c *gin.Context) {
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.DeleteDraftSales(c.Request.Context(), id, uid, uname)
	})
}

func (h *ERPHandler) ShipSales(c *gin.Context) {
	scope, scopeID := scopeFrom(c)
	h.transition(c, func(id, uid int64, uname string) error {
		return h.svc.ShipSales(c.Request.Context(), id, uid, uname, scope, scopeID)
	})
}

// auditActionRequest 是审核动作的可选请求体：通过/驳回都允许带备注。
type auditActionRequest struct {
	Comment string `json:"comment" binding:"max=500"`
}

// bindOptionalJSON 允许空请求体：老前端与测试调用 /approve 时不带 body。
func bindOptionalJSON(c *gin.Context, dst any) bool {
	if c.Request.ContentLength == 0 {
		return true
	}
	if err := c.ShouldBindJSON(dst); err != nil {
		httpx.FailBindError(c, err)
		return false
	}
	return true
}

// reject 是采购/销售单驳回的共用流程：取路径 id、读可选备注、写操作记录。
func (h *ERPHandler) reject(c *gin.Context, fn func(id, uid int64, uname, comment string) error) {
	uid, uname, ok := currentOperator(c)
	if !ok {
		return
	}
	id, err := pathID(c)
	if err != nil {
		return
	}
	var req auditActionRequest
	if !bindOptionalJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.Comment) == "" {
		httpx.Fail(c, http.StatusBadRequest, httpx.KeyValidationFailed)
		return
	}
	if err := fn(id, uid, uname, req.Comment); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

// validBillType 收窄操作记录接口允许的单据类型，避免任意字符串打进来。
func validBillType(t string) bool {
	switch t {
	case "purchase", "sales", "purchase-return", "sale-return", "stock-adjustment", "transfer", "stock-check":
		return true
	default:
		return false
	}
}

func (h *ERPHandler) transition(c *gin.Context, fn func(id, uid int64, uname string) error) {
	uid, uname, ok := currentOperator(c)
	if !ok {
		return
	}
	id, err := pathID(c)
	if err != nil {
		return
	}
	if err := fn(id, uid, uname); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"id": id})
}

func (h *ERPHandler) BillOperations(c *gin.Context) {
	billType := c.Param("type")
	if !validBillType(billType) {
		httpx.Fail(c, http.StatusBadRequest, httpx.KeyInvalidID)
		return
	}
	id, err := pathID(c)
	if err != nil {
		return
	}
	list, err := h.svc.ListBillOperations(c.Request.Context(), billType, id)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"list": list})
}

func toItems(in []orderItemRequest) []domain.OrderItem {
	out := make([]domain.OrderItem, 0, len(in))
	for _, it := range in {
		out = append(out, domain.OrderItem{GoodsID: it.GoodsID, Qty: it.Qty, Price: it.Price})
	}
	return out
}

// ---------- 开关 ----------

func (h *ERPHandler) GetSwitches(c *gin.Context) {
	scope, id := scopeFrom(c)
	cfg, err := h.switches.Config(c.Request.Context(), scope, id)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"scope": scope, "scope_id": id, "config": cfg})
}

// ListBugSwitches 返回缺陷目录（含未接线项），给开关页渲染用。
func (h *ERPHandler) ListBugSwitches(c *gin.Context) {
	httpx.Success(c, gin.H{"list": domain.BugSwitches})
}

type putSwitchRequest struct {
	BugCode string `json:"bug_code" binding:"required,max=32"`
	Enabled bool   `json:"enabled"`
}

// SetBug 是给演示/调试用的单缺陷开关；正式下发走 /internal/switches。
func (h *ERPHandler) SetBug(c *gin.Context) {
	var req putSwitchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailBindError(c, err)
		return
	}
	// 拼错的码写进配置后没有任何分支会读它，表现为「开关打开了却没变化」，
	// 极难排查。目录里没有的码直接拒绝。
	if !domain.BugDeclared(req.BugCode) {
		httpx.Fail(c, http.StatusBadRequest, httpx.KeyInvalidParent)
		return
	}
	scope, id := scopeFrom(c)
	cfg, err := h.switches.Config(c.Request.Context(), scope, id)
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	if cfg.Bugs == nil {
		cfg.Bugs = map[string]bool{}
	}
	cfg.Bugs[req.BugCode] = req.Enabled
	if err := h.switches.Set(c.Request.Context(), scope, id, cfg); err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, cfg)
}

// AuditSwitches 返回所有作用域配置，供管理端排查。
func (h *ERPHandler) AuditSwitches(c *gin.Context) {
	list, err := h.switches.List(c.Request.Context())
	if err != nil {
		httpx.FailFromError(c, err)
		return
	}
	httpx.Success(c, gin.H{"list": list})
}

// queryInt64 读取查询参数里的整数；缺失或非法都按 0（不过滤）。
func queryInt64(c *gin.Context, key string) int64 {
	v, err := strconv.ParseInt(c.Query(key), 10, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}
