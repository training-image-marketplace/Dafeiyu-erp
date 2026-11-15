// 有问题欢迎联系 hot_kun@hotmail.com
package service

import (
	"context"
	"fmt"
	"strings"

	"practical-training/erp-system/internal/domain"
	"practical-training/erp-system/internal/repository"
)

// ---------- 采购退货 ----------

func (s *ERPService) ListPurchaseReturns(ctx context.Context, f repository.ReturnFilter, page, pageSize int) (Page[domain.PurchaseReturn], error) {
	page, pageSize = normalizePage(page, pageSize)
	total, err := s.orders.CountPurchaseReturns(ctx, f)
	if err != nil {
		return Page[domain.PurchaseReturn]{}, err
	}
	page, offset := clampPage(page, pageSize, total)
	list, err := s.orders.ListPurchaseReturns(ctx, f, pageSize, offset)
	if err != nil {
		return Page[domain.PurchaseReturn]{}, err
	}
	return Page[domain.PurchaseReturn]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *ERPService) GetPurchaseReturn(ctx context.Context, id int64) (domain.PurchaseReturn, error) {
	return s.orders.GetPurchaseReturn(ctx, id)
}

// CreatePurchaseReturn 从已完成的采购入库单发起退货。
//
// 单价一律取来源单明细，不接受前端传入：退货金额必须能追溯到原采购价，
// 否则「按当前售价退」会凭空产生差价。
func (s *ERPService) CreatePurchaseReturn(ctx context.Context, sourceOrderID int64, req []domain.OrderItem, remark string, operatorID int64, operatorName string) (domain.PurchaseReturn, error) {
	source, err := s.orders.GetPurchase(ctx, sourceOrderID)
	if err != nil {
		return domain.PurchaseReturn{}, err
	}
	if source.Status != "completed" {
		return domain.PurchaseReturn{}, fmt.Errorf("source purchase %d is %s: %w", sourceOrderID, source.Status, domain.ErrInvalidState)
	}
	items, total, err := buildReturnItems(source.OrderNo, source.Items, req)
	if err != nil {
		return domain.PurchaseReturn{}, err
	}
	if err := s.orders.ValidateReturnQty(ctx, "purchase", source.ID, 0, items); err != nil {
		return domain.PurchaseReturn{}, err
	}
	ret := domain.PurchaseReturn{
		SourceOrderID: source.ID,
		SourceOrderNo: source.OrderNo,
		SupplierID:    source.SupplierID,
		WarehouseID:   source.WarehouseID,
		TotalAmount:   total,
		Remark:        strings.TrimSpace(remark),
		CreatedBy:     operatorID,
		Items:         items,
	}
	return s.orders.CreatePurchaseReturn(ctx, ret, operatorName)
}

func (s *ERPService) UpdatePurchaseReturn(ctx context.Context, id int64, req []domain.OrderItem, remark string, operatorID int64, operatorName string) error {
	existing, err := s.orders.GetPurchaseReturn(ctx, id)
	if err != nil {
		return err
	}
	if existing.Status != "draft" {
		return domain.ErrInvalidState
	}
	source, err := s.orders.GetPurchase(ctx, existing.SourceOrderID)
	if err != nil {
		return err
	}
	items, total, err := buildReturnItems(source.OrderNo, source.Items, req)
	if err != nil {
		return err
	}
	if err := s.orders.ValidateReturnQty(ctx, "purchase", source.ID, existing.ID, items); err != nil {
		return err
	}
	existing.TotalAmount = total
	existing.Remark = strings.TrimSpace(remark)
	existing.Items = items
	return s.orders.UpdatePurchaseReturn(ctx, existing, operatorID, operatorName)
}

func (s *ERPService) SubmitPurchaseReturn(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.SubmitPurchaseReturn(ctx, id, operatorID, operatorName)
}

func (s *ERPService) ApprovePurchaseReturn(ctx context.Context, id, operatorID int64, operatorName, comment, scope string, scopeID int64) error {
	allowSelf := s.switches.BugEnabled(ctx, domain.BugApproveSelf, scope, scopeID)
	return s.orders.ApprovePurchaseReturn(ctx, id, operatorID, operatorName, comment, allowSelf)
}

func (s *ERPService) RejectPurchaseReturn(ctx context.Context, id, operatorID int64, operatorName, comment, scope string, scopeID int64) error {
	allowSelf := s.switches.BugEnabled(ctx, domain.BugApproveSelf, scope, scopeID)
	return s.orders.RejectPurchaseReturn(ctx, id, operatorID, operatorName, comment, allowSelf)
}

func (s *ERPService) WithdrawPurchaseReturn(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.WithdrawPurchaseReturn(ctx, id, operatorID, operatorName)
}

