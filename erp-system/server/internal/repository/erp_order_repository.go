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

// ERPOrderRepository 负责采购单、销售单、库存扣减与单据操作记录。
type ERPOrderRepository struct {
	db *sql.DB
}

func NewERPOrderRepository(db *sql.DB) *ERPOrderRepository {
	return &ERPOrderRepository{db: db}
}

type OrderFilter struct {
	Status  string
	Keyword string
}

func (f OrderFilter) where(alias string) (string, []any) {
	return f.whereColumn(alias, "order_no")
}

// whereColumn 让盘点/调拨等单据用各自的单号列过滤。
func (f OrderFilter) whereColumn(alias, noColumn string) (string, []any) {
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
		conds = append(conds, alias+"."+noColumn+" LIKE ? ESCAPE '\\\\'")
		args = append(args, p)
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

const stockAdjustmentSelect = `
	SELECT a.id, a.created_at, a.updated_at, a.adjustment_no, a.warehouse_id, w.name,
	       a.goods_id, g.name, g.unit, a.direction, a.qty, a.reason, a.status,
	       a.created_by, a.submitted_at, a.approved_by, a.approved_at
	FROM erp_stock_adjustments a
	JOIN erp_warehouses w ON w.id = a.warehouse_id
	JOIN erp_goods g ON g.id = a.goods_id`

func scanStockAdjustment(s interface{ Scan(...any) error }) (domain.StockAdjustment, error) {
	var a domain.StockAdjustment
	err := s.Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt, &a.AdjustmentNo, &a.WarehouseID, &a.WarehouseName,
		&a.GoodsID, &a.GoodsName, &a.Unit, &a.Direction, &a.Qty, &a.Reason, &a.Status,
		&a.CreatedBy, &a.SubmittedAt, &a.ApprovedBy, &a.ApprovedAt)
	return a, err
}

func (r *ERPOrderRepository) GetStockAdjustment(ctx context.Context, id int64) (domain.StockAdjustment, error) {
	a, err := scanStockAdjustment(r.db.QueryRowContext(ctx, stockAdjustmentSelect+` WHERE a.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.StockAdjustment{}, domain.ErrNotFound
	}
	return a, err
}

func (r *ERPOrderRepository) CreateStockAdjustment(ctx context.Context, a domain.StockAdjustment, operatorName string) (domain.StockAdjustment, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.StockAdjustment{}, err
	}
	defer tx.Rollback() //nolint:errcheck

	no, err := nextBillNo(ctx, tx, "ADJ", time.Now())
	if err != nil {
		return domain.StockAdjustment{}, err
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO erp_stock_adjustments
		(adjustment_no, warehouse_id, goods_id, direction, qty, reason, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, no, a.WarehouseID, a.GoodsID, a.Direction, a.Qty, a.Reason, a.CreatedBy)
	if err != nil {
		return domain.StockAdjustment{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.StockAdjustment{}, err
	}
	if err := recordOperation(ctx, tx, "stock-adjustment", id, "create", a.CreatedBy, operatorName, ""); err != nil {
		return domain.StockAdjustment{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.StockAdjustment{}, err
	}
	return r.GetStockAdjustment(ctx, id)
}

func (r *ERPOrderRepository) SubmitStockAdjustment(ctx context.Context, id, operatorID int64, operatorName string) error {
	return r.transition(ctx, "erp_stock_adjustments", "stock-adjustment", id, operatorID, operatorName,
		"draft", "submitted", "submit", nil, "")
}

func (r *ERPOrderRepository) DeleteDraftStockAdjustment(ctx context.Context, id, operatorID int64, operatorName string) error {
	// 库存调整没有明细表，itemTable/itemColumn 传空串跳过删明细。
	return r.deleteDraft(ctx, "erp_stock_adjustments", "", "", "stock-adjustment", id, operatorID, operatorName)
}

func (r *ERPOrderRepository) AuditStockAdjustment(ctx context.Context, id, operatorID int64, operatorName, comment string, allowSelf, allowNegative, pass bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var a domain.StockAdjustment
	if err := tx.QueryRowContext(ctx, `SELECT id, adjustment_no, warehouse_id, goods_id, direction, qty, status, created_by
		FROM erp_stock_adjustments WHERE id = ? FOR UPDATE`, id).
		Scan(&a.ID, &a.AdjustmentNo, &a.WarehouseID, &a.GoodsID, &a.Direction, &a.Qty, &a.Status, &a.CreatedBy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	if a.Status != "submitted" {
		return domain.ErrInvalidState
	}
	if a.CreatedBy == operatorID && !allowSelf {
		return fmt.Errorf("cannot audit own stock adjustment: %w", domain.ErrForbidden)
	}

	status, action := "rejected", "reject"
	if pass {
		status, action = "approved", "approve"
		delta := a.Qty
		if a.Direction == "out" {
			delta = -delta
			if !allowNegative {
				var have float64
				if err := tx.QueryRowContext(ctx,
					`SELECT COALESCE(qty, 0) FROM erp_stocks WHERE warehouse_id = ? AND goods_id = ? FOR UPDATE`,
					a.WarehouseID, a.GoodsID).Scan(&have); err != nil && !errors.Is(err, sql.ErrNoRows) {
					return err
				}
				if have < a.Qty {
					return fmt.Errorf("insufficient stock: %w", domain.ErrInvalidState)
				}
			}
		}
		if err := addStockAdjustmentDelta(ctx, tx, a.WarehouseID, a.GoodsID, delta, a.AdjustmentNo); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE erp_stock_adjustments
		SET status = ?, approved_by = ?, approved_at = CURRENT_TIMESTAMP(3), updated_at = CURRENT_TIMESTAMP(3)
		WHERE id = ?`, status, operatorID, id); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, "stock-adjustment", id, action, operatorID, operatorName, comment); err != nil {
		return err
	}
	return tx.Commit()
}

