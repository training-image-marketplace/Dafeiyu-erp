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
