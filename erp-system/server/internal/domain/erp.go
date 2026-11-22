// 有问题欢迎联系 hot_kun@hotmail.com
package domain

import "time"

// 缺陷编码。开关打开时，对应逻辑走「有 bug」的分支。
const (
	BugAmountError    = "BUG-001" // 金额少算 1 分
	BugNegativeStock  = "BUG-002" // 出库允许库存为负
	BugApproveSelf    = "BUG-003" // 允许自己审批自己的单据
	BugEditCompleted  = "BUG-005" // 已完成单据仍可编辑
	BugDuplicatePage  = "BUG-004" // 列表翻页重复（列表侧实现）
	BugOrphanCustomer = "BUG-006" // 删除客户后订单仍可见（预留）
)

// BugSwitch 是一条可注入的教学缺陷。
//
// Wired 为 false 表示只声明了常量、业务代码里还没有对应分支：打开也不会有任何变化。
// 开关页必须把它和真正生效的缺陷区分开，否则做验收的人会以为「开关坏了」。
type BugSwitch struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Area  string `json:"area"`
	Wired bool   `json:"wired"`
}

// BugSwitches 是缺陷目录的唯一真源：开关页、内部接口与平台面板都读它。
//
// 新增缺陷 = 这里加一条 + 在业务里写 BugEnabled 分支 + 把 Wired 置 true。
var BugSwitches = []BugSwitch{
	{Code: BugAmountError, Name: "金额少算 1 分", Area: "purchase/sales", Wired: true},
	{Code: BugNegativeStock, Name: "出库允许负库存", Area: "sales/transfer/return", Wired: true},
	{Code: BugApproveSelf, Name: "允许自审自批", Area: "approve/reject", Wired: true},
	{Code: BugEditCompleted, Name: "已完成单据仍可编辑", Area: "purchase/sales", Wired: true},
	{Code: BugDuplicatePage, Name: "列表翻页重复第 1 页", Area: "list", Wired: false},
	{Code: BugOrphanCustomer, Name: "删除客户后订单仍可见", Area: "customers/sales", Wired: false},
}

// WiredBugSwitches 只返回已接线的缺陷。平台开关面板只下发这些——
// 把未接线的也下发，等于给平台一个点了没反应的开关。
func WiredBugSwitches() []BugSwitch {
	out := make([]BugSwitch, 0, len(BugSwitches))
	for _, b := range BugSwitches {
		if b.Wired {
			out = append(out, b)
		}
	}
	return out
}

// BugDeclared 判断缺陷码是否在目录里。用于挡住拼错的 bug_code 写进配置。
func BugDeclared(code string) bool {
	for _, b := range BugSwitches {
		if b.Code == code {
			return true
		}
	}
	return false
}

// MenuSwitch 是一个可被平台关闭的业务菜单。
//
// Code 与网关（menuCodeOfPath）拦截时用的码完全一致；平台只认这两个字段，
// 不认识具体业务——换一个系统就换一份注册表。
type MenuSwitch struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	NameEN string `json:"name_en"`
}

// MenuSwitches 是菜单开关的注册表：菜单被关掉时对应接口直接 403。
//
// 这是「哪些菜单可关」的唯一真源：平台通过 GET /internal/menus 读取它来渲染
// 勾选项，网关用 Code 做拦截。新增一个可关的菜单 = 这里加一条 + 网关映射里加资源段。
var MenuSwitches = []MenuSwitch{
	{Code: "goods", Name: "商品（含分类/品牌/单位）", NameEN: "Goods (incl. categories, brands, units)"},
	{Code: "customers", Name: "客户", NameEN: "Customers"},
	{Code: "suppliers", Name: "供应商", NameEN: "Suppliers"},
	{Code: "warehouses", Name: "仓库", NameEN: "Warehouses"},
	{Code: "stock", Name: "库存（含调整/调拨/盘点）", NameEN: "Stock (incl. adjustments, transfers, checks)"},
	{Code: "reports", Name: "库存报表", NameEN: "Stock reports"},
	{Code: "purchase", Name: "采购", NameEN: "Purchase"},
	{Code: "sales", Name: "销售", NameEN: "Sales"},
	{Code: "purchase_return", Name: "采购退货", NameEN: "Purchase returns"},
	{Code: "sales_return", Name: "销售退货", NameEN: "Sales returns"},
	{Code: "audit", Name: "审核中心", NameEN: "Audit center"},
}