func (s *ERPService) DeleteDraftPurchaseReturn(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.DeleteDraftPurchaseReturn(ctx, id, operatorID, operatorName)
}

func (s *ERPService) CompletePurchaseReturn(ctx context.Context, id, operatorID int64, operatorName, scope string, scopeID int64) error {
	allowNegative := s.switches.BugEnabled(ctx, domain.BugNegativeStock, scope, scopeID)
	return s.orders.CompletePurchaseReturn(ctx, id, operatorID, operatorName, allowNegative)
}

// ---------- 销售退货 ----------

func (s *ERPService) ListSalesReturns(ctx context.Context, f repository.ReturnFilter, page, pageSize int) (Page[domain.SalesReturn], error) {
	page, pageSize = normalizePage(page, pageSize)
	total, err := s.orders.CountSalesReturns(ctx, f)
	if err != nil {
		return Page[domain.SalesReturn]{}, err
	}
	page, offset := clampPage(page, pageSize, total)
	list, err := s.orders.ListSalesReturns(ctx, f, pageSize, offset)
	if err != nil {
		return Page[domain.SalesReturn]{}, err
	}
	return Page[domain.SalesReturn]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *ERPService) GetSalesReturn(ctx context.Context, id int64) (domain.SalesReturn, error) {
	return s.orders.GetSalesReturn(ctx, id)
}

func (s *ERPService) CreateSalesReturn(ctx context.Context, sourceOrderID int64, req []domain.OrderItem, remark string, operatorID int64, operatorName string) (domain.SalesReturn, error) {
	source, err := s.orders.GetSales(ctx, sourceOrderID)
	if err != nil {
		return domain.SalesReturn{}, err
	}
	if source.Status != "completed" {
		return domain.SalesReturn{}, fmt.Errorf("source sales %d is %s: %w", sourceOrderID, source.Status, domain.ErrInvalidState)
	}
	items, total, err := buildReturnItems(source.OrderNo, source.Items, req)
	if err != nil {
		return domain.SalesReturn{}, err
	}
	if err := s.orders.ValidateReturnQty(ctx, "sales", source.ID, 0, items); err != nil {
		return domain.SalesReturn{}, err
	}
	ret := domain.SalesReturn{
		SourceOrderID: source.ID,
		SourceOrderNo: source.OrderNo,
		CustomerID:    source.CustomerID,
		WarehouseID:   source.WarehouseID,
		TotalAmount:   total,
		Remark:        strings.TrimSpace(remark),
		CreatedBy:     operatorID,
		Items:         items,
	}
	return s.orders.CreateSalesReturn(ctx, ret, operatorName)
}

func (s *ERPService) UpdateSalesReturn(ctx context.Context, id int64, req []domain.OrderItem, remark string, operatorID int64, operatorName string) error {
	existing, err := s.orders.GetSalesReturn(ctx, id)
	if err != nil {
		return err
	}
	if existing.Status != "draft" {
		return domain.ErrInvalidState
	}
	source, err := s.orders.GetSales(ctx, existing.SourceOrderID)
	if err != nil {
		return err
	}
	items, total, err := buildReturnItems(source.OrderNo, source.Items, req)
	if err != nil {
		return err
	}
	if err := s.orders.ValidateReturnQty(ctx, "sales", source.ID, existing.ID, items); err != nil {
		return err
	}
	existing.TotalAmount = total
	existing.Remark = strings.TrimSpace(remark)
	existing.Items = items
	return s.orders.UpdateSalesReturn(ctx, existing, operatorID, operatorName)
}

func (s *ERPService) SubmitSalesReturn(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.SubmitSalesReturn(ctx, id, operatorID, operatorName)
}

func (s *ERPService) ApproveSalesReturn(ctx context.Context, id, operatorID int64, operatorName, comment, scope string, scopeID int64) error {
	allowSelf := s.switches.BugEnabled(ctx, domain.BugApproveSelf, scope, scopeID)
	return s.orders.ApproveSalesReturn(ctx, id, operatorID, operatorName, comment, allowSelf)
}

func (s *ERPService) RejectSalesReturn(ctx context.Context, id, operatorID int64, operatorName, comment, scope string, scopeID int64) error {
	allowSelf := s.switches.BugEnabled(ctx, domain.BugApproveSelf, scope, scopeID)
	return s.orders.RejectSalesReturn(ctx, id, operatorID, operatorName, comment, allowSelf)
}

func (s *ERPService) WithdrawSalesReturn(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.WithdrawSalesReturn(ctx, id, operatorID, operatorName)
}

func (s *ERPService) DeleteDraftSalesReturn(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.DeleteDraftSalesReturn(ctx, id, operatorID, operatorName)
}

func (s *ERPService) CompleteSalesReturn(ctx context.Context, id, operatorID int64, operatorName string) error {
	return s.orders.CompleteSalesReturn(ctx, id, operatorID, operatorName)
}

