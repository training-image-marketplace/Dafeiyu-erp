// 有问题欢迎联系 hot_kun@hotmail.com
package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"practical-training/erp-system/internal/domain"
	"practical-training/erp-system/internal/repository"
)

// ERPService 负责主数据、库存与采购/销售单据。
type ERPService struct {
	master   *repository.ERPMasterRepository
	orders   *repository.ERPOrderRepository
	switches *SwitchService
}

func NewERPService(
	master *repository.ERPMasterRepository,
	orders *repository.ERPOrderRepository,
	switches *SwitchService,
) *ERPService {
	return &ERPService{master: master, orders: orders, switches: switches}
}

// ---------- 客户 / 供应商 / 仓库 ----------

func (s *ERPService) ListCustomers(ctx context.Context, f repository.MasterFilter, page, pageSize int) (Page[domain.Customer], error) {
	page, pageSize = normalizePage(page, pageSize)
	total, err := s.master.CountCustomers(ctx, f)
	if err != nil {
		return Page[domain.Customer]{}, err
	}
	page, offset := clampPage(page, pageSize, total)
	list, err := s.master.ListCustomers(ctx, f, pageSize, offset)
	if err != nil {
		return Page[domain.Customer]{}, err
	}
	return Page[domain.Customer]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *ERPService) CreateCustomer(ctx context.Context, c domain.Customer) (domain.Customer, error) {
	if err := s.ensureCodeFree(ctx, "erp_customers", c.Code, 0); err != nil {
		return domain.Customer{}, err
	}
	c.Status = orActive(c.Status)
	return s.master.CreateCustomer(ctx, c)
}

func (s *ERPService) UpdateCustomer(ctx context.Context, c domain.Customer) error {
	c.Status = orActive(c.Status)
	return s.master.UpdateCustomer(ctx, c)
}

func (s *ERPService) DeleteCustomer(ctx context.Context, id int64) error {
	n, err := s.master.CustomerRefs(ctx, id)
	if err != nil {
		return err
	}
	if err := domain.HasDependents(map[string]int64{"sales_orders": n}); err != nil {
		return err
	}
	return s.master.DeleteCustomer(ctx, id)
}

func (s *ERPService) ListSuppliers(ctx context.Context, f repository.MasterFilter, page, pageSize int) (Page[domain.Supplier], error) {
	page, pageSize = normalizePage(page, pageSize)
	total, err := s.master.CountSuppliers(ctx, f)
	if err != nil {
		return Page[domain.Supplier]{}, err
	}
	page, offset := clampPage(page, pageSize, total)
	list, err := s.master.ListSuppliers(ctx, f, pageSize, offset)
	if err != nil {
		return Page[domain.Supplier]{}, err
	}
	return Page[domain.Supplier]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *ERPService) CreateSupplier(ctx context.Context, v domain.Supplier) (domain.Supplier, error) {
	if err := s.ensureCodeFree(ctx, "erp_suppliers", v.Code, 0); err != nil {
		return domain.Supplier{}, err
	}
	v.Status = orActive(v.Status)
	return s.master.CreateSupplier(ctx, v)
}

func (s *ERPService) UpdateSupplier(ctx context.Context, v domain.Supplier) error {
	v.Status = orActive(v.Status)
	return s.master.UpdateSupplier(ctx, v)
}

func (s *ERPService) DeleteSupplier(ctx context.Context, id int64) error {
	n, err := s.master.SupplierRefs(ctx, id)
	if err != nil {
		return err
	}
	if err := domain.HasDependents(map[string]int64{"purchase_orders": n}); err != nil {
		return err
	}
	return s.master.DeleteSupplier(ctx, id)
}

func (s *ERPService) ListWarehouses(ctx context.Context, f repository.MasterFilter, page, pageSize int) (Page[domain.Warehouse], error) {
	page, pageSize = normalizePage(page, pageSize)
	total, err := s.master.CountWarehouses(ctx, f)
	if err != nil {
		return Page[domain.Warehouse]{}, err
	}
	page, offset := clampPage(page, pageSize, total)
	list, err := s.master.ListWarehouses(ctx, f, pageSize, offset)
	if err != nil {
		return Page[domain.Warehouse]{}, err
	}
	return Page[domain.Warehouse]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *ERPService) CreateWarehouse(ctx context.Context, w domain.Warehouse) (domain.Warehouse, error) {
	if err := s.ensureCodeFree(ctx, "erp_warehouses", w.Code, 0); err != nil {
		return domain.Warehouse{}, err
	}
	w.Status = orActive(w.Status)
	return s.master.CreateWarehouse(ctx, w)
}