// SwitchItem 是开关组里的一项：平台只按「组 → code」透传，语义由本系统决定。
type SwitchItem struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	NameEN string `json:"name_en,omitempty"`
	Area   string `json:"area,omitempty"`
}

// SwitchGroup 是被测系统自报的一组可切换项（GET /internal/switches/schema）。
//
// Default 是「清单里有、但没配置过」时的状态：缺陷默认关、菜单默认开。
// 平台不内置任何组名与 code，换一个镜像就换一份清单。
type SwitchGroup struct {
	Code    string       `json:"code"`
	Name    string       `json:"name"`
	NameEN  string       `json:"name_en"`
	Default *bool        `json:"default,omitempty"`
	Items   []SwitchItem `json:"items"`
}

// SwitchSchema 是平台面板渲染开关需要的全部信息。
//
// 分三组而不是「缺陷 / 菜单」两组：教师按业务域找开关比按实现方式找更符合直觉。
func SwitchSchema() []SwitchGroup {
	off, on := false, true

	// 缺陷按 Area 归到三个业务域
	areaGroup := func(area string) string {
		switch area {
		case "purchase/sales", "sales/transfer/return", "purchase", "sales":
			return "bugs-docs"
		case "approve/reject":
			return "bugs-approval"
		case "customers/sales", "customers":
			return "bugs-master"
		case "list":
			return "bugs-list"
		default:
			return "bugs-other"
		}
	}

	groups := []SwitchGroup{
		{Code: "bugs-docs", Name: "单据类缺陷", NameEN: "Document defects", Default: &off},
		{Code: "bugs-approval", Name: "审批类缺陷", NameEN: "Approval defects", Default: &off},
		{Code: "bugs-stock", Name: "库存类缺陷", NameEN: "Stock defects", Default: &off},
		{Code: "bugs-master", Name: "主数据类缺陷", NameEN: "Master-data defects", Default: &off},
		{Code: "bugs-list", Name: "列表类缺陷", NameEN: "List defects", Default: &off},
		{Code: "bugs-other", Name: "其它缺陷", NameEN: "Other defects", Default: &off},
		{Code: "menus", Name: "可关闭菜单", NameEN: "Closable menus", Default: &on},
	}

	index := make(map[string]int, len(groups))
	for i, g := range groups {
		index[g.Code] = i
	}

	for _, b := range WiredBugSwitches() {
		code := areaGroup(b.Area)
		// 库存缺陷单独成组，便于「只练库存」的场景一键打开
		if b.Code == BugNegativeStock {
			code = "bugs-stock"
		}
		if i, ok := index[code]; ok {
			groups[i].Items = append(groups[i].Items, SwitchItem{
				Code: b.Code, Name: b.Name, Area: b.Area,
			})
		}
	}

	for _, m := range MenuSwitches {
		groups[index["menus"]].Items = append(groups[index["menus"]].Items, SwitchItem{
			Code: m.Code, Name: m.Name, NameEN: m.NameEN,
		})
	}

	// 丢掉没有项的组：平台面板里出现空组只会让人困惑
	out := groups[:0]
	for _, g := range groups {
		if len(g.Items) > 0 {
			out = append(out, g)
		}
	}
	return out
}