func addStockAdjustmentDelta(ctx context.Context, tx *sql.Tx, warehouseID, goodsID int64, delta float64, refNo string) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO erp_stocks (warehouse_id, goods_id, qty) VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE qty = qty + VALUES(qty)`, warehouseID, goodsID, delta); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO erp_stock_logs (warehouse_id, goods_id, change_qty, biz_type, ref_no)
		VALUES (?, ?, ?, 'adjust', ?)`, warehouseID, goodsID, delta, refNo)
	return err
}

// AuditTransfer 审核调拨单：通过置 approved，驳回置 rejected；两者都不动库存。
func (r *ERPOrderRepository) AuditTransfer(ctx context.Context, id, operatorID int64, operatorName, comment string, allowSelf, pass bool) error {
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
		`SELECT status, created_by FROM erp_transfers WHERE id = ? FOR UPDATE`, id).
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
		return fmt.Errorf("cannot audit own transfer: %w", domain.ErrForbidden)
	}

	next, action := "rejected", "reject"
	if pass {
		next, action = "approved", "approve"
	}
	if _, err := tx.ExecContext(ctx, `UPDATE erp_transfers
		SET status = ?, approved_by = ?, approved_at = CURRENT_TIMESTAMP(3), updated_at = CURRENT_TIMESTAMP(3)
		WHERE id = ?`, next, operatorID, id); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, "transfer", id, action, operatorID, operatorName, comment); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------- 库存盘点 ----------

const stockCheckSelect = `
	SELECT c.id, c.created_at, c.updated_at, c.check_no, c.warehouse_id, w.name, c.status,
	       c.remark, c.created_by, c.submitted_at, c.approved_by, c.approved_at
	FROM erp_stock_checks c
	JOIN erp_warehouses w ON w.id = c.warehouse_id`

func scanStockCheck(s interface{ Scan(...any) error }) (domain.StockCheck, error) {
	var c domain.StockCheck
	err := s.Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt, &c.CheckNo, &c.WarehouseID, &c.WarehouseName,
		&c.Status, &c.Remark, &c.CreatedBy, &c.SubmittedAt, &c.ApprovedBy, &c.ApprovedAt)
	return c, err
}

