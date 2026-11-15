// 有问题欢迎联系 hot_kun@hotmail.com
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"practical-training/erp-system/internal/domain"
)

// 本文件是采购/销售退货与审核中心的 SQL 层。
//
// 退货与单据共用 ERPOrderRepository：两者共享取号、库存、操作记录等辅助函数，
// 拆成独立仓储反而要把这些辅助函数导出或复制一份。

// ReturnFilter 退货单列表筛选。SourceOrderID 用于「按来源单汇总已退数量」。
type ReturnFilter struct {
	Status        string
	Keyword       string
	SourceOrderID int64
}

func (f ReturnFilter) where(alias string) (string, []any) {
	var (
		conds []string
		args  []any
	)
	if f.Status != "" {
		conds = append(conds, alias+".status = ?")
		args = append(args, f.Status)
	}
	if f.Keyword != "" {
		p := likePattern(f.Keyword)
		conds = append(conds, "("+alias+".return_no LIKE ? ESCAPE '\\\\' OR "+
			alias+".source_order_no LIKE ? ESCAPE '\\\\')")
		args = append(args, p, p)
	}
	if f.SourceOrderID > 0 {
		conds = append(conds, alias+".source_order_id = ?")
		args = append(args, f.SourceOrderID)
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// ---------- 采购退货 ----------

const purchaseReturnSelect = `
	SELECT r.id, r.created_at, r.updated_at, r.return_no, r.source_order_id, r.source_order_no,
	       r.supplier_id, s.name, r.warehouse_id, w.name, r.status, r.total_amount, r.remark,
	       r.created_by, r.submitted_at, r.approved_by, r.approved_at
	FROM erp_purchase_returns r
	JOIN erp_suppliers s ON s.id = r.supplier_id
	JOIN erp_warehouses w ON w.id = r.warehouse_id`

func scanPurchaseReturn(s interface{ Scan(...any) error }) (domain.PurchaseReturn, error) {
	var r domain.PurchaseReturn
	err := s.Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt, &r.ReturnNo, &r.SourceOrderID, &r.SourceOrderNo,
		&r.SupplierID, &r.SupplierName, &r.WarehouseID, &r.WarehouseName, &r.Status,
		&r.TotalAmount, &r.Remark, &r.CreatedBy, &r.SubmittedAt, &r.ApprovedBy, &r.ApprovedAt)
	return r, err
}

func (r *ERPOrderRepository) CountPurchaseReturns(ctx context.Context, f ReturnFilter) (int64, error) {
	where, args := f.where("r")
	var n int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM erp_purchase_returns r`+where, args...).Scan(&n)
	return n, err
}

func (r *ERPOrderRepository) ListPurchaseReturns(ctx context.Context, f ReturnFilter, limit, offset int) ([]domain.PurchaseReturn, error) {
	where, args := f.where("r")
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		purchaseReturnSelect+where+` ORDER BY r.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.PurchaseReturn, 0)
	for rows.Next() {
		item, err := scanPurchaseReturn(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func (r *ERPOrderRepository) GetPurchaseReturn(ctx context.Context, id int64) (domain.PurchaseReturn, error) {
	item, err := scanPurchaseReturn(r.db.QueryRowContext(ctx, purchaseReturnSelect+` WHERE r.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PurchaseReturn{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.PurchaseReturn{}, err
	}
	items, err := r.listReturnItems(ctx, r.db, "erp_purchase_return_items", id)
	if err != nil {
		return domain.PurchaseReturn{}, err
	}
	item.Items = items
	return item, nil
}

func (r *ERPOrderRepository) CreatePurchaseReturn(ctx context.Context, ret domain.PurchaseReturn, operatorName string) (domain.PurchaseReturn, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.PurchaseReturn{}, err
	}
	defer tx.Rollback() //nolint:errcheck

	no, err := nextBillNo(ctx, tx, "PR", time.Now())
	if err != nil {
		return domain.PurchaseReturn{}, err
	}
	res, err := tx.ExecContext(ctx, `
		INSERT INTO erp_purchase_returns
			(return_no, source_order_id, source_order_no, supplier_id, warehouse_id,
			 status, total_amount, remark, created_by)
		VALUES (?, ?, ?, ?, ?, 'draft', ?, ?, ?)`,
		no, ret.SourceOrderID, ret.SourceOrderNo, ret.SupplierID, ret.WarehouseID,
		ret.TotalAmount, ret.Remark, ret.CreatedBy)
	if err != nil {
		return domain.PurchaseReturn{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.PurchaseReturn{}, err
	}
	if err := insertReturnItems(ctx, tx, "erp_purchase_return_items", id, ret.Items); err != nil {
		return domain.PurchaseReturn{}, err
	}
	if err := recordOperation(ctx, tx, "purchase-return", id, "create", ret.CreatedBy, operatorName, ""); err != nil {
		return domain.PurchaseReturn{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.PurchaseReturn{}, err
	}
	return r.GetPurchaseReturn(ctx, id)
}

func (r *ERPOrderRepository) UpdatePurchaseReturn(ctx context.Context, ret domain.PurchaseReturn, operatorID int64, operatorName string) error {
	return r.updateReturn(ctx, "erp_purchase_returns", "erp_purchase_return_items", "purchase-return",
		ret.ID, ret.TotalAmount, ret.Remark, ret.Items, operatorID, operatorName)
}

func (r *ERPOrderRepository) DeleteDraftPurchaseReturn(ctx context.Context, id, operatorID int64, operatorName string) error {
	return r.deleteDraftReturn(ctx, "erp_purchase_returns", "erp_purchase_return_items", "purchase-return",
		id, operatorID, operatorName)
}

func (r *ERPOrderRepository) SubmitPurchaseReturn(ctx context.Context, id, operatorID int64, operatorName string) error {
	return r.submitReturn(ctx, "erp_purchase_returns", "erp_purchase_return_items", "purchase-return",
		"purchase", id, operatorID, operatorName)
}

func (r *ERPOrderRepository) ApprovePurchaseReturn(ctx context.Context, id, operatorID int64, operatorName, comment string, allowSelf bool) error {
	return r.approveReturn(ctx, "erp_purchase_returns", "purchase-return", id, operatorID, operatorName, comment, allowSelf)
}

func (r *ERPOrderRepository) RejectPurchaseReturn(ctx context.Context, id, operatorID int64, operatorName, comment string, allowSelf bool) error {
	return r.rejectReturn(ctx, "erp_purchase_returns", "purchase-return", id, operatorID, operatorName, comment, allowSelf)
}

func (r *ERPOrderRepository) WithdrawPurchaseReturn(ctx context.Context, id, operatorID int64, operatorName string) error {
	return r.transition(ctx, "erp_purchase_returns", "purchase-return", id, operatorID, operatorName,
		"submitted", "draft", "withdraw", nil, "submitted_at = NULL")
}

// CompletePurchaseReturn 退货出库：库存减少并写流水；allowNegative 为 false 时不允许扣成负数。
func (r *ERPOrderRepository) CompletePurchaseReturn(ctx context.Context, id, operatorID int64, operatorName string, allowNegative bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var (
		status      string
		warehouseID int64
		returnNo    string
	)
	if err := tx.QueryRowContext(ctx, `
		SELECT status, warehouse_id, return_no FROM erp_purchase_returns WHERE id = ? FOR UPDATE`, id).
		Scan(&status, &warehouseID, &returnNo); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	if status != "approved" {
		return domain.ErrInvalidState
	}

	items, err := r.listReturnItems(ctx, tx, "erp_purchase_return_items", id)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return domain.ErrInvalidState
	}
	for _, it := range items {
		if !allowNegative {
			var qty float64
			err := tx.QueryRowContext(ctx, `
				SELECT qty FROM erp_stocks WHERE warehouse_id = ? AND goods_id = ? FOR UPDATE`,
				warehouseID, it.GoodsID).Scan(&qty)
			if errors.Is(err, sql.ErrNoRows) {
				qty = 0
			} else if err != nil {
				return err
			}
			if qty < it.Qty {
				return fmt.Errorf("goods %d needs %v, stock %v: %w",
					it.GoodsID, it.Qty, qty, domain.ErrInsufficientStock)
			}
		}
		if err := addStock(ctx, tx, warehouseID, it.GoodsID, -it.Qty, "purchase_return", returnNo); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE erp_purchase_returns SET status = 'completed', updated_at = CURRENT_TIMESTAMP(3) WHERE id = ?`, id); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, "purchase-return", id, "complete", operatorID, operatorName, ""); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------- 销售退货 ----------

const salesReturnSelect = `
	SELECT r.id, r.created_at, r.updated_at, r.return_no, r.source_order_id, r.source_order_no,
	       r.customer_id, c.name, r.warehouse_id, w.name, r.status, r.total_amount, r.remark,
	       r.created_by, r.submitted_at, r.approved_by, r.approved_at
	FROM erp_sale_returns r
	JOIN erp_customers c ON c.id = r.customer_id
	JOIN erp_warehouses w ON w.id = r.warehouse_id`

func scanSalesReturn(s interface{ Scan(...any) error }) (domain.SalesReturn, error) {
	var r domain.SalesReturn
	err := s.Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt, &r.ReturnNo, &r.SourceOrderID, &r.SourceOrderNo,
		&r.CustomerID, &r.CustomerName, &r.WarehouseID, &r.WarehouseName, &r.Status,
		&r.TotalAmount, &r.Remark, &r.CreatedBy, &r.SubmittedAt, &r.ApprovedBy, &r.ApprovedAt)
	return r, err
}

func (r *ERPOrderRepository) CountSalesReturns(ctx context.Context, f ReturnFilter) (int64, error) {
	where, args := f.where("r")
	var n int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM erp_sale_returns r`+where, args...).Scan(&n)
	return n, err
}

func (r *ERPOrderRepository) ListSalesReturns(ctx context.Context, f ReturnFilter, limit, offset int) ([]domain.SalesReturn, error) {
	where, args := f.where("r")
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		salesReturnSelect+where+` ORDER BY r.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.SalesReturn, 0)
	for rows.Next() {
		item, err := scanSalesReturn(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func (r *ERPOrderRepository) GetSalesReturn(ctx context.Context, id int64) (domain.SalesReturn, error) {
	item, err := scanSalesReturn(r.db.QueryRowContext(ctx, salesReturnSelect+` WHERE r.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.SalesReturn{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.SalesReturn{}, err
	}
	items, err := r.listReturnItems(ctx, r.db, "erp_sale_return_items", id)
	if err != nil {
		return domain.SalesReturn{}, err
	}
	item.Items = items
	return item, nil
}

func (r *ERPOrderRepository) CreateSalesReturn(ctx context.Context, ret domain.SalesReturn, operatorName string) (domain.SalesReturn, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.SalesReturn{}, err
	}
	defer tx.Rollback() //nolint:errcheck

	no, err := nextBillNo(ctx, tx, "SR", time.Now())
	if err != nil {
		return domain.SalesReturn{}, err
	}
	res, err := tx.ExecContext(ctx, `
		INSERT INTO erp_sale_returns
			(return_no, source_order_id, source_order_no, customer_id, warehouse_id,
			 status, total_amount, remark, created_by)
		VALUES (?, ?, ?, ?, ?, 'draft', ?, ?, ?)`,
		no, ret.SourceOrderID, ret.SourceOrderNo, ret.CustomerID, ret.WarehouseID,
		ret.TotalAmount, ret.Remark, ret.CreatedBy)
	if err != nil {
		return domain.SalesReturn{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.SalesReturn{}, err
	}
	if err := insertReturnItems(ctx, tx, "erp_sale_return_items", id, ret.Items); err != nil {
		return domain.SalesReturn{}, err
	}
	if err := recordOperation(ctx, tx, "sale-return", id, "create", ret.CreatedBy, operatorName, ""); err != nil {
		return domain.SalesReturn{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.SalesReturn{}, err
	}
	return r.GetSalesReturn(ctx, id)
}

func (r *ERPOrderRepository) UpdateSalesReturn(ctx context.Context, ret domain.SalesReturn, operatorID int64, operatorName string) error {
	return r.updateReturn(ctx, "erp_sale_returns", "erp_sale_return_items", "sale-return",
		ret.ID, ret.TotalAmount, ret.Remark, ret.Items, operatorID, operatorName)
}

func (r *ERPOrderRepository) DeleteDraftSalesReturn(ctx context.Context, id, operatorID int64, operatorName string) error {
	return r.deleteDraftReturn(ctx, "erp_sale_returns", "erp_sale_return_items", "sale-return",
		id, operatorID, operatorName)
}

func (r *ERPOrderRepository) SubmitSalesReturn(ctx context.Context, id, operatorID int64, operatorName string) error {
	return r.submitReturn(ctx, "erp_sale_returns", "erp_sale_return_items", "sale-return",
		"sales", id, operatorID, operatorName)
}

func (r *ERPOrderRepository) ApproveSalesReturn(ctx context.Context, id, operatorID int64, operatorName, comment string, allowSelf bool) error {
	return r.approveReturn(ctx, "erp_sale_returns", "sale-return", id, operatorID, operatorName, comment, allowSelf)
}

func (r *ERPOrderRepository) RejectSalesReturn(ctx context.Context, id, operatorID int64, operatorName, comment string, allowSelf bool) error {
	return r.rejectReturn(ctx, "erp_sale_returns", "sale-return", id, operatorID, operatorName, comment, allowSelf)
}

func (r *ERPOrderRepository) WithdrawSalesReturn(ctx context.Context, id, operatorID int64, operatorName string) error {
	return r.transition(ctx, "erp_sale_returns", "sale-return", id, operatorID, operatorName,
		"submitted", "draft", "withdraw", nil, "submitted_at = NULL")
}

// CompleteSalesReturn 退货入库：库存增加并写流水。
func (r *ERPOrderRepository) CompleteSalesReturn(ctx context.Context, id, operatorID int64, operatorName string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var (
		status      string
		warehouseID int64
		returnNo    string
	)
	if err := tx.QueryRowContext(ctx, `
		SELECT status, warehouse_id, return_no FROM erp_sale_returns WHERE id = ? FOR UPDATE`, id).
		Scan(&status, &warehouseID, &returnNo); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	if status != "approved" {
		return domain.ErrInvalidState
	}

	items, err := r.listReturnItems(ctx, tx, "erp_sale_return_items", id)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return domain.ErrInvalidState
	}
	for _, it := range items {
		if err := addStock(ctx, tx, warehouseID, it.GoodsID, it.Qty, "sales_return", returnNo); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE erp_sale_returns SET status = 'completed', updated_at = CURRENT_TIMESTAMP(3) WHERE id = ?`, id); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, "sale-return", id, "complete", operatorID, operatorName, ""); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------- 退货共用实现 ----------

func (r *ERPOrderRepository) updateReturn(
	ctx context.Context,
	table, itemTable, billType string,
	id int64, total float64, remark string, items []domain.OrderItem,
	operatorID int64, operatorName string,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var status string
	if err := tx.QueryRowContext(ctx,
		`SELECT status FROM `+table+` WHERE id = ? FOR UPDATE`, id).Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	if status != "draft" {
		return domain.ErrInvalidState
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE `+table+` SET total_amount = ?, remark = ?, updated_at = CURRENT_TIMESTAMP(3) WHERE id = ?`,
		total, remark, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM `+itemTable+` WHERE return_id = ?`, id); err != nil {
		return err
	}
	if err := insertReturnItems(ctx, tx, itemTable, id, items); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, billType, id, "update", operatorID, operatorName, ""); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ERPOrderRepository) deleteDraftReturn(
	ctx context.Context,
	table, itemTable, billType string,
	id, operatorID int64, operatorName string,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var status string
	if err := tx.QueryRowContext(ctx,
		`SELECT status FROM `+table+` WHERE id = ? FOR UPDATE`, id).Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	if status != "draft" {
		return domain.ErrInvalidState
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM `+itemTable+` WHERE return_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE id = ?`, id); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, billType, id, "delete", operatorID, operatorName, ""); err != nil {
		return err
	}
	return tx.Commit()
}

// submitReturn 提交退货：在事务内重新计算可退数量，防止两张草稿先后提交造成超退。
func (r *ERPOrderRepository) submitReturn(
	ctx context.Context,
	table, itemTable, billType, kind string,
	id, operatorID int64, operatorName string,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var (
		status        string
		sourceOrderID int64
	)
	if err := tx.QueryRowContext(ctx,
		`SELECT status, source_order_id FROM `+table+` WHERE id = ? FOR UPDATE`, id).
		Scan(&status, &sourceOrderID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	if status != "draft" {
		return domain.ErrInvalidState
	}
	// 锁住来源单，串行化同一来源单的多次提交；否则两张草稿会各自读到对方
	// 尚未提交，双双通过可退数量校验。
	if err := lockSourceOrder(ctx, tx, kind, sourceOrderID); err != nil {
		return err
	}
	items, err := r.listReturnItems(ctx, tx, itemTable, id)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return domain.ErrInvalidState
	}
	if err := validateReturnQty(ctx, tx, kind, sourceOrderID, id, items); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE `+table+`
		SET status = 'submitted', submitted_at = CURRENT_TIMESTAMP(3), updated_at = CURRENT_TIMESTAMP(3)
		WHERE id = ?`, id); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, billType, id, "submit", operatorID, operatorName, ""); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ERPOrderRepository) approveReturn(
	ctx context.Context,
	table, billType string,
	id, operatorID int64, operatorName, comment string, allowSelf bool,
) error {
	return r.auditReturn(ctx, table, billType, id, operatorID, operatorName, comment, allowSelf, true)
}

func (r *ERPOrderRepository) rejectReturn(
	ctx context.Context,
	table, billType string,
	id, operatorID int64, operatorName, comment string, allowSelf bool,
) error {
	return r.auditReturn(ctx, table, billType, id, operatorID, operatorName, comment, allowSelf, false)
}

// auditReturn 是退货单审核的共用实现；pass 决定 approved 还是 rejected。
func (r *ERPOrderRepository) auditReturn(
	ctx context.Context,
	table, billType string,
	id, operatorID int64, operatorName, comment string,
	allowSelf, pass bool,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var (
		status    string
		createdBy int64
	)
	if err := tx.QueryRowContext(ctx,
		`SELECT status, created_by FROM `+table+` WHERE id = ? FOR UPDATE`, id).
		Scan(&status, &createdBy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	if status != "submitted" {
		return domain.ErrInvalidState
	}
	if createdBy == operatorID && !allowSelf {
		return fmt.Errorf("cannot audit own bill: %w", domain.ErrForbidden)
	}

	next, action := "approved", "approve"
	if !pass {
		next, action = "rejected", "reject"
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE `+table+`
		SET status = ?, approved_by = ?, approved_at = CURRENT_TIMESTAMP(3),
		    updated_at = CURRENT_TIMESTAMP(3)
		WHERE id = ?`, next, operatorID, id); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, billType, id, action, operatorID, operatorName, comment); err != nil {
		return err
	}
	return tx.Commit()
}

// lockSourceOrder 在提交事务内锁住来源单，作为同一来源单退货提交的串行化点。
func lockSourceOrder(ctx context.Context, tx *sql.Tx, kind string, sourceOrderID int64) error {
	table := "erp_purchase_orders"
	if kind == "sales" {
		table = "erp_sales_orders"
	}
	var id int64
	if err := tx.QueryRowContext(ctx,
		`SELECT id FROM `+table+` WHERE id = ? FOR UPDATE`, sourceOrderID).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrInvalidParent
		}
		return err
	}
	return nil
}

// ValidateReturnQty 供建单/改单时做前置校验；提交时会在事务内再校验一次。
func (r *ERPOrderRepository) ValidateReturnQty(ctx context.Context, kind string, sourceOrderID, excludeReturnID int64, items []domain.OrderItem) error {
	return validateReturnQty(ctx, r.db, kind, sourceOrderID, excludeReturnID, items)
}

// validateReturnQty 校验「本次退货量 ≤ 来源单数量 - 其它未驳回退货量」。
//
// kind: purchase（来源 erp_purchase_orders）或 sales（来源 erp_sales_orders）。
// 计入已退的状态是 submitted/approved/completed：草稿不占额度，驳回释放额度。
func validateReturnQty(
	ctx context.Context,
	q queryer,
	kind string,
	sourceOrderID, excludeReturnID int64,
	items []domain.OrderItem,
) error {
	var (
		sourceItemsTable string
		returnsTable     string
		itemsTable       string
	)
	switch kind {
	case "purchase":
		sourceItemsTable = "erp_purchase_order_items"
		returnsTable = "erp_purchase_returns"
		itemsTable = "erp_purchase_return_items"
	case "sales":
		sourceItemsTable = "erp_sales_order_items"
		returnsTable = "erp_sale_returns"
		itemsTable = "erp_sale_return_items"
	default:
		return fmt.Errorf("unknown return kind %q: %w", kind, domain.ErrInvalidParent)
	}

	// 来源单数量
	sourceQty := map[int64]float64{}
	rows, err := q.QueryContext(ctx,
		`SELECT goods_id, qty FROM `+sourceItemsTable+` WHERE order_id = ?`, sourceOrderID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var goodsID int64
		var qty float64
		if err := rows.Scan(&goodsID, &qty); err != nil {
			rows.Close()
			return err
		}
		sourceQty[goodsID] += qty
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if len(sourceQty) == 0 {
		return fmt.Errorf("source bill %d has no items: %w", sourceOrderID, domain.ErrInvalidState)
	}

	// 其它未驳回退货已占用的数量
	claimed := map[int64]float64{}
	rows, err = q.QueryContext(ctx, `
		SELECT i.goods_id, COALESCE(SUM(i.qty), 0)
		FROM `+itemsTable+` i
		JOIN `+returnsTable+` r ON r.id = i.return_id
		WHERE r.source_order_id = ? AND r.status IN ('submitted','approved','completed') AND r.id <> ?
		GROUP BY i.goods_id`, sourceOrderID, excludeReturnID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var goodsID int64
		var qty float64
		if err := rows.Scan(&goodsID, &qty); err != nil {
			rows.Close()
			return err
		}
		claimed[goodsID] = qty
	}
	if err := rows.Close(); err != nil {
		return err
	}

	for _, it := range items {
		total, ok := sourceQty[it.GoodsID]
		if !ok {
			return fmt.Errorf("goods %d not in source bill %d: %w",
				it.GoodsID, sourceOrderID, domain.ErrReturnQtyExceeded)
		}
		if claimed[it.GoodsID]+it.Qty > total+0.0001 {
			return fmt.Errorf("goods %d return %v + claimed %v > source %v: %w",
				it.GoodsID, it.Qty, claimed[it.GoodsID], total, domain.ErrReturnQtyExceeded)
		}
	}
	return nil
}

func insertReturnItems(ctx context.Context, tx *sql.Tx, table string, returnID int64, items []domain.OrderItem) error {
	if len(items) == 0 {
		return fmt.Errorf("return has no items: %w", domain.ErrInvalidState)
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO `+table+` (return_id, goods_id, qty, price, amount) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close() //nolint:errcheck
	for _, it := range items {
		if _, err := stmt.ExecContext(ctx, returnID, it.GoodsID, it.Qty, it.Price, it.Amount); err != nil {
			return err
		}
	}
	return nil
}

func (r *ERPOrderRepository) listReturnItems(ctx context.Context, q queryer, table string, returnID int64) ([]domain.OrderItem, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT i.id, i.goods_id, g.name, i.qty, i.price, i.amount
		FROM `+table+` i JOIN erp_goods g ON g.id = i.goods_id
		WHERE i.return_id = ? ORDER BY i.id`, returnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.OrderItem, 0)
	for rows.Next() {
		var it domain.OrderItem
		if err := rows.Scan(&it.ID, &it.GoodsID, &it.GoodsName, &it.Qty, &it.Price, &it.Amount); err != nil {
			return nil, err
		}
		list = append(list, it)
	}
	return list, rows.Err()
}

// ---------- 审核中心 ----------

// AuditFilter 审核中心筛选。direction 由调用方校验。
type AuditFilter struct {
	Status  string
	Keyword string
}

func (f AuditFilter) where() (string, []any) {
	var (
		conds []string
		args  []any
	)
	if f.Status != "" {
		conds = append(conds, "u.status = ?")
		args = append(args, f.Status)
	}
	if f.Keyword != "" {
		p := likePattern(f.Keyword)
		conds = append(conds, "(u.bill_no LIKE ? ESCAPE '\\\\' OR u.partner_name LIKE ? ESCAPE '\\\\')")
		args = append(args, p, p)
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// auditUnion 返回两个单据类型拼成的统一视图。
//
//   - in : 采购单（入库待办）+ 销售退货（退货入库待办）
//   - out: 销售单（出库待办）+ 采购退货（退货出库待办）
//
// 用 UNION ALL 而不是在 Go 里合并两个列表：分页、排序、筛选都在库里做完，
// 否则跨类型分页会算错。
func auditUnion(direction string) (string, error) {
	checkNetPredicate := ""
	switch direction {
	case "in":
		// 盘点有盈有亏时按净差异归队列；净差为 0 的盘点归入库待办，避免双重展示。
		checkNetPredicate = "COALESCE((SELECT SUM(i.diff_qty) FROM erp_stock_check_items i WHERE i.check_id = c.id), 0) >= 0"
	case "out":
		checkNetPredicate = "COALESCE((SELECT SUM(i.diff_qty) FROM erp_stock_check_items i WHERE i.check_id = c.id), 0) < 0"
	default:
		return "", fmt.Errorf("direction %q: %w", direction, domain.ErrInvalidParent)
	}

	purchaseOrders := `SELECT 'purchase' AS bill_type, o.id AS bill_id, o.order_no AS bill_no,
			s.name AS partner_name, w.name AS warehouse_name, o.total_amount, o.status,
			o.created_by, COALESCE(NULLIF(cu.nickname, ''), cu.username, '') AS creator_name,
			o.submitted_at, o.approved_by,
			COALESCE(NULLIF(au.nickname, ''), au.username, '') AS approver_name,
			o.approved_at, o.created_at, '' AS direction, 0 AS qty
		FROM erp_purchase_orders o
		JOIN erp_suppliers s ON s.id = o.supplier_id
		JOIN erp_warehouses w ON w.id = o.warehouse_id
		LEFT JOIN sys_users cu ON cu.id = o.created_by
		LEFT JOIN sys_users au ON au.id = o.approved_by`
	salesOrders := `SELECT 'sales' AS bill_type, o.id AS bill_id, o.order_no AS bill_no,
			c.name AS partner_name, w.name AS warehouse_name, o.total_amount, o.status,
			o.created_by, COALESCE(NULLIF(cu.nickname, ''), cu.username, '') AS creator_name,
			o.submitted_at, o.approved_by,
			COALESCE(NULLIF(au.nickname, ''), au.username, '') AS approver_name,
			o.approved_at, o.created_at, '' AS direction, 0 AS qty
		FROM erp_sales_orders o
		JOIN erp_customers c ON c.id = o.customer_id
		JOIN erp_warehouses w ON w.id = o.warehouse_id
		LEFT JOIN sys_users cu ON cu.id = o.created_by
		LEFT JOIN sys_users au ON au.id = o.approved_by`
	purchaseReturns := `SELECT 'purchase-return' AS bill_type, r.id AS bill_id, r.return_no AS bill_no,
			s.name AS partner_name, w.name AS warehouse_name, r.total_amount, r.status,
			r.created_by, COALESCE(NULLIF(cu.nickname, ''), cu.username, '') AS creator_name,
			r.submitted_at, r.approved_by,
			COALESCE(NULLIF(au.nickname, ''), au.username, '') AS approver_name,
			r.approved_at, r.created_at, '' AS direction, 0 AS qty
		FROM erp_purchase_returns r
		JOIN erp_suppliers s ON s.id = r.supplier_id
		JOIN erp_warehouses w ON w.id = r.warehouse_id
		LEFT JOIN sys_users cu ON cu.id = r.created_by
		LEFT JOIN sys_users au ON au.id = r.approved_by`
	salesReturns := `SELECT 'sale-return' AS bill_type, r.id AS bill_id, r.return_no AS bill_no,
			c.name AS partner_name, w.name AS warehouse_name, r.total_amount, r.status,
			r.created_by, COALESCE(NULLIF(cu.nickname, ''), cu.username, '') AS creator_name,
			r.submitted_at, r.approved_by,
			COALESCE(NULLIF(au.nickname, ''), au.username, '') AS approver_name,
			r.approved_at, r.created_at, '' AS direction, 0 AS qty
		FROM erp_sale_returns r
		JOIN erp_customers c ON c.id = r.customer_id
		JOIN erp_warehouses w ON w.id = r.warehouse_id
		LEFT JOIN sys_users cu ON cu.id = r.created_by
		LEFT JOIN sys_users au ON au.id = r.approved_by`

	stockAdjustments := `SELECT 'stock-adjustment' AS bill_type, a.id AS bill_id, a.adjustment_no AS bill_no,
			g.name AS partner_name, w.name AS warehouse_name, 0 AS total_amount, a.status,
			a.created_by, COALESCE(NULLIF(cu.nickname, ''), cu.username, '') AS creator_name,
			a.submitted_at, a.approved_by, COALESCE(NULLIF(au.nickname, ''), au.username, '') AS approver_name,
			a.approved_at, a.created_at, a.direction, a.qty
		FROM erp_stock_adjustments a
		JOIN erp_warehouses w ON w.id = a.warehouse_id
		JOIN erp_goods g ON g.id = a.goods_id
		LEFT JOIN sys_users cu ON cu.id = a.created_by
		LEFT JOIN sys_users au ON au.id = a.approved_by
		WHERE a.direction = '` + direction + `'`
	transfers := `SELECT 'transfer' AS bill_type, t.id AS bill_id, t.transfer_no AS bill_no,
			CONCAT(fw.name, ' → ', tw.name) AS partner_name, tw.name AS warehouse_name, 0 AS total_amount, t.status,
			t.created_by, COALESCE(NULLIF(cu.nickname, ''), cu.username, '') AS creator_name,
			t.submitted_at, t.approved_by, COALESCE(NULLIF(au.nickname, ''), au.username, '') AS approver_name,
			t.approved_at, t.created_at, '' AS direction, 0 AS qty
		FROM erp_transfers t
		JOIN erp_warehouses fw ON fw.id = t.from_warehouse_id
		JOIN erp_warehouses tw ON tw.id = t.to_warehouse_id
		LEFT JOIN sys_users cu ON cu.id = t.created_by
		LEFT JOIN sys_users au ON au.id = t.approved_by`
	stockChecks := `SELECT 'stock-check' AS bill_type, c.id AS bill_id, c.check_no AS bill_no,
			w.name AS partner_name, w.name AS warehouse_name, 0 AS total_amount, c.status,
			c.created_by, COALESCE(NULLIF(cu.nickname, ''), cu.username, '') AS creator_name,
			c.submitted_at, c.approved_by, COALESCE(NULLIF(au.nickname, ''), au.username, '') AS approver_name,
			c.approved_at, c.created_at, '` + direction + `' AS direction, 0 AS qty
		FROM erp_stock_checks c
		JOIN erp_warehouses w ON w.id = c.warehouse_id
		JOIN sys_users cu ON cu.id = c.created_by
		LEFT JOIN sys_users au ON au.id = c.approved_by
		WHERE c.status = 'submitted' AND ` + checkNetPredicate
	switch direction {
	case "in":
		// 调拨与盘点差异各自按出/入方向进入对应待办，避免同一单重复出现两次。
		return purchaseOrders + ` UNION ALL ` + salesReturns + ` UNION ALL ` + stockAdjustments + ` UNION ALL ` + transfers + ` UNION ALL ` + stockChecks, nil
	case "out":
		return salesOrders + ` UNION ALL ` + purchaseReturns + ` UNION ALL ` + stockAdjustments + ` UNION ALL ` + transfers + ` UNION ALL ` + stockChecks, nil
	}
	return "", fmt.Errorf("direction %q: %w", direction, domain.ErrInvalidParent)
}

func (r *ERPOrderRepository) CountAuditBills(ctx context.Context, direction string, f AuditFilter) (int64, error) {
	union, err := auditUnion(direction)
	if err != nil {
		return 0, err
	}
	where, args := f.where()
	var n int64
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM (`+union+`) u`+where, args...).Scan(&n)
	return n, err
}

func (r *ERPOrderRepository) ListAuditBills(ctx context.Context, direction string, f AuditFilter, limit, offset int) ([]domain.AuditBill, error) {
	union, err := auditUnion(direction)
	if err != nil {
		return nil, err
	}
	where, args := f.where()
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.bill_type, u.bill_id, u.bill_no, u.partner_name, u.warehouse_name,
		       u.total_amount, u.status, u.created_by, u.creator_name, u.submitted_at,
		       u.approved_by, u.approver_name, u.approved_at, u.direction, u.qty
		FROM (`+union+`) u`+where+` ORDER BY u.created_at DESC, u.bill_id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.AuditBill, 0)
	for rows.Next() {
		var b domain.AuditBill
		if err := rows.Scan(&b.BillType, &b.BillID, &b.BillNo, &b.PartnerName, &b.WarehouseName,
			&b.TotalAmount, &b.Status, &b.CreatedBy, &b.CreatorName, &b.SubmittedAt,
			&b.ApprovedBy, &b.ApproverName, &b.ApprovedAt, &b.Direction, &b.Qty); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}