// SwitchConfig 是某作用域下的一份开关配置。
//
// 缺失的键按「关闭」处理：默认不注入缺陷，避免新环境一上来就是坏的。
type SwitchConfig struct {
	Bugs  map[string]bool `json:"bugs"`
	Menus map[string]bool `json:"menus"`
}

type Customer struct {
	ID            int64     `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Contact       string    `json:"contact"`
	Phone         string    `json:"phone"`
	Address       string    `json:"address"`
	PaymentMethod string    `json:"payment_method"`
	Status        string    `json:"status"`
}

type Supplier struct {
	ID            int64     `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Contact       string    `json:"contact"`
	Phone         string    `json:"phone"`
	Address       string    `json:"address"`
	PaymentMethod string    `json:"payment_method"`
	Status        string    `json:"status"`
}

type Category struct {
	ID       int64  `json:"id"`
	ParentID *int64 `json:"parent_id"`
	Name     string `json:"name"`
	Sort     int    `json:"sort"`
}

type Brand struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Unit struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
}

type Goods struct {
	ID            int64     `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Spec          string    `json:"spec"`
	CategoryID    *int64    `json:"category_id"`
	BrandID       *int64    `json:"brand_id"`
	Unit          string    `json:"unit"`
	UnitID        *int64    `json:"unit_id"`
	Price         float64   `json:"price"`
	Cost          float64   `json:"cost"`
	PurchasePrice float64   `json:"purchase_price"`
	SalePrice     float64   `json:"sale_price"`
	MinQty        float64   `json:"min_qty"`
	Status        string    `json:"status"`
	StockQty      float64   `json:"stock_qty"`
}

type Warehouse struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Location  string    `json:"location"`
	Keeper    string    `json:"keeper"`
	Phone     string    `json:"phone"`
	Address   string    `json:"address"`
	Status    string    `json:"status"`
}

type Stock struct {
	WarehouseID   int64     `json:"warehouse_id"`
	WarehouseName string    `json:"warehouse_name"`
	GoodsID       int64     `json:"goods_id"`
	GoodsCode     string    `json:"goods_code"`
	GoodsName     string    `json:"goods_name"`
	Unit          string    `json:"unit"`
	Qty           float64   `json:"qty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type StockLog struct {
	ID          int64     `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	WarehouseID int64     `json:"warehouse_id"`
	GoodsID     int64     `json:"goods_id"`
	GoodsName   string    `json:"goods_name"`
	ChangeQty   float64   `json:"change_qty"`
	BizType     string    `json:"biz_type"`
	RefNo       string    `json:"ref_no"`
}

type StockAdjustment struct {
	ID            int64      `json:"id"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	AdjustmentNo  string     `json:"adjustment_no"`
	WarehouseID   int64      `json:"warehouse_id"`
	WarehouseName string     `json:"warehouse_name"`
	GoodsID       int64      `json:"goods_id"`
	GoodsName     string     `json:"goods_name"`
	Unit          string     `json:"unit"`
	Direction     string     `json:"direction"`
	Qty           float64    `json:"qty"`
	Reason        string     `json:"reason"`
	Status        string     `json:"status"`
	CreatedBy     int64      `json:"created_by"`
	SubmittedAt   *time.Time `json:"submitted_at"`
	ApprovedBy    *int64     `json:"approved_by"`
	ApprovedAt    *time.Time `json:"approved_at"`
}

// OrderItem 是采购/销售单共用的一行明细。
type OrderItem struct {
	ID        int64   `json:"id"`
	GoodsID   int64   `json:"goods_id"`
	GoodsName string  `json:"goods_name"`
	Qty       float64 `json:"qty"`
	Price     float64 `json:"price"`
	Amount    float64 `json:"amount"`
}