func (r *ERPOrderRepository) CountStockChecks(ctx context.Context, f OrderFilter) (int64, error) {
	where, args := f.whereColumn("erp_stock_checks", "check_no")
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM erp_stock_checks`+where, args...).Scan(&n)
	return n, err
}

func (r *ERPOrderRepository) ListStockChecks(ctx context.Context, f OrderFilter, limit, offset int) ([]domain.StockCheck, error) {
	where, args := f.whereColumn("c", "check_no")
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, stockCheckSelect+where+` ORDER BY c.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.StockCheck, 0)
	for rows.Next() {
		c, err := scanStockCheck(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (r *ERPOrderRepository) GetStockCheck(ctx context.Context, id int64) (domain.StockCheck, error) {
	c, err := scanStockCheck(r.db.QueryRowContext(ctx, stockCheckSelect+` WHERE c.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.StockCheck{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.StockCheck{}, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT i.id, i.goods_id, g.name, g.unit, i.book_qty, i.actual_qty, i.diff_qty
		FROM erp_stock_check_items i JOIN erp_goods g ON g.id = i.goods_id
		WHERE i.check_id = ? ORDER BY i.id`, id)
	if err != nil {
		return domain.StockCheck{}, err
	}
	defer rows.Close()
	c.Items = make([]domain.StockCheckItem, 0)
	for rows.Next() {
		var it domain.StockCheckItem
		if err := rows.Scan(&it.ID, &it.GoodsID, &it.GoodsName, &it.Unit,
			&it.BookQty, &it.ActualQty, &it.DiffQty); err != nil {
			return domain.StockCheck{}, err
		}
		c.Items = append(c.Items, it)
	}
	return c, rows.Err()
}

// CreateStockCheck 建盘点单：账面数由服务端从当前库存读取，避免前端伪造。
func (r *ERPOrderRepository) CreateStockCheck(ctx context.Context, c domain.StockCheck, operatorName string) (domain.StockCheck, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.StockCheck{}, err
	}
	defer tx.Rollback() //nolint:errcheck

	no, err := nextBillNo(ctx, tx, "SC", time.Now())
	if err != nil {
		return domain.StockCheck{}, err
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO erp_stock_checks
		(check_no, warehouse_id, remark, created_by) VALUES (?, ?, ?, ?)`,
		no, c.WarehouseID, c.Remark, c.CreatedBy)
	if err != nil {
		return domain.StockCheck{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.StockCheck{}, err
	}
	for _, it := range c.Items {
		var book float64
		if err := tx.QueryRowContext(ctx,
			`SELECT COALESCE(qty, 0) FROM erp_stocks WHERE warehouse_id = ? AND goods_id = ?`,
			c.WarehouseID, it.GoodsID).Scan(&book); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return domain.StockCheck{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO erp_stock_check_items
			(check_id, goods_id, book_qty, actual_qty, diff_qty) VALUES (?, ?, ?, ?, ?)`,
			id, it.GoodsID, book, it.ActualQty, it.ActualQty-book); err != nil {
			return domain.StockCheck{}, err
		}
	}
	if err := recordOperation(ctx, tx, "stock-check", id, "create", c.CreatedBy, operatorName, ""); err != nil {
		return domain.StockCheck{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.StockCheck{}, err
	}
	return r.GetStockCheck(ctx, id)
}

func (r *ERPOrderRepository) SubmitStockCheck(ctx context.Context, id, operatorID int64, operatorName string) error {
	return r.transition(ctx, "erp_stock_checks", "stock-check", id, operatorID, operatorName,
		"draft", "submitted", "submit", nil, "")
}

func (r *ERPOrderRepository) DeleteDraftStockCheck(ctx context.Context, id, operatorID int64, operatorName string) error {
	return r.deleteDraft(ctx, "erp_stock_checks", "erp_stock_check_items", "check_id", "stock-check", id, operatorID, operatorName)
}

// AuditStockCheck 审核盘点单：通过时按差异生成「已审核」的库存调整单并入账，
// 驳回只改状态。调整单与盘点单在同一事务里落库，库存流水因此有明确来源。
func (r *ERPOrderRepository) AuditStockCheck(ctx context.Context, id, operatorID int64, operatorName, comment string, allowSelf, pass bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var (
		status      string
		createdBy   int64
		warehouseID int64
		checkNo     string
	)
	if err := tx.QueryRowContext(ctx, `SELECT status, created_by, warehouse_id, check_no
		FROM erp_stock_checks WHERE id = ? FOR UPDATE`, id).
		Scan(&status, &createdBy, &warehouseID, &checkNo); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	if status != "submitted" {
		return domain.ErrInvalidState
	}
	if createdBy == operatorID && !allowSelf {
		return fmt.Errorf("cannot audit own stock check: %w", domain.ErrForbidden)
	}

	if !pass {
		if _, err := tx.ExecContext(ctx, `UPDATE erp_stock_checks
			SET status = 'rejected', approved_by = ?, approved_at = CURRENT_TIMESTAMP(3),
			    updated_at = CURRENT_TIMESTAMP(3) WHERE id = ?`, operatorID, id); err != nil {
			return err
		}
		if err := recordOperation(ctx, tx, "stock-check", id, "reject", operatorID, operatorName, comment); err != nil {
			return err
		}
		return tx.Commit()
	}

	rows, err := tx.QueryContext(ctx,
		`SELECT goods_id, diff_qty FROM erp_stock_check_items WHERE check_id = ? AND diff_qty <> 0`, id)
	if err != nil {
		return err
	}
	type diff struct {
		goodsID int64
		qty     float64
	}
	diffs := make([]diff, 0)
	for rows.Next() {
		var d diff
		if err := rows.Scan(&d.goodsID, &d.qty); err != nil {
			rows.Close()
			return err
		}
		diffs = append(diffs, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// 每个有差异的商品生成一张调整单：调整单是单商品结构，
	// 合并成一张会让除第一个商品外的差异无法从单据追溯。
	for _, d := range diffs {
		no, err := nextBillNo(ctx, tx, "ADJ", time.Now())
		if err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO erp_stock_adjustments
			(adjustment_no, warehouse_id, goods_id, direction, qty, reason, status,
			 created_by, approved_by, approved_at)
			VALUES (?, ?, ?, ?, ?, ?, 'approved', ?, ?, CURRENT_TIMESTAMP(3))`,
			no, warehouseID, d.goodsID, directionOf(d.qty), abs(d.qty),
			"盘点差异 "+checkNo, operatorID, operatorID)
		if err != nil {
			return err
		}
		adjustmentID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		if err := addStock(ctx, tx, warehouseID, d.goodsID, d.qty, "adjust", no); err != nil {
			return err
		}
		if err := recordOperation(ctx, tx, "stock-adjustment", adjustmentID, "approve",
			operatorID, operatorName, "盘点 "+checkNo); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, `UPDATE erp_stock_checks
		SET status = 'approved', approved_by = ?, approved_at = CURRENT_TIMESTAMP(3),
		    updated_at = CURRENT_TIMESTAMP(3) WHERE id = ?`, operatorID, id); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, "stock-check", id, "approve", operatorID, operatorName, comment); err != nil {
		return err
	}
	return tx.Commit()
}

func directionOf(qty float64) string {
	if qty > 0 {
		return "in"
	}
	return "out"
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// ---------- 库存调拨 ----------

const transferSelect = `
	SELECT t.id, t.created_at, t.updated_at, t.transfer_no, t.from_warehouse_id, fw.name,
	       t.to_warehouse_id, tw.name, t.status, t.remark, t.created_by,
	       t.submitted_at, t.approved_by, t.approved_at, t.completed_at
	FROM erp_transfers t
	JOIN erp_warehouses fw ON fw.id = t.from_warehouse_id
	JOIN erp_warehouses tw ON tw.id = t.to_warehouse_id`

func scanTransfer(s interface{ Scan(...any) error }) (domain.StockTransfer, error) {
	var t domain.StockTransfer
	err := s.Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt, &t.TransferNo, &t.FromWarehouseID, &t.FromName,
		&t.ToWarehouseID, &t.ToName, &t.Status, &t.Remark, &t.CreatedBy,
		&t.SubmittedAt, &t.ApprovedBy, &t.ApprovedAt, &t.CompletedAt)
	return t, err
}

func (r *ERPOrderRepository) CountTransfers(ctx context.Context, f OrderFilter) (int64, error) {
	where, args := f.whereColumn("erp_transfers", "transfer_no")
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM erp_transfers`+where, args...).Scan(&n)
	return n, err
}

func (r *ERPOrderRepository) ListTransfers(ctx context.Context, f OrderFilter, limit, offset int) ([]domain.StockTransfer, error) {
	where, args := f.whereColumn("t", "transfer_no")
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, transferSelect+where+` ORDER BY t.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.StockTransfer, 0)
	for rows.Next() {
		t, err := scanTransfer(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (r *ERPOrderRepository) GetTransfer(ctx context.Context, id int64) (domain.StockTransfer, error) {
	t, err := scanTransfer(r.db.QueryRowContext(ctx, transferSelect+` WHERE t.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.StockTransfer{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.StockTransfer{}, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT i.id, i.goods_id, g.name, g.unit, i.qty
		FROM erp_transfer_items i JOIN erp_goods g ON g.id = i.goods_id
		WHERE i.transfer_id = ? ORDER BY i.id`, id)
	if err != nil {
		return domain.StockTransfer{}, err
	}
	defer rows.Close()
	t.Items = make([]domain.TransferItem, 0)
	for rows.Next() {
		var it domain.TransferItem
		if err := rows.Scan(&it.ID, &it.GoodsID, &it.GoodsName, &it.Unit, &it.Qty); err != nil {
			return domain.StockTransfer{}, err
		}
		t.Items = append(t.Items, it)
	}
	return t, rows.Err()
}

func (r *ERPOrderRepository) CreateTransfer(ctx context.Context, t domain.StockTransfer, operatorName string) (domain.StockTransfer, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.StockTransfer{}, err
	}
	defer tx.Rollback() //nolint:errcheck

	no, err := nextBillNo(ctx, tx, "TR", time.Now())
	if err != nil {
		return domain.StockTransfer{}, err
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO erp_transfers
		(transfer_no, from_warehouse_id, to_warehouse_id, remark, created_by)
		VALUES (?, ?, ?, ?, ?)`, no, t.FromWarehouseID, t.ToWarehouseID, t.Remark, t.CreatedBy)
	if err != nil {
		return domain.StockTransfer{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.StockTransfer{}, err
	}
	for _, it := range t.Items {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO erp_transfer_items (transfer_id, goods_id, qty) VALUES (?, ?, ?)`,
			id, it.GoodsID, it.Qty); err != nil {
			return domain.StockTransfer{}, err
		}
	}
	if err := recordOperation(ctx, tx, "transfer", id, "create", t.CreatedBy, operatorName, ""); err != nil {
		return domain.StockTransfer{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.StockTransfer{}, err
	}
	return r.GetTransfer(ctx, id)
}

func (r *ERPOrderRepository) SubmitTransfer(ctx context.Context, id, operatorID int64, operatorName string) error {
	return r.transition(ctx, "erp_transfers", "transfer", id, operatorID, operatorName,
		"draft", "submitted", "submit", nil, "")
}

func (r *ERPOrderRepository) DeleteDraftTransfer(ctx context.Context, id, operatorID int64, operatorName string) error {
	return r.deleteDraft(ctx, "erp_transfers", "erp_transfer_items", "transfer_id", "transfer", id, operatorID, operatorName)
}

// CompleteTransfer 完成调拨：从调出仓扣减、向调入仓增加，同一事务写两条流水。
func (r *ERPOrderRepository) CompleteTransfer(ctx context.Context, id, operatorID int64, operatorName string, allowNegative bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var (
		status string
		fromID int64
		toID   int64
		no     string
	)
	if err := tx.QueryRowContext(ctx, `SELECT status, from_warehouse_id, to_warehouse_id, transfer_no
		FROM erp_transfers WHERE id = ? FOR UPDATE`, id).Scan(&status, &fromID, &toID, &no); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	if status != "approved" {
		return domain.ErrInvalidState
	}

	rows, err := tx.QueryContext(ctx, `SELECT goods_id, qty FROM erp_transfer_items WHERE transfer_id = ?`, id)
	if err != nil {
		return err
	}
	type line struct {
		goodsID int64
		qty     float64
	}
	lines := make([]line, 0)
	for rows.Next() {
		var l line
		if err := rows.Scan(&l.goodsID, &l.qty); err != nil {
			rows.Close()
			return err
		}
		lines = append(lines, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, l := range lines {
		if !allowNegative {
			var have float64
			if err := tx.QueryRowContext(ctx,
				`SELECT COALESCE(qty, 0) FROM erp_stocks WHERE warehouse_id = ? AND goods_id = ? FOR UPDATE`,
				fromID, l.goodsID).Scan(&have); err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			if have < l.qty {
				return fmt.Errorf("insufficient stock: %w", domain.ErrInvalidState)
			}
		}
		if err := addStock(ctx, tx, fromID, l.goodsID, -l.qty, "transfer-out", no); err != nil {
			return err
		}
		if err := addStock(ctx, tx, toID, l.goodsID, l.qty, "transfer-in", no); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE erp_transfers
		SET status = 'completed', completed_at = CURRENT_TIMESTAMP(3), updated_at = CURRENT_TIMESTAMP(3)
		WHERE id = ?`, id); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, "transfer", id, "complete", operatorID, operatorName, ""); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------- 采购单 ----------

const purchaseSelect = `
	SELECT o.id, o.created_at, o.updated_at, o.order_no, o.supplier_id, s.name,
	       o.warehouse_id, w.name, o.status, o.total_amount, o.created_by,
	       o.submitted_at, o.approved_by, o.approved_at
	FROM erp_purchase_orders o
	JOIN erp_suppliers s ON s.id = o.supplier_id
	JOIN erp_warehouses w ON w.id = o.warehouse_id`

func scanPurchase(s interface{ Scan(...any) error }) (domain.PurchaseOrder, error) {
	var o domain.PurchaseOrder
	err := s.Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt, &o.OrderNo, &o.SupplierID, &o.SupplierName,
		&o.WarehouseID, &o.WarehouseName, &o.Status, &o.TotalAmount, &o.CreatedBy,
		&o.SubmittedAt, &o.ApprovedBy, &o.ApprovedAt)
	return o, err
}

func (r *ERPOrderRepository) CountPurchases(ctx context.Context, f OrderFilter) (int64, error) {
	where, args := f.where("erp_purchase_orders")
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM erp_purchase_orders`+where, args...).Scan(&n)
	return n, err
}

func (r *ERPOrderRepository) ListPurchases(ctx context.Context, f OrderFilter, limit, offset int) ([]domain.PurchaseOrder, error) {
	where, args := f.where("o")
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, purchaseSelect+where+` ORDER BY o.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.PurchaseOrder, 0)
	for rows.Next() {
		o, err := scanPurchase(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

func (r *ERPOrderRepository) GetPurchase(ctx context.Context, id int64) (domain.PurchaseOrder, error) {
	o, err := scanPurchase(r.db.QueryRowContext(ctx, purchaseSelect+` WHERE o.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PurchaseOrder{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	items, err := r.listItems(ctx, "erp_purchase_order_items", id)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	o.Items = items
	return o, nil
}

func (r *ERPOrderRepository) CreatePurchase(ctx context.Context, o domain.PurchaseOrder, operatorName string) (domain.PurchaseOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	defer tx.Rollback() //nolint:errcheck

	no, err := nextBillNo(ctx, tx, "PO", time.Now())
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	res, err := tx.ExecContext(ctx, `
		INSERT INTO erp_purchase_orders
			(order_no, supplier_id, warehouse_id, status, total_amount, created_by)
		VALUES (?, ?, ?, 'draft', ?, ?)`, no, o.SupplierID, o.WarehouseID, o.TotalAmount, o.CreatedBy)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	if err := insertItems(ctx, tx, "erp_purchase_order_items", id, o.Items); err != nil {
		return domain.PurchaseOrder{}, err
	}
	if err := recordOperation(ctx, tx, "purchase", id, "create", o.CreatedBy, operatorName, ""); err != nil {
		return domain.PurchaseOrder{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.PurchaseOrder{}, err
	}
	return r.GetPurchase(ctx, id)
}

// UpdatePurchase 只允许草稿。行锁 + 状态判断都在事务里，避免并发提交后又被改。
func (r *ERPOrderRepository) UpdatePurchase(ctx context.Context, o domain.PurchaseOrder, operatorName string, allowAny bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var status string
	if err := tx.QueryRowContext(ctx,
		`SELECT status FROM erp_purchase_orders WHERE id = ? FOR UPDATE`, o.ID).Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	if status != "draft" && !allowAny {
		return domain.ErrInvalidState
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE erp_purchase_orders
		SET supplier_id = ?, warehouse_id = ?, total_amount = ?, updated_at = CURRENT_TIMESTAMP(3)
		WHERE id = ?`, o.SupplierID, o.WarehouseID, o.TotalAmount, o.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM erp_purchase_order_items WHERE order_id = ?`, o.ID); err != nil {
		return err
	}
	if err := insertItems(ctx, tx, "erp_purchase_order_items", o.ID, o.Items); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, "purchase", o.ID, "update", o.CreatedBy, operatorName, ""); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ERPOrderRepository) SubmitPurchase(ctx context.Context, id, operatorID int64, operatorName string) error {
	return r.transition(ctx, "erp_purchase_orders", "purchase", id, operatorID, operatorName,
		"draft", "submitted", "submit", nil, "")
}

// ApprovePurchase 审批。allowSelf 为 true 时跳过「不能审自己的单据」检查（BUG-003）。
func (r *ERPOrderRepository) ApprovePurchase(ctx context.Context, id, operatorID int64, operatorName, comment string, allowSelf bool) error {
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
		`SELECT status, created_by FROM erp_purchase_orders WHERE id = ? FOR UPDATE`, id).
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
		return fmt.Errorf("cannot approve own order: %w", domain.ErrForbidden)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE erp_purchase_orders
		SET status = 'approved', approved_by = ?, approved_at = CURRENT_TIMESTAMP(3),
		    updated_at = CURRENT_TIMESTAMP(3)
		WHERE id = ?`, operatorID, id); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, "purchase", id, "approve", operatorID, operatorName, comment); err != nil {
		return err
	}
	return tx.Commit()
}

// RejectPurchase 驳回：只允许 submitted → rejected，驳回原因写进操作记录。
func (r *ERPOrderRepository) RejectPurchase(ctx context.Context, id, operatorID int64, operatorName, comment string, allowSelf bool) error {
	return r.reject(ctx, "erp_purchase_orders", "purchase", id, operatorID, operatorName, comment, allowSelf)
}

// WithdrawPurchase 撤销提交：submitted → draft，清空提交时间，之后可继续编辑。
func (r *ERPOrderRepository) WithdrawPurchase(ctx context.Context, id, operatorID int64, operatorName string) error {
	return r.transition(ctx, "erp_purchase_orders", "purchase", id, operatorID, operatorName,
		"submitted", "draft", "withdraw", nil, "submitted_at = NULL")
}

// DeleteDraftPurchase 物理删除草稿单及其明细；操作记录保留作审计痕迹。
func (r *ERPOrderRepository) DeleteDraftPurchase(ctx context.Context, id, operatorID int64, operatorName string) error {
	return r.deleteDraft(ctx, "erp_purchase_orders", "erp_purchase_order_items", "order_id", "purchase", id, operatorID, operatorName)
}

// ReceivePurchase 入库：逐行加库存 + 写流水，最后把单据置为 completed。
func (r *ERPOrderRepository) ReceivePurchase(ctx context.Context, id, operatorID int64, operatorName string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var (
		status      string
		warehouseID int64
		orderNo     string
	)
	if err := tx.QueryRowContext(ctx, `
		SELECT status, warehouse_id, order_no FROM erp_purchase_orders WHERE id = ? FOR UPDATE`, id).
		Scan(&status, &warehouseID, &orderNo); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	if status != "approved" {
		return domain.ErrInvalidState
	}

	items, err := r.listItemsTx(ctx, tx, "erp_purchase_order_items", id)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return domain.ErrInvalidState
	}
	for _, it := range items {
		if err := addStock(ctx, tx, warehouseID, it.GoodsID, it.Qty, "purchase_in", orderNo); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE erp_purchase_orders SET status = 'completed', updated_at = CURRENT_TIMESTAMP(3) WHERE id = ?`, id); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, "purchase", id, "receive", operatorID, operatorName, ""); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------- 销售单 ----------

const salesSelect = `
	SELECT o.id, o.created_at, o.updated_at, o.order_no, o.customer_id, c.name,
	       o.warehouse_id, w.name, o.status, o.total_amount, o.created_by,
	       o.submitted_at, o.approved_by, o.approved_at
	FROM erp_sales_orders o
	JOIN erp_customers c ON c.id = o.customer_id
	JOIN erp_warehouses w ON w.id = o.warehouse_id`

func scanSales(s interface{ Scan(...any) error }) (domain.SalesOrder, error) {
	var o domain.SalesOrder
	err := s.Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt, &o.OrderNo, &o.CustomerID, &o.CustomerName,
		&o.WarehouseID, &o.WarehouseName, &o.Status, &o.TotalAmount, &o.CreatedBy,
		&o.SubmittedAt, &o.ApprovedBy, &o.ApprovedAt)
	return o, err
}

func (r *ERPOrderRepository) CountSales(ctx context.Context, f OrderFilter) (int64, error) {
	where, args := f.where("erp_sales_orders")
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM erp_sales_orders`+where, args...).Scan(&n)
	return n, err
}

func (r *ERPOrderRepository) ListSales(ctx context.Context, f OrderFilter, limit, offset int) ([]domain.SalesOrder, error) {
	where, args := f.where("o")
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, salesSelect+where+` ORDER BY o.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.SalesOrder, 0)
	for rows.Next() {
		o, err := scanSales(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

func (r *ERPOrderRepository) GetSales(ctx context.Context, id int64) (domain.SalesOrder, error) {
	o, err := scanSales(r.db.QueryRowContext(ctx, salesSelect+` WHERE o.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.SalesOrder{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.SalesOrder{}, err
	}
	items, err := r.listItems(ctx, "erp_sales_order_items", id)
	if err != nil {
		return domain.SalesOrder{}, err
	}
	o.Items = items
	return o, nil
}

func (r *ERPOrderRepository) CreateSales(ctx context.Context, o domain.SalesOrder, operatorName string) (domain.SalesOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.SalesOrder{}, err
	}
	defer tx.Rollback() //nolint:errcheck

	no, err := nextBillNo(ctx, tx, "SO", time.Now())
	if err != nil {
		return domain.SalesOrder{}, err
	}
	res, err := tx.ExecContext(ctx, `
		INSERT INTO erp_sales_orders
			(order_no, customer_id, warehouse_id, status, total_amount, created_by)
		VALUES (?, ?, ?, 'draft', ?, ?)`, no, o.CustomerID, o.WarehouseID, o.TotalAmount, o.CreatedBy)
	if err != nil {
		return domain.SalesOrder{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.SalesOrder{}, err
	}
	if err := insertItems(ctx, tx, "erp_sales_order_items", id, o.Items); err != nil {
		return domain.SalesOrder{}, err
	}
	if err := recordOperation(ctx, tx, "sales", id, "create", o.CreatedBy, operatorName, ""); err != nil {
		return domain.SalesOrder{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.SalesOrder{}, err
	}
	return r.GetSales(ctx, id)
}

func (r *ERPOrderRepository) UpdateSales(ctx context.Context, o domain.SalesOrder, operatorName string, allowAny bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var status string
	if err := tx.QueryRowContext(ctx,
		`SELECT status FROM erp_sales_orders WHERE id = ? FOR UPDATE`, o.ID).Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	if status != "draft" && !allowAny {
		return domain.ErrInvalidState
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE erp_sales_orders
		SET customer_id = ?, warehouse_id = ?, total_amount = ?, updated_at = CURRENT_TIMESTAMP(3)
		WHERE id = ?`, o.CustomerID, o.WarehouseID, o.TotalAmount, o.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM erp_sales_order_items WHERE order_id = ?`, o.ID); err != nil {
		return err
	}
	if err := insertItems(ctx, tx, "erp_sales_order_items", o.ID, o.Items); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, "sales", o.ID, "update", o.CreatedBy, operatorName, ""); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ERPOrderRepository) SubmitSales(ctx context.Context, id, operatorID int64, operatorName string) error {
	return r.transition(ctx, "erp_sales_orders", "sales", id, operatorID, operatorName,
		"draft", "submitted", "submit", nil, "")
}

func (r *ERPOrderRepository) ApproveSales(ctx context.Context, id, operatorID int64, operatorName, comment string, allowSelf bool) error {
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
		`SELECT status, created_by FROM erp_sales_orders WHERE id = ? FOR UPDATE`, id).
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
		return fmt.Errorf("cannot approve own order: %w", domain.ErrForbidden)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE erp_sales_orders
		SET status = 'approved', approved_by = ?, approved_at = CURRENT_TIMESTAMP(3),
		    updated_at = CURRENT_TIMESTAMP(3)
		WHERE id = ?`, operatorID, id); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, "sales", id, "approve", operatorID, operatorName, comment); err != nil {
		return err
	}
	return tx.Commit()
}

// RejectSales 驳回销售单。
func (r *ERPOrderRepository) RejectSales(ctx context.Context, id, operatorID int64, operatorName, comment string, allowSelf bool) error {
	return r.reject(ctx, "erp_sales_orders", "sales", id, operatorID, operatorName, comment, allowSelf)
}

// WithdrawSales 撤销提交。
func (r *ERPOrderRepository) WithdrawSales(ctx context.Context, id, operatorID int64, operatorName string) error {
	return r.transition(ctx, "erp_sales_orders", "sales", id, operatorID, operatorName,
		"submitted", "draft", "withdraw", nil, "submitted_at = NULL")
}

// DeleteDraftSales 物理删除草稿销售单。
func (r *ERPOrderRepository) DeleteDraftSales(ctx context.Context, id, operatorID int64, operatorName string) error {
	return r.deleteDraft(ctx, "erp_sales_orders", "erp_sales_order_items", "order_id", "sales", id, operatorID, operatorName)
}

// ShipSales 出库。allowNegative 为 true 时不拦负库存（BUG-002）。
func (r *ERPOrderRepository) ShipSales(ctx context.Context, id, operatorID int64, operatorName string, allowNegative bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var (
		status      string
		warehouseID int64
		orderNo     string
	)
	if err := tx.QueryRowContext(ctx, `
		SELECT status, warehouse_id, order_no FROM erp_sales_orders WHERE id = ? FOR UPDATE`, id).
		Scan(&status, &warehouseID, &orderNo); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	if status != "approved" {
		return domain.ErrInvalidState
	}

	items, err := r.listItemsTx(ctx, tx, "erp_sales_order_items", id)
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
		if err := addStock(ctx, tx, warehouseID, it.GoodsID, -it.Qty, "sales_out", orderNo); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE erp_sales_orders SET status = 'completed', updated_at = CURRENT_TIMESTAMP(3) WHERE id = ?`, id); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, "sales", id, "ship", operatorID, operatorName, ""); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------- 操作记录 ----------

func (r *ERPOrderRepository) ListBillOperations(ctx context.Context, billType string, billID int64) ([]domain.BillOperation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, created_at, bill_type, bill_id, action, operator_id, operator_name, comment
		FROM erp_bill_operations WHERE bill_type = ? AND bill_id = ? ORDER BY id`, billType, billID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.BillOperation, 0)
	for rows.Next() {
		var op domain.BillOperation
		if err := rows.Scan(&op.ID, &op.CreatedAt, &op.BillType, &op.BillID, &op.Action,
			&op.OperatorID, &op.OperatorName, &op.Comment); err != nil {
			return nil, err
		}
		list = append(list, op)
	}
	return list, rows.Err()
}

// ---------- 内部辅助 ----------

func (r *ERPOrderRepository) transition(
	ctx context.Context,
	table, billType string,
	id, operatorID int64,
	operatorName, from, to, action string,
	extra []any,
	extraSet string,
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
	if status != from {
		return domain.ErrInvalidState
	}
	set := `status = ?, updated_at = CURRENT_TIMESTAMP(3)`
	args := []any{to}
	if to == "submitted" {
		set += `, submitted_at = CURRENT_TIMESTAMP(3)`
	}
	if extraSet != "" {
		set += ", " + extraSet
		args = append(args, extra...)
	}
	args = append(args, id)
	if _, err := tx.ExecContext(ctx, `UPDATE `+table+` SET `+set+` WHERE id = ?`, args...); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, billType, id, action, operatorID, operatorName, ""); err != nil {
		return err
	}
	return tx.Commit()
}

// reject 是采购/销售单共用的驳回实现：submitted → rejected。
//
// 驳回也要写 approved_by/approved_at：语义是「审核人 / 审核时间」，不是「通过人」。
// 这样列表上「谁审的」对通过与驳回都成立。
func (r *ERPOrderRepository) reject(
	ctx context.Context,
	table, billType string,
	id, operatorID int64,
	operatorName, comment string,
	allowSelf bool,
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
		return fmt.Errorf("cannot reject own bill: %w", domain.ErrForbidden)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE `+table+`
		SET status = 'rejected', approved_by = ?, approved_at = CURRENT_TIMESTAMP(3),
		    updated_at = CURRENT_TIMESTAMP(3)
		WHERE id = ?`, operatorID, id); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, billType, id, "reject", operatorID, operatorName, comment); err != nil {
		return err
	}
	return tx.Commit()
}

// deleteDraft 删除草稿单：锁行确认状态后删明细、删主表。
//
// 操作记录留在 erp_bill_operations 里（bill_id 指向已删除的行），
// 目的是保留「谁在什么时候删了哪张单」的痕迹；单据本身已不可访问。
//
// itemTable/itemColumn 必须成对给出：各单据的明细表外键名并不统一
// （采购/销售是 order_id，调拨是 transfer_id，盘点是 check_id）。
// 传错列名会让 DELETE 报 1054 未知列，接口表现为 500 而不是删掉草稿。
// 没有明细表的单据（库存调整）两个参数都传空串，跳过删明细这一步。
func (r *ERPOrderRepository) deleteDraft(
	ctx context.Context,
	table, itemTable, itemColumn, billType string,
	id, operatorID int64,
	operatorName string,
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
	if itemTable != "" {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+itemTable+` WHERE `+itemColumn+` = ?`, id); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE id = ?`, id); err != nil {
		return err
	}
	if err := recordOperation(ctx, tx, billType, id, "delete", operatorID, operatorName, ""); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ERPOrderRepository) listItems(ctx context.Context, table string, orderID int64) ([]domain.OrderItem, error) {
	return r.listItemsQuery(ctx, r.db, table, orderID)
}

func (r *ERPOrderRepository) listItemsTx(ctx context.Context, tx *sql.Tx, table string, orderID int64) ([]domain.OrderItem, error) {
	return r.listItemsQuery(ctx, tx, table, orderID)
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (r *ERPOrderRepository) listItemsQuery(ctx context.Context, q queryer, table string, orderID int64) ([]domain.OrderItem, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT i.id, i.goods_id, g.name, i.qty, i.price, i.amount
		FROM `+table+` i JOIN erp_goods g ON g.id = i.goods_id
		WHERE i.order_id = ? ORDER BY i.id`, orderID)
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

func insertItems(ctx context.Context, tx *sql.Tx, table string, orderID int64, items []domain.OrderItem) error {
	if len(items) == 0 {
		return fmt.Errorf("order has no items: %w", domain.ErrInvalidState)
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO `+table+` (order_id, goods_id, qty, price, amount) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close() //nolint:errcheck
	for _, it := range items {
		if _, err := stmt.ExecContext(ctx, orderID, it.GoodsID, it.Qty, it.Price, it.Amount); err != nil {
			return err
		}
	}
	return nil
}

func addStock(ctx context.Context, tx *sql.Tx, warehouseID, goodsID int64, delta float64, bizType, refNo string) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO erp_stocks (warehouse_id, goods_id, qty) VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE qty = qty + VALUES(qty)`,
		warehouseID, goodsID, delta); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO erp_stock_logs (warehouse_id, goods_id, change_qty, biz_type, ref_no)
		VALUES (?, ?, ?, ?, ?)`, warehouseID, goodsID, delta, bizType, refNo)
	return err
}

func recordOperation(ctx context.Context, tx *sql.Tx, billType string, billID int64, action string, operatorID int64, operatorName, comment string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO erp_bill_operations (bill_type, bill_id, action, operator_id, operator_name, comment)
		VALUES (?, ?, ?, ?, ?, ?)`, billType, billID, action, operatorID, operatorName, comment)
	return err
}

// nextBillNo 在事务内取号：PO-20261025-000001。
//
// 用行锁串行化，不靠自增 id：单号要能按类型/日期阅读，且删单后不应回退。
func nextBillNo(ctx context.Context, tx *sql.Tx, prefix string, now time.Time) (string, error) {
	var next int64
	if err := tx.QueryRowContext(ctx,
		`SELECT next_val FROM erp_bill_no_seq WHERE prefix = ? FOR UPDATE`, prefix).Scan(&next); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO erp_bill_no_seq (prefix, next_val) VALUES (?, 1)`, prefix); err != nil {
				return "", err
			}
			next = 1
		} else {
			return "", err
		}
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE erp_bill_no_seq SET next_val = ? WHERE prefix = ?`, next+1, prefix); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%06d", prefix, now.Format("20060102"), next), nil
}