func (s *ERPService) UpdateWarehouse(ctx context.Context, w domain.Warehouse) error {
	w.Status = orActive(w.Status)
	return s.master.UpdateWarehouse(ctx, w)
}

func (s *ERPService) DeleteWarehouse(ctx context.Context, id int64) error {
	n, err := s.master.WarehouseRefs(ctx, id)
	if err != nil {
		return err
	}
	if err := domain.HasDependents(map[string]int64{"references": n}); err != nil {
		return err
	}
	return s.master.DeleteWarehouse(ctx, id)
}

// ---------- 计量单位 ----------

func (s *ERPService) ListUnits(ctx context.Context, status string) ([]domain.Unit, error) {
	return s.master.ListUnits(ctx, status)
}

func (s *ERPService) CreateUnit(ctx context.Context, u domain.Unit) (domain.Unit, error) {
	u.Name = strings.TrimSpace(u.Name)
	if u.Name == "" {
		return domain.Unit{}, fmt.Errorf("单位名称不能为空: %w", domain.ErrInvalidParent)
	}
	u.Status = orActive(u.Status)
	return s.master.CreateUnit(ctx, u)
}

func (s *ERPService) UpdateUnit(ctx context.Context, u domain.Unit) error {
	u.Name = strings.TrimSpace(u.Name)
	if u.Name == "" {
		return fmt.Errorf("单位名称不能为空: %w", domain.ErrInvalidParent)
	}
	u.Status = orActive(u.Status)
	return s.master.UpdateUnit(ctx, u)
}

func (s *ERPService) DeleteUnit(ctx context.Context, id int64) error {
	n, err := s.master.UnitRefs(ctx, id)
	if err != nil {
		return err
	}
	if err := domain.HasDependents(map[string]int64{"goods": n}); err != nil {
		return err
	}
	return s.master.DeleteUnit(ctx, id)
}

// ---------- 商品 ----------

func (s *ERPService) ListGoods(ctx context.Context, f repository.MasterFilter, page, pageSize int) (Page[domain.Goods], error) {
	page, pageSize = normalizePage(page, pageSize)
	total, err := s.master.CountGoods(ctx, f)
	if err != nil {
		return Page[domain.Goods]{}, err
	}
	page, offset := clampPage(page, pageSize, total)
	list, err := s.master.ListGoods(ctx, f, pageSize, offset)
	if err != nil {
		return Page[domain.Goods]{}, err
	}
	return Page[domain.Goods]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *ERPService) CreateGoods(ctx context.Context, g domain.Goods) (domain.Goods, error) {
	if err := s.ensureCodeFree(ctx, "erp_goods", g.Code, 0); err != nil {
		return domain.Goods{}, err
	}
	g.Status = orActive(g.Status)
	if g.Unit == "" {
		g.Unit = "件"
	}
	return s.master.CreateGoods(ctx, g)
}

func (s *ERPService) UpdateGoods(ctx context.Context, g domain.Goods) error {
	g.Status = orActive(g.Status)
	return s.master.UpdateGoods(ctx, g)
}

func (s *ERPService) DeleteGoods(ctx context.Context, id int64) error {
	n, err := s.master.GoodsRefs(ctx, id)
	if err != nil {
		return err
	}
	if err := domain.HasDependents(map[string]int64{"references": n}); err != nil {
		return err
	}
	return s.master.DeleteGoods(ctx, id)
}

func (s *ERPService) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return s.master.ListCategories(ctx)
}

func (s *ERPService) CreateCategory(ctx context.Context, c domain.Category) (domain.Category, error) {
	return s.master.CreateCategory(ctx, c)
}

func (s *ERPService) UpdateCategory(ctx context.Context, c domain.Category) error {
	return s.master.UpdateCategory(ctx, c)
}

func (s *ERPService) DeleteCategory(ctx context.Context, id int64) error {
	n, err := s.master.CategoryRefs(ctx, id)
	if err != nil {
		return err
	}
	if err := domain.HasDependents(map[string]int64{"references": n}); err != nil {
		return err
	}
	return s.master.DeleteCategory(ctx, id)
}