// ---------- 审核中心 ----------

func (s *ERPService) ListAuditBills(ctx context.Context, direction string, f repository.AuditFilter, page, pageSize int) (Page[domain.AuditBill], error) {
	if direction != "in" && direction != "out" {
		return Page[domain.AuditBill]{}, fmt.Errorf("direction %q: %w", direction, domain.ErrInvalidParent)
	}
	page, pageSize = normalizePage(page, pageSize)
	total, err := s.orders.CountAuditBills(ctx, direction, f)
	if err != nil {
		return Page[domain.AuditBill]{}, err
	}
	page, offset := clampPage(page, pageSize, total)
	list, err := s.orders.ListAuditBills(ctx, direction, f, pageSize, offset)
	if err != nil {
		return Page[domain.AuditBill]{}, err
	}
	return Page[domain.AuditBill]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

// AuditBill 是审核中心唯一的动作入口：通过或驳回，按 billType 分发到对应单据。
//
// 复用各单据自己的审核实现（含「不能审自己的单」与 BUG-003 开关），
// 审核中心只负责路由，不复制规则。
func (s *ERPService) AuditBill(ctx context.Context, billType string, id, operatorID int64, operatorName, comment, scope string, scopeID int64, pass bool) error {
	switch billType {
	case "purchase":
		if pass {
			return s.ApprovePurchase(ctx, id, operatorID, operatorName, comment, scope, scopeID)
		}
		return s.RejectPurchase(ctx, id, operatorID, operatorName, comment, scope, scopeID)
	case "sales":
		if pass {
			return s.ApproveSales(ctx, id, operatorID, operatorName, comment, scope, scopeID)
		}
		return s.RejectSales(ctx, id, operatorID, operatorName, comment, scope, scopeID)
	case "purchase-return":
		if pass {
			return s.ApprovePurchaseReturn(ctx, id, operatorID, operatorName, comment, scope, scopeID)
		}
		return s.RejectPurchaseReturn(ctx, id, operatorID, operatorName, comment, scope, scopeID)
	case "sale-return":
		if pass {
			return s.ApproveSalesReturn(ctx, id, operatorID, operatorName, comment, scope, scopeID)
		}
		return s.RejectSalesReturn(ctx, id, operatorID, operatorName, comment, scope, scopeID)
	case "stock-adjustment":
		return s.AuditStockAdjustment(ctx, id, operatorID, operatorName, comment, scope, scopeID, pass)
	case "stock-check":
		return s.AuditStockCheck(ctx, id, operatorID, operatorName, comment, scope, scopeID, pass)
	case "transfer":
		if pass {
			return s.ApproveTransfer(ctx, id, operatorID, operatorName, comment, scope, scopeID)
		}
		return s.RejectTransfer(ctx, id, operatorID, operatorName, comment, scope, scopeID)
	default:
		return fmt.Errorf("unknown bill type %q: %w", billType, domain.ErrInvalidParent)
	}
}

// buildReturnItems 把「只填商品与数量」的请求展开成完整明细。
//
// 同一商品出现多行时合并数量，避免前端重复行导致校验与显示不一致。
func buildReturnItems(sourceOrderNo string, source, req []domain.OrderItem) ([]domain.OrderItem, float64, error) {
	if len(req) == 0 {
		return nil, 0, fmt.Errorf("return has no items: %w", domain.ErrInvalidParent)
	}
	type sourceLine struct {
		price float64
		name  string
	}
	priceByGoods := make(map[int64]sourceLine, len(source))
	for _, it := range source {
		priceByGoods[it.GoodsID] = sourceLine{price: it.Price, name: it.GoodsName}
	}

	var order []int64
	qtyByGoods := make(map[int64]float64, len(req))
	for _, it := range req {
		if it.Qty <= 0 {
			return nil, 0, fmt.Errorf("qty %v: %w", it.Qty, domain.ErrInvalidParent)
		}
		if _, ok := priceByGoods[it.GoodsID]; !ok {
			return nil, 0, fmt.Errorf("goods %d not in %s: %w", it.GoodsID, sourceOrderNo, domain.ErrInvalidParent)
		}
		if _, seen := qtyByGoods[it.GoodsID]; !seen {
			order = append(order, it.GoodsID)
		}
		qtyByGoods[it.GoodsID] += it.Qty
	}

	out := make([]domain.OrderItem, 0, len(order))
	var total float64
	for _, goodsID := range order {
		src := priceByGoods[goodsID]
		qty := qtyByGoods[goodsID]
		amount := round2(qty * src.price)
		total += amount
		out = append(out, domain.OrderItem{
			GoodsID: goodsID, GoodsName: src.name, Qty: qty, Price: src.price, Amount: amount,
		})
	}
	return out, round2(total), nil
}