type PurchaseOrder struct {
	ID            int64       `json:"id"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	OrderNo       string      `json:"order_no"`
	SupplierID    int64       `json:"supplier_id"`
	SupplierName  string      `json:"supplier_name"`
	WarehouseID   int64       `json:"warehouse_id"`
	WarehouseName string      `json:"warehouse_name"`
	Status        string      `json:"status"`
	TotalAmount   float64     `json:"total_amount"`
	CreatedBy     int64       `json:"created_by"`
	SubmittedAt   *time.Time  `json:"submitted_at"`
	ApprovedBy    *int64      `json:"approved_by"`
	ApprovedAt    *time.Time  `json:"approved_at"`
	Items         []OrderItem `json:"items"`
}

type SalesOrder struct {
	ID            int64       `json:"id"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	OrderNo       string      `json:"order_no"`
	CustomerID    int64       `json:"customer_id"`
	CustomerName  string      `json:"customer_name"`
	WarehouseID   int64       `json:"warehouse_id"`
	WarehouseName string      `json:"warehouse_name"`
	Status        string      `json:"status"`
	TotalAmount   float64     `json:"total_amount"`
	CreatedBy     int64       `json:"created_by"`
	SubmittedAt   *time.Time  `json:"submitted_at"`
	ApprovedBy    *int64      `json:"approved_by"`
	ApprovedAt    *time.Time  `json:"approved_at"`
	Items         []OrderItem `json:"items"`
}

type BillOperation struct {
	ID           int64     `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	BillType     string    `json:"bill_type"`
	BillID       int64     `json:"bill_id"`
	Action       string    `json:"action"`
	OperatorID   int64     `json:"operator_id"`
	OperatorName string    `json:"operator_name"`
	Comment      string    `json:"comment"`
}

// PurchaseReturn 采购退货单。来源单必须是已完成的采购入库单；
// 供应商与仓库从来源单带出，前端不能改。
type PurchaseReturn struct {
	ID            int64       `json:"id"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	ReturnNo      string      `json:"return_no"`
	SourceOrderID int64       `json:"source_order_id"`
	SourceOrderNo string      `json:"source_order_no"`
	SupplierID    int64       `json:"supplier_id"`
	SupplierName  string      `json:"supplier_name"`
	WarehouseID   int64       `json:"warehouse_id"`
	WarehouseName string      `json:"warehouse_name"`
	Status        string      `json:"status"`
	TotalAmount   float64     `json:"total_amount"`
	Remark        string      `json:"remark"`
	CreatedBy     int64       `json:"created_by"`
	SubmittedAt   *time.Time  `json:"submitted_at"`
	ApprovedBy    *int64      `json:"approved_by"`
	ApprovedAt    *time.Time  `json:"approved_at"`
	Items         []OrderItem `json:"items"`
}

// SalesReturn 销售退货单。来源单必须是已完成的销售出库单。
type SalesReturn struct {
	ID            int64       `json:"id"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	ReturnNo      string      `json:"return_no"`
	SourceOrderID int64       `json:"source_order_id"`
	SourceOrderNo string      `json:"source_order_no"`
	CustomerID    int64       `json:"customer_id"`
	CustomerName  string      `json:"customer_name"`
	WarehouseID   int64       `json:"warehouse_id"`
	WarehouseName string      `json:"warehouse_name"`
	Status        string      `json:"status"`
	TotalAmount   float64     `json:"total_amount"`
	Remark        string      `json:"remark"`
	CreatedBy     int64       `json:"created_by"`
	SubmittedAt   *time.Time  `json:"submitted_at"`
	ApprovedBy    *int64      `json:"approved_by"`
	ApprovedAt    *time.Time  `json:"approved_at"`
	Items         []OrderItem `json:"items"`
}

// StockCheckItem 是盘点单的一行：账面数、实盘数与差异。
type StockCheckItem struct {
	ID        int64   `json:"id"`
	GoodsID   int64   `json:"goods_id"`
	GoodsName string  `json:"goods_name"`
	Unit      string  `json:"unit"`
	BookQty   float64 `json:"book_qty"`
	ActualQty float64 `json:"actual_qty"`
	DiffQty   float64 `json:"diff_qty"`
}

// StockCheck 是库存盘点单。审核通过时按差异生成库存调整单。
type StockCheck struct {
	ID            int64            `json:"id"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	CheckNo       string           `json:"check_no"`
	WarehouseID   int64            `json:"warehouse_id"`
	WarehouseName string           `json:"warehouse_name"`
	Status        string           `json:"status"`
	Remark        string           `json:"remark"`
	CreatedBy     int64            `json:"created_by"`
	SubmittedAt   *time.Time       `json:"submitted_at"`
	ApprovedBy    *int64           `json:"approved_by"`
	ApprovedAt    *time.Time       `json:"approved_at"`
	Items         []StockCheckItem `json:"items"`
}