func (s *ERPService) ListBrands(ctx context.Context) ([]domain.Brand, error) {
	return s.master.ListBrands(ctx)
}

func (s *ERPService) UpdateBrand(ctx context.Context, b domain.Brand) error {
	return s.master.UpdateBrand(ctx, b)
}

func (s *ERPService) DeleteBrand(ctx context.Context, id int64) error {
	n, err := s.master.BrandRefs(ctx, id)
	if err != nil {
		return err
	}
	if err := domain.HasDependents(map[string]int64{"references": n}); err != nil {
		return err
	}
	return s.master.DeleteBrand(ctx, id)
}

func (s *ERPService) CreateBrand(ctx context.Context, b domain.Brand) (domain.Brand, error) {
	return s.master.CreateBrand(ctx, b)
}

// ---------- 库存 ----------

func (s *ERPService) ListStocks(ctx context.Context, f repository.StockFilter, page, pageSize int) (Page[domain.Stock], error) {
	page, pageSize = normalizePage(page, pageSize)
	total, err := s.master.CountStocks(ctx, f)
	if err != nil {
		return Page[domain.Stock]{}, err
	}
	page, offset := clampPage(page, pageSize, total)
	list, err := s.master.ListStocks(ctx, f, pageSize, offset)
	if err != nil {
		return Page[domain.Stock]{}, err
	}
	return Page[domain.Stock]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *ERPService) ListStockLogs(ctx context.Context, f repository.StockFilter, page, pageSize int) (Page[domain.StockLog], error) {
	page, pageSize = normalizePage(page, pageSize)
	total, err := s.master.CountStockLogs(ctx, f)
	if err != nil {
		return Page[domain.StockLog]{}, err
	}
	page, offset := clampPage(page, pageSize, total)
	list, err := s.master.ListStockLogs(ctx, f, pageSize, offset)
	if err != nil {
		return Page[domain.StockLog]{}, err
	}
	return Page[domain.StockLog]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *ERPService) CreateStockAdjustment(ctx context.Context, warehouseID, goodsID int64, direction string, qty float64, reason string, operatorID int64, operatorName string) (domain.StockAdjustment, error) {
	if direction != "in" && direction != "out" || qty <= 0 || math.IsNaN(qty) || math.IsInf(qty, 0) {
		return domain.StockAdjustment{}, fmt.Errorf("库存调整方向或数量无效: %w", domain.ErrInvalidParent)
	}
	if ok, err := s.master.Exists(ctx, "erp_warehouses", warehouseID); err != nil {
		return domain.StockAdjustment{}, err
	} else if !ok {
		return domain.StockAdjustment{}, domain.ErrInvalidParent
	}
	if ok, err := s.master.Exists(ctx, "erp_goods", goodsID); err != nil {
		return domain.StockAdjustment{}, err
	} else if !ok {
		return domain.StockAdjustment{}, domain.ErrInvalidParent
	}
	return s.orders.CreateStockAdjustment(ctx, domain.StockAdjustment{
		WarehouseID: warehouseID, GoodsID: goodsID, Direction: direction, Qty: qty,
		Reason: reason, CreatedBy: operatorID,
	}, operatorName)
}

func (s *ERPService) GetStockAdjustment(ctx context.Context, id int64) (domain.StockAdjustment, error) {
	return s.orders.GetStockAdjustment(ctx, id)
}

func (s *ERPService) SubmitStockAdjustment(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.SubmitStockAdjustment(ctx, id, operatorID, operatorName)
}

func (s *ERPService) DeleteDraftStockAdjustment(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.DeleteDraftStockAdjustment(ctx, id, operatorID, operatorName)
}

func (s *ERPService) AuditStockAdjustment(ctx context.Context, id, operatorID int64, operatorName, comment, scope string, scopeID int64, pass bool) error {
	allowSelf := s.switches.BugEnabled(ctx, domain.BugApproveSelf, scope, scopeID)
	allowNegative := s.switches.BugEnabled(ctx, domain.BugNegativeStock, scope, scopeID)
	return s.orders.AuditStockAdjustment(ctx, id, operatorID, operatorName, comment, allowSelf, allowNegative, pass)
}