// TransferItem 是调拨单的一行明细。
type TransferItem struct {
	ID        int64   `json:"id"`
	GoodsID   int64   `json:"goods_id"`
	GoodsName string  `json:"goods_name"`
	Unit      string  `json:"unit"`
	Qty       float64 `json:"qty"`
}

// StockTransfer 是仓库调拨单。完成时从调出仓扣减、向调入仓增加。
type StockTransfer struct {
	ID              int64          `json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	TransferNo      string         `json:"transfer_no"`
	FromWarehouseID int64          `json:"from_warehouse_id"`
	FromName        string         `json:"from_warehouse_name"`
	ToWarehouseID   int64          `json:"to_warehouse_id"`
	ToName          string         `json:"to_warehouse_name"`
	Status          string         `json:"status"`
	Remark          string         `json:"remark"`
	CreatedBy       int64          `json:"created_by"`
	SubmittedAt     *time.Time     `json:"submitted_at"`
	ApprovedBy      *int64         `json:"approved_by"`
	ApprovedAt      *time.Time     `json:"approved_at"`
	CompletedAt     *time.Time     `json:"completed_at"`
	Items           []TransferItem `json:"items"`
}

// Dashboard 是首页仪表盘的聚合数据。
//
// 只读、跨全部业务表，任何登录角色都能看：它不改变业务数据，也不提供
// 超出列表页的字段。待办只取最近 5 条，完整列表在审核中心。
type Dashboard struct {
	GoodsCount      int64       `json:"goods_count"`
	CustomerCount   int64       `json:"customer_count"`
	SupplierCount   int64       `json:"supplier_count"`
	WarehouseCount  int64       `json:"warehouse_count"`
	StockGoodsCount int64       `json:"stock_goods_count"`
	StockTotal      float64     `json:"stock_total"`
	PendingTotal    int64       `json:"pending_total"`
	PendingIn       int64       `json:"pending_in"`
	PendingOut      int64       `json:"pending_out"`
	StockAlerts     int64       `json:"stock_alerts"`
	Pending         []AuditBill `json:"pending"`
}

// AuditBill 是审核中心聚合视图的一行，跨采购/销售/两类退货统一形状。
//
// BillType 用 URL 里的短名（purchase / sales / purchase-return / sale-return），
// 前端据此决定详情与审核动作打到哪个端点。
type AuditBill struct {
	BillType      string     `json:"bill_type"`
	BillID        int64      `json:"bill_id"`
	BillNo        string     `json:"bill_no"`
	PartnerName   string     `json:"partner_name"`
	WarehouseName string     `json:"warehouse_name"`
	TotalAmount   float64    `json:"total_amount"`
	Direction     string     `json:"direction"`
	Qty           float64    `json:"qty"`
	Status        string     `json:"status"`
	CreatedBy     int64      `json:"created_by"`
	CreatorName   string     `json:"creator_name"`
	SubmittedAt   *time.Time `json:"submitted_at"`
	ApprovedBy    *int64     `json:"approved_by"`
	ApproverName  string     `json:"approver_name"`
	ApprovedAt    *time.Time `json:"approved_at"`
}