// ---------- 报表 ----------

// StockFlowReport 返回日期区间内的出入库汇总。日期按 UTC 存储，调用方传 YYYY-MM-DD。
func (s *ERPService) StockFlowReport(ctx context.Context, from, to string, warehouseID int64) ([]repository.StockFlowRow, error) {
	if from == "" || to == "" {
		return nil, fmt.Errorf("报表需要起止日期: %w", domain.ErrInvalidParent)
	}
	start, err := time.Parse("2006-01-02", from)
	if err != nil {
		return nil, fmt.Errorf("起始日期格式应为 YYYY-MM-DD: %w", domain.ErrInvalidParent)
	}
	end, err := time.Parse("2006-01-02", to)
	if err != nil {
		return nil, fmt.Errorf("结束日期格式应为 YYYY-MM-DD: %w", domain.ErrInvalidParent)
	}
	if !end.After(start) {
		return nil, fmt.Errorf("结束日期必须晚于起始日期: %w", domain.ErrInvalidParent)
	}
	return s.master.StockFlowReport(ctx, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"), warehouseID)
}

// ---------- 库存盘点 ----------

func (s *ERPService) ListStockChecks(ctx context.Context, f repository.OrderFilter, page, pageSize int) (Page[domain.StockCheck], error) {
	page, pageSize = normalizePage(page, pageSize)
	total, err := s.orders.CountStockChecks(ctx, f)
	if err != nil {
		return Page[domain.StockCheck]{}, err
	}
	page, offset := clampPage(page, pageSize, total)
	list, err := s.orders.ListStockChecks(ctx, f, pageSize, offset)
	if err != nil {
		return Page[domain.StockCheck]{}, err
	}
	return Page[domain.StockCheck]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *ERPService) GetStockCheck(ctx context.Context, id int64) (domain.StockCheck, error) {
	return s.orders.GetStockCheck(ctx, id)
}

func (s *ERPService) CreateStockCheck(ctx context.Context, c domain.StockCheck, operatorName string) (domain.StockCheck, error) {
	if c.WarehouseID <= 0 {
		return domain.StockCheck{}, fmt.Errorf("盘点仓库必填: %w", domain.ErrInvalidParent)
	}
	if ok, err := s.master.Exists(ctx, "erp_warehouses", c.WarehouseID); err != nil {
		return domain.StockCheck{}, err
	} else if !ok {
		return domain.StockCheck{}, domain.ErrInvalidParent
	}
	if len(c.Items) == 0 {
		return domain.StockCheck{}, fmt.Errorf("盘点单没有明细: %w", domain.ErrInvalidParent)
	}
	seen := make(map[int64]struct{}, len(c.Items))
	for _, it := range c.Items {
		if it.ActualQty < 0 || math.IsNaN(it.ActualQty) || math.IsInf(it.ActualQty, 0) {
			return domain.StockCheck{}, fmt.Errorf("实盘数量不能为负: %w", domain.ErrInvalidParent)
		}
		if _, dup := seen[it.GoodsID]; dup {
			return domain.StockCheck{}, fmt.Errorf("盘点单同一商品不能重复: %w", domain.ErrInvalidParent)
		}
		seen[it.GoodsID] = struct{}{}
		if ok, err := s.master.Exists(ctx, "erp_goods", it.GoodsID); err != nil {
			return domain.StockCheck{}, err
		} else if !ok {
			return domain.StockCheck{}, domain.ErrInvalidParent
		}
	}
	return s.orders.CreateStockCheck(ctx, c, operatorName)
}

func (s *ERPService) SubmitStockCheck(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.SubmitStockCheck(ctx, id, operatorID, operatorName)
}

func (s *ERPService) DeleteDraftStockCheck(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.DeleteDraftStockCheck(ctx, id, operatorID, operatorName)
}

func (s *ERPService) AuditStockCheck(ctx context.Context, id, operatorID int64, operatorName, comment, scope string, scopeID int64, pass bool) error {
	allowSelf := s.switches.BugEnabled(ctx, domain.BugApproveSelf, scope, scopeID)
	return s.orders.AuditStockCheck(ctx, id, operatorID, operatorName, comment, allowSelf, pass)
}

// ---------- 库存调拨 ----------

func (s *ERPService) ListTransfers(ctx context.Context, f repository.OrderFilter, page, pageSize int) (Page[domain.StockTransfer], error) {
	page, pageSize = normalizePage(page, pageSize)
	total, err := s.orders.CountTransfers(ctx, f)
	if err != nil {
		return Page[domain.StockTransfer]{}, err
	}
	page, offset := clampPage(page, pageSize, total)
	list, err := s.orders.ListTransfers(ctx, f, pageSize, offset)
	if err != nil {
		return Page[domain.StockTransfer]{}, err
	}
	return Page[domain.StockTransfer]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *ERPService) GetTransfer(ctx context.Context, id int64) (domain.StockTransfer, error) {
	return s.orders.GetTransfer(ctx, id)
}

func (s *ERPService) CreateTransfer(ctx context.Context, t domain.StockTransfer, operatorName string) (domain.StockTransfer, error) {
	if t.FromWarehouseID <= 0 || t.ToWarehouseID <= 0 {
		return domain.StockTransfer{}, fmt.Errorf("调出/调入仓库必填: %w", domain.ErrInvalidParent)
	}
	if t.FromWarehouseID == t.ToWarehouseID {
		return domain.StockTransfer{}, fmt.Errorf("调出与调入仓库不能相同: %w", domain.ErrInvalidParent)
	}
	for _, id := range []int64{t.FromWarehouseID, t.ToWarehouseID} {
		if ok, err := s.master.Exists(ctx, "erp_warehouses", id); err != nil {
			return domain.StockTransfer{}, err
		} else if !ok {
			return domain.StockTransfer{}, domain.ErrInvalidParent
		}
	}
	if len(t.Items) == 0 {
		return domain.StockTransfer{}, fmt.Errorf("调拨单没有明细: %w", domain.ErrInvalidParent)
	}
	for _, it := range t.Items {
		if it.Qty <= 0 || math.IsNaN(it.Qty) || math.IsInf(it.Qty, 0) {
			return domain.StockTransfer{}, fmt.Errorf("调拨数量必须为正: %w", domain.ErrInvalidParent)
		}
		if ok, err := s.master.Exists(ctx, "erp_goods", it.GoodsID); err != nil {
			return domain.StockTransfer{}, err
		} else if !ok {
			return domain.StockTransfer{}, domain.ErrInvalidParent
		}
	}
	return s.orders.CreateTransfer(ctx, t, operatorName)
}

func (s *ERPService) SubmitTransfer(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.SubmitTransfer(ctx, id, operatorID, operatorName)
}

func (s *ERPService) DeleteDraftTransfer(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.DeleteDraftTransfer(ctx, id, operatorID, operatorName)
}

func (s *ERPService) ApproveTransfer(ctx context.Context, id, operatorID int64, operatorName, comment, scope string, scopeID int64) error {
	allowSelf := s.switches.BugEnabled(ctx, domain.BugApproveSelf, scope, scopeID)
	return s.orders.AuditTransfer(ctx, id, operatorID, operatorName, comment, allowSelf, true)
}

func (s *ERPService) RejectTransfer(ctx context.Context, id, operatorID int64, operatorName, comment, scope string, scopeID int64) error {
	allowSelf := s.switches.BugEnabled(ctx, domain.BugApproveSelf, scope, scopeID)
	return s.orders.AuditTransfer(ctx, id, operatorID, operatorName, comment, allowSelf, false)
}

func (s *ERPService) CompleteTransfer(ctx context.Context, id, operatorID int64, operatorName, scope string, scopeID int64) error {
	allowNegative := s.switches.BugEnabled(ctx, domain.BugNegativeStock, scope, scopeID)
	return s.orders.CompleteTransfer(ctx, id, operatorID, operatorName, allowNegative)
}

// ---------- 采购单 ----------

func (s *ERPService) ListPurchases(ctx context.Context, f repository.OrderFilter, page, pageSize int) (Page[domain.PurchaseOrder], error) {
	page, pageSize = normalizePage(page, pageSize)
	total, err := s.orders.CountPurchases(ctx, f)
	if err != nil {
		return Page[domain.PurchaseOrder]{}, err
	}
	page, offset := clampPage(page, pageSize, total)
	list, err := s.orders.ListPurchases(ctx, f, pageSize, offset)
	if err != nil {
		return Page[domain.PurchaseOrder]{}, err
	}
	return Page[domain.PurchaseOrder]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *ERPService) GetPurchase(ctx context.Context, id int64) (domain.PurchaseOrder, error) {
	return s.orders.GetPurchase(ctx, id)
}

func (s *ERPService) CreatePurchase(ctx context.Context, o domain.PurchaseOrder, operatorName, scope string, scopeID int64) (domain.PurchaseOrder, error) {
	if err := s.validateOrderRefs(ctx, "erp_suppliers", o.SupplierID, o.WarehouseID); err != nil {
		return domain.PurchaseOrder{}, err
	}
	items, total, err := s.prepareItems(ctx, o.Items,
		s.switches.BugEnabled(ctx, domain.BugAmountError, scope, scopeID))
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	o.Items, o.TotalAmount = items, total
	return s.orders.CreatePurchase(ctx, o, operatorName)
}

func (s *ERPService) UpdatePurchase(ctx context.Context, o domain.PurchaseOrder, operatorName, scope string, scopeID int64) error {
	if err := s.validateOrderRefs(ctx, "erp_suppliers", o.SupplierID, o.WarehouseID); err != nil {
		return err
	}
	items, total, err := s.prepareItems(ctx, o.Items,
		s.switches.BugEnabled(ctx, domain.BugAmountError, scope, scopeID))
	if err != nil {
		return err
	}
	o.Items, o.TotalAmount = items, total
	allowAny := s.switches.BugEnabled(ctx, domain.BugEditCompleted, scope, scopeID)
	return s.orders.UpdatePurchase(ctx, o, operatorName, allowAny)
}

func (s *ERPService) SubmitPurchase(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.SubmitPurchase(ctx, id, operatorID, operatorName)
}

func (s *ERPService) ApprovePurchase(ctx context.Context, id, operatorID int64, operatorName, comment, scope string, scopeID int64) error {
	allowSelf := s.switches.BugEnabled(ctx, domain.BugApproveSelf, scope, scopeID)
	return s.orders.ApprovePurchase(ctx, id, operatorID, operatorName, comment, allowSelf)
}

func (s *ERPService) RejectPurchase(ctx context.Context, id, operatorID int64, operatorName, comment, scope string, scopeID int64) error {
	allowSelf := s.switches.BugEnabled(ctx, domain.BugApproveSelf, scope, scopeID)
	return s.orders.RejectPurchase(ctx, id, operatorID, operatorName, comment, allowSelf)
}

func (s *ERPService) WithdrawPurchase(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.WithdrawPurchase(ctx, id, operatorID, operatorName)
}

func (s *ERPService) DeleteDraftPurchase(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.DeleteDraftPurchase(ctx, id, operatorID, operatorName)
}

func (s *ERPService) ReceivePurchase(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.ReceivePurchase(ctx, id, operatorID, operatorName)
}

// ---------- 销售单 ----------

func (s *ERPService) ListSales(ctx context.Context, f repository.OrderFilter, page, pageSize int) (Page[domain.SalesOrder], error) {
	page, pageSize = normalizePage(page, pageSize)
	total, err := s.orders.CountSales(ctx, f)
	if err != nil {
		return Page[domain.SalesOrder]{}, err
	}
	page, offset := clampPage(page, pageSize, total)
	list, err := s.orders.ListSales(ctx, f, pageSize, offset)
	if err != nil {
		return Page[domain.SalesOrder]{}, err
	}
	return Page[domain.SalesOrder]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *ERPService) GetSales(ctx context.Context, id int64) (domain.SalesOrder, error) {
	return s.orders.GetSales(ctx, id)
}

func (s *ERPService) CreateSales(ctx context.Context, o domain.SalesOrder, operatorName, scope string, scopeID int64) (domain.SalesOrder, error) {
	if err := s.validateOrderRefs(ctx, "erp_customers", o.CustomerID, o.WarehouseID); err != nil {
		return domain.SalesOrder{}, err
	}
	items, total, err := s.prepareItems(ctx, o.Items,
		s.switches.BugEnabled(ctx, domain.BugAmountError, scope, scopeID))
	if err != nil {
		return domain.SalesOrder{}, err
	}
	o.Items, o.TotalAmount = items, total
	return s.orders.CreateSales(ctx, o, operatorName)
}

func (s *ERPService) UpdateSales(ctx context.Context, o domain.SalesOrder, operatorName, scope string, scopeID int64) error {
	if err := s.validateOrderRefs(ctx, "erp_customers", o.CustomerID, o.WarehouseID); err != nil {
		return err
	}
	items, total, err := s.prepareItems(ctx, o.Items,
		s.switches.BugEnabled(ctx, domain.BugAmountError, scope, scopeID))
	if err != nil {
		return err
	}
	o.Items, o.TotalAmount = items, total
	allowAny := s.switches.BugEnabled(ctx, domain.BugEditCompleted, scope, scopeID)
	return s.orders.UpdateSales(ctx, o, operatorName, allowAny)
}

func (s *ERPService) SubmitSales(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.SubmitSales(ctx, id, operatorID, operatorName)
}

func (s *ERPService) ApproveSales(ctx context.Context, id, operatorID int64, operatorName, comment, scope string, scopeID int64) error {
	allowSelf := s.switches.BugEnabled(ctx, domain.BugApproveSelf, scope, scopeID)
	return s.orders.ApproveSales(ctx, id, operatorID, operatorName, comment, allowSelf)
}

func (s *ERPService) RejectSales(ctx context.Context, id, operatorID int64, operatorName, comment, scope string, scopeID int64) error {
	allowSelf := s.switches.BugEnabled(ctx, domain.BugApproveSelf, scope, scopeID)
	return s.orders.RejectSales(ctx, id, operatorID, operatorName, comment, allowSelf)
}

func (s *ERPService) WithdrawSales(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.WithdrawSales(ctx, id, operatorID, operatorName)
}

func (s *ERPService) DeleteDraftSales(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.DeleteDraftSales(ctx, id, operatorID, operatorName)
}

func (s *ERPService) ShipSales(ctx context.Context, id, operatorID int64, operatorName, scope string, scopeID int64) error {
	allowNegative := s.switches.BugEnabled(ctx, domain.BugNegativeStock, scope, scopeID)
	return s.orders.ShipSales(ctx, id, operatorID, operatorName, allowNegative)
}

func (s *ERPService) ListBillOperations(ctx context.Context, billType string, billID int64) ([]domain.BillOperation, error) {
	return s.orders.ListBillOperations(ctx, billType, billID)
}

// ---------- 内部 ----------

func (s *ERPService) ensureCodeFree(ctx context.Context, table, code string, excludeID int64) error {
	exists, err := s.master.CodeExists(ctx, table, code, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return domain.Duplicate("code")
	}
	return nil
}

func (s *ERPService) validateOrderRefs(ctx context.Context, partyTable string, partyID, warehouseID int64) error {
	if ok, err := s.master.Exists(ctx, partyTable, partyID); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("%s %d: %w", partyTable, partyID, domain.ErrInvalidParent)
	}
	if ok, err := s.master.Exists(ctx, "erp_warehouses", warehouseID); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("warehouse %d: %w", warehouseID, domain.ErrInvalidParent)
	}
	return nil
}

// prepareItems 校验明细并计算金额；injectBug 为 true 时总额少 1 分（BUG-001）。
func (s *ERPService) prepareItems(ctx context.Context, items []domain.OrderItem, injectBug bool) ([]domain.OrderItem, float64, error) {
	if len(items) == 0 {
		return nil, 0, fmt.Errorf("订单至少一行明细: %w", domain.ErrInvalidParent)
	}
	out := make([]domain.OrderItem, 0, len(items))
	var total float64
	for _, it := range items {
		if it.Qty <= 0 || it.Price < 0 {
			return nil, 0, fmt.Errorf("qty/price 非法: %w", domain.ErrInvalidParent)
		}
		if ok, err := s.master.Exists(ctx, "erp_goods", it.GoodsID); err != nil {
			return nil, 0, err
		} else if !ok {
			return nil, 0, fmt.Errorf("goods %d: %w", it.GoodsID, domain.ErrInvalidParent)
		}
		amount := round2(it.Qty * it.Price)
		total += amount
		out = append(out, domain.OrderItem{
			GoodsID: it.GoodsID, Qty: it.Qty, Price: it.Price, Amount: amount,
		})
	}
	total = round2(total)
	if injectBug && total > 0 {
		total = round2(total - 0.01)
	}
	return out, total, nil
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
