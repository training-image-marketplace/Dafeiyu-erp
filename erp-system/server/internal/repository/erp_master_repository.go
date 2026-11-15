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

// ERPMasterRepository 负责主数据、库存与开关配置。
type ERPMasterRepository struct {
	db *sql.DB
}

func NewERPMasterRepository(db *sql.DB) *ERPMasterRepository {
	return &ERPMasterRepository{db: db}
}

// MasterFilter 是主数据列表的通用筛选。
type MasterFilter struct {
	Keyword string
	Status  string
}

func (f MasterFilter) where(alias string, nameCols ...string) (string, []any) {
	var (
		conds []string
		args  []any
	)
	if f.Keyword != "" {
		p := likePattern(f.Keyword)
		parts := make([]string, 0, len(nameCols))
		for _, c := range nameCols {
			parts = append(parts, alias+"."+c+" LIKE ? ESCAPE '\\\\'")
			args = append(args, p)
		}
		conds = append(conds, "("+strings.Join(parts, " OR ")+")")
	}
	if f.Status != "" {
		conds = append(conds, alias+".status = ?")
		args = append(args, f.Status)
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// ---------- 客户 ----------

func (r *ERPMasterRepository) CountCustomers(ctx context.Context, f MasterFilter) (int64, error) {
	where, args := f.where("c", "code", "name")
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM erp_customers c`+where, args...).Scan(&n)
	return n, err
}

func (r *ERPMasterRepository) ListCustomers(ctx context.Context, f MasterFilter, limit, offset int) ([]domain.Customer, error) {
	where, args := f.where("c", "code", "name")
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.created_at, c.updated_at, c.code, c.name, c.contact, c.phone,
		       c.address, c.payment_method, c.status
		FROM erp_customers c`+where+` ORDER BY c.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.Customer, 0)
	for rows.Next() {
		var c domain.Customer
		if err := rows.Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt, &c.Code, &c.Name,
			&c.Contact, &c.Phone, &c.Address, &c.PaymentMethod, &c.Status); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (r *ERPMasterRepository) CreateCustomer(ctx context.Context, c domain.Customer) (domain.Customer, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO erp_customers (code, name, contact, phone, address, payment_method, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.Code, c.Name, c.Contact, c.Phone, c.Address, c.PaymentMethod, c.Status)
	if err != nil {
		return domain.Customer{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Customer{}, err
	}
	c.ID = id
	if err := r.db.QueryRowContext(ctx,
		`SELECT created_at, updated_at FROM erp_customers WHERE id = ?`, id).
		Scan(&c.CreatedAt, &c.UpdatedAt); err != nil {
		return domain.Customer{}, err
	}
	return c, nil
}

func (r *ERPMasterRepository) UpdateCustomer(ctx context.Context, c domain.Customer) error {
	return r.updateSimple(ctx, "erp_customers",
		`name = ?, contact = ?, phone = ?, address = ?, payment_method = ?, status = ?,
		 updated_at = CURRENT_TIMESTAMP(3)`,
		[]any{c.Name, c.Contact, c.Phone, c.Address, c.PaymentMethod, c.Status, c.ID})
}

func (r *ERPMasterRepository) DeleteCustomer(ctx context.Context, id int64) error {
	return r.deleteSimple(ctx, "erp_customers", id)
}

func (r *ERPMasterRepository) CustomerRefs(ctx context.Context, id int64) (int64, error) {
	return r.countRef(ctx, "erp_sales_orders", "customer_id", id)
}

// ---------- 供应商 ----------

func (r *ERPMasterRepository) CountSuppliers(ctx context.Context, f MasterFilter) (int64, error) {
	where, args := f.where("s", "code", "name")
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM erp_suppliers s`+where, args...).Scan(&n)
	return n, err
}

func (r *ERPMasterRepository) ListSuppliers(ctx context.Context, f MasterFilter, limit, offset int) ([]domain.Supplier, error) {
	where, args := f.where("s", "code", "name")
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.id, s.created_at, s.updated_at, s.code, s.name, s.contact, s.phone,
		       s.address, s.payment_method, s.status
		FROM erp_suppliers s`+where+` ORDER BY s.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.Supplier, 0)
	for rows.Next() {
		var s domain.Supplier
		if err := rows.Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt, &s.Code, &s.Name,
			&s.Contact, &s.Phone, &s.Address, &s.PaymentMethod, &s.Status); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *ERPMasterRepository) CreateSupplier(ctx context.Context, s domain.Supplier) (domain.Supplier, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO erp_suppliers (code, name, contact, phone, address, payment_method, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		s.Code, s.Name, s.Contact, s.Phone, s.Address, s.PaymentMethod, s.Status)
	if err != nil {
		return domain.Supplier{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Supplier{}, err
	}
	s.ID = id
	if err := r.db.QueryRowContext(ctx,
		`SELECT created_at, updated_at FROM erp_suppliers WHERE id = ?`, id).
		Scan(&s.CreatedAt, &s.UpdatedAt); err != nil {
		return domain.Supplier{}, err
	}
	return s, nil
}

func (r *ERPMasterRepository) UpdateSupplier(ctx context.Context, s domain.Supplier) error {
	return r.updateSimple(ctx, "erp_suppliers",
		`name = ?, contact = ?, phone = ?, address = ?, payment_method = ?, status = ?,
		 updated_at = CURRENT_TIMESTAMP(3)`,
		[]any{s.Name, s.Contact, s.Phone, s.Address, s.PaymentMethod, s.Status, s.ID})
}

func (r *ERPMasterRepository) DeleteSupplier(ctx context.Context, id int64) error {
	return r.deleteSimple(ctx, "erp_suppliers", id)
}

func (r *ERPMasterRepository) SupplierRefs(ctx context.Context, id int64) (int64, error) {
	return r.countRef(ctx, "erp_purchase_orders", "supplier_id", id)
}

// ---------- 仓库 ----------

func (r *ERPMasterRepository) CountWarehouses(ctx context.Context, f MasterFilter) (int64, error) {
	where, args := f.where("w", "code", "name")
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM erp_warehouses w`+where, args...).Scan(&n)
	return n, err
}

func (r *ERPMasterRepository) ListWarehouses(ctx context.Context, f MasterFilter, limit, offset int) ([]domain.Warehouse, error) {
	where, args := f.where("w", "code", "name")
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, `
		SELECT w.id, w.created_at, w.updated_at, w.code, w.name, w.location,
		       w.keeper, w.phone, w.address, w.status
		FROM erp_warehouses w`+where+` ORDER BY w.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.Warehouse, 0)
	for rows.Next() {
		var w domain.Warehouse
		if err := rows.Scan(&w.ID, &w.CreatedAt, &w.UpdatedAt, &w.Code, &w.Name, &w.Location,
			&w.Keeper, &w.Phone, &w.Address, &w.Status); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

func (r *ERPMasterRepository) CreateWarehouse(ctx context.Context, w domain.Warehouse) (domain.Warehouse, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO erp_warehouses (code, name, location, keeper, phone, address, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		w.Code, w.Name, w.Location, w.Keeper, w.Phone, w.Address, w.Status)
	if err != nil {
		return domain.Warehouse{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Warehouse{}, err
	}
	w.ID = id
	if err := r.db.QueryRowContext(ctx,
		`SELECT created_at, updated_at FROM erp_warehouses WHERE id = ?`, id).
		Scan(&w.CreatedAt, &w.UpdatedAt); err != nil {
		return domain.Warehouse{}, err
	}
	return w, nil
}

func (r *ERPMasterRepository) UpdateWarehouse(ctx context.Context, w domain.Warehouse) error {
	return r.updateSimple(ctx, "erp_warehouses",
		`name = ?, location = ?, keeper = ?, phone = ?, address = ?, status = ?,
		 updated_at = CURRENT_TIMESTAMP(3)`,
		[]any{w.Name, w.Location, w.Keeper, w.Phone, w.Address, w.Status, w.ID})
}

func (r *ERPMasterRepository) DeleteWarehouse(ctx context.Context, id int64) error {
	return r.deleteSimple(ctx, "erp_warehouses", id)
}

func (r *ERPMasterRepository) WarehouseRefs(ctx context.Context, id int64) (int64, error) {
	tables := []string{"erp_stocks", "erp_purchase_orders", "erp_sales_orders",
		"erp_purchase_returns", "erp_sale_returns", "erp_stock_adjustments",
		"erp_transfers", "erp_stock_checks"}
	var total int64
	for _, table := range tables {
		column := "warehouse_id"
		if table == "erp_transfers" {
			// 调拨有两个仓库列，任一引用都算依赖。
			n, err := r.countRef(ctx, table, "from_warehouse_id", id)
			if err != nil {
				return 0, err
			}
			total += n
			column = "to_warehouse_id"
		}
		n, err := r.countRef(ctx, table, column, id)
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

// ---------- 商品 ----------

func (r *ERPMasterRepository) CountGoods(ctx context.Context, f MasterFilter) (int64, error) {
	where, args := f.where("g", "code", "name")
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM erp_goods g`+where, args...).Scan(&n)
	return n, err
}

func (r *ERPMasterRepository) ListGoods(ctx context.Context, f MasterFilter, limit, offset int) ([]domain.Goods, error) {
	where, args := f.where("g", "code", "name")
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, `
		SELECT g.id, g.created_at, g.updated_at, g.code, g.name, g.spec, g.category_id, g.brand_id,
		       g.unit, g.unit_id, g.price, g.cost, g.purchase_price, g.sale_price, g.min_qty, g.status,
		       COALESCE((SELECT SUM(s.qty) FROM erp_stocks s WHERE s.goods_id = g.id), 0)
		FROM erp_goods g`+where+` ORDER BY g.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.Goods, 0)
	for rows.Next() {
		var (
			g  domain.Goods
			ci sql.NullInt64
			bi sql.NullInt64
			ui sql.NullInt64
		)
		if err := rows.Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt, &g.Code, &g.Name, &g.Spec, &ci, &bi,
			&g.Unit, &ui, &g.Price, &g.Cost, &g.PurchasePrice, &g.SalePrice, &g.MinQty,
			&g.Status, &g.StockQty); err != nil {
			return nil, err
		}
		if ci.Valid {
			g.CategoryID = &ci.Int64
		}
		if bi.Valid {
			g.BrandID = &bi.Int64
		}
		if ui.Valid {
			g.UnitID = &ui.Int64
		}
		list = append(list, g)
	}
	return list, rows.Err()
}

func (r *ERPMasterRepository) CreateGoods(ctx context.Context, g domain.Goods) (domain.Goods, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO erp_goods (code, name, spec, category_id, brand_id, unit, unit_id,
		                       price, cost, purchase_price, sale_price, min_qty, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		g.Code, g.Name, g.Spec, g.CategoryID, g.BrandID, g.Unit, g.UnitID,
		g.Price, g.Cost, g.PurchasePrice, g.SalePrice, g.MinQty, g.Status)
	if err != nil {
		return domain.Goods{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Goods{}, err
	}
	g.ID = id
	if err := r.db.QueryRowContext(ctx,
		`SELECT created_at, updated_at FROM erp_goods WHERE id = ?`, id).
		Scan(&g.CreatedAt, &g.UpdatedAt); err != nil {
		return domain.Goods{}, err
	}
	return g, nil
}

func (r *ERPMasterRepository) UpdateGoods(ctx context.Context, g domain.Goods) error {
	return r.updateSimple(ctx, "erp_goods", `
		name = ?, spec = ?, category_id = ?, brand_id = ?, unit = ?, unit_id = ?,
		price = ?, cost = ?, purchase_price = ?, sale_price = ?, min_qty = ?, status = ?,
		updated_at = CURRENT_TIMESTAMP(3)`,
		[]any{g.Name, g.Spec, g.CategoryID, g.BrandID, g.Unit, g.UnitID,
			g.Price, g.Cost, g.PurchasePrice, g.SalePrice, g.MinQty, g.Status, g.ID})
}

func (r *ERPMasterRepository) DeleteGoods(ctx context.Context, id int64) error {
	return r.deleteSimple(ctx, "erp_goods", id)
}

func (r *ERPMasterRepository) GoodsRefs(ctx context.Context, id int64) (int64, error) {
	tables := []string{"erp_stocks", "erp_purchase_order_items", "erp_sales_order_items",
		"erp_purchase_return_items", "erp_sale_return_items", "erp_stock_adjustments",
		"erp_transfer_items", "erp_stock_check_items"}
	var total int64
	for _, table := range tables {
		n, err := r.countRef(ctx, table, "goods_id", id)
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

// ---------- 分类 / 品牌 ----------

func (r *ERPMasterRepository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, parent_id, name, sort FROM erp_categories ORDER BY sort, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.Category, 0)
	for rows.Next() {
		var (
			c  domain.Category
			pi sql.NullInt64
		)
		if err := rows.Scan(&c.ID, &pi, &c.Name, &c.Sort); err != nil {
			return nil, err
		}
		if pi.Valid {
			c.ParentID = &pi.Int64
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (r *ERPMasterRepository) CreateCategory(ctx context.Context, c domain.Category) (domain.Category, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO erp_categories (parent_id, name, sort) VALUES (?, ?, ?)`,
		c.ParentID, c.Name, c.Sort)
	if err != nil {
		return domain.Category{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Category{}, err
	}
	c.ID = id
	return c, nil
}

func (r *ERPMasterRepository) UpdateCategory(ctx context.Context, c domain.Category) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE erp_categories SET parent_id = ?, name = ?, sort = ? WHERE id = ?`,
		c.ParentID, c.Name, c.Sort, c.ID)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (r *ERPMasterRepository) DeleteCategory(ctx context.Context, id int64) error {
	return r.deleteSimple(ctx, "erp_categories", id)
}

func (r *ERPMasterRepository) CategoryRefs(ctx context.Context, id int64) (int64, error) {
	n1, err := r.countRef(ctx, "erp_goods", "category_id", id)
	if err != nil {
		return 0, err
	}
	n2, err := r.countRef(ctx, "erp_categories", "parent_id", id)
	return n1 + n2, err
}

func (r *ERPMasterRepository) UpdateBrand(ctx context.Context, b domain.Brand) error {
	res, err := r.db.ExecContext(ctx, `UPDATE erp_brands SET name = ? WHERE id = ?`, b.Name, b.ID)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (r *ERPMasterRepository) DeleteBrand(ctx context.Context, id int64) error {
	return r.deleteSimple(ctx, "erp_brands", id)
}

func (r *ERPMasterRepository) BrandRefs(ctx context.Context, id int64) (int64, error) {
	return r.countRef(ctx, "erp_goods", "brand_id", id)
}

func (r *ERPMasterRepository) ListBrands(ctx context.Context) ([]domain.Brand, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name FROM erp_brands ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.Brand, 0)
	for rows.Next() {
		var b domain.Brand
		if err := rows.Scan(&b.ID, &b.Name); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

func (r *ERPMasterRepository) CreateBrand(ctx context.Context, b domain.Brand) (domain.Brand, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO erp_brands (name) VALUES (?)`, b.Name)
	if err != nil {
		return domain.Brand{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Brand{}, err
	}
	b.ID = id
	return b, nil
}

// ---------- 计量单位 ----------

func (r *ERPMasterRepository) ListUnits(ctx context.Context, status string) ([]domain.Unit, error) {
	where := ""
	args := []any{}
	if status != "" {
		where = " WHERE status = ?"
		args = append(args, status)
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, created_at, updated_at, name, status FROM erp_units`+where+` ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.Unit, 0)
	for rows.Next() {
		var u domain.Unit
		if err := rows.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt, &u.Name, &u.Status); err != nil {
			return nil, err
		}
		list = append(list, u)
	}
	return list, rows.Err()
}

func (r *ERPMasterRepository) CreateUnit(ctx context.Context, u domain.Unit) (domain.Unit, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO erp_units (name, status) VALUES (?, ?)`, u.Name, u.Status)
	if err != nil {
		return domain.Unit{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Unit{}, err
	}
	return r.GetUnit(ctx, id)
}

func (r *ERPMasterRepository) GetUnit(ctx context.Context, id int64) (domain.Unit, error) {
	var u domain.Unit
	err := r.db.QueryRowContext(ctx,
		`SELECT id, created_at, updated_at, name, status FROM erp_units WHERE id = ?`, id).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt, &u.Name, &u.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Unit{}, domain.ErrNotFound
	}
	return u, err
}

func (r *ERPMasterRepository) UpdateUnit(ctx context.Context, u domain.Unit) error {
	return r.updateSimple(ctx, "erp_units", `name = ?, status = ?, updated_at = CURRENT_TIMESTAMP(3)`,
		[]any{u.Name, u.Status, u.ID})
}

func (r *ERPMasterRepository) DeleteUnit(ctx context.Context, id int64) error {
	return r.deleteSimple(ctx, "erp_units", id)
}

// UnitRefs 统计有多少商品引用该单位，被引用时不允许删除。
func (r *ERPMasterRepository) UnitRefs(ctx context.Context, id int64) (int64, error) {
	return r.countRef(ctx, "erp_goods", "unit_id", id)
}

// ---------- 库存 ----------

type StockFilter struct {
	WarehouseID int64
	Keyword     string
}

func (r *ERPMasterRepository) CountStocks(ctx context.Context, f StockFilter) (int64, error) {
	where, args := stockWhere(f, "s")
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM erp_stocks s
		JOIN erp_goods g ON g.id = s.goods_id`+where, args...).Scan(&n)
	return n, err
}

// StockTotals 返回有库存的商品种类数与库存总量，供仪表盘使用。
//
// 只统计 qty > 0 的行：0 和负数不构成「有库存」，否则一个从未入库的商品
// 也会被算进种类数里。
func (r *ERPMasterRepository) StockTotals(ctx context.Context) (int64, float64, error) {
	var (
		skus  int64
		total float64
	)
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT goods_id), COALESCE(SUM(qty), 0)
		FROM erp_stocks WHERE qty > 0`).Scan(&skus, &total)
	return skus, total, err
}

func stockWhere(f StockFilter, alias string) (string, []any) {
	var (
		conds []string
		args  []any
	)
	if f.WarehouseID > 0 {
		conds = append(conds, alias+".warehouse_id = ?")
		args = append(args, f.WarehouseID)
	}
	if f.Keyword != "" {
		p := likePattern(f.Keyword)
		conds = append(conds, "(g.name LIKE ? ESCAPE '\\\\' OR g.code LIKE ? ESCAPE '\\\\')")
		args = append(args, p, p)
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// ListStockAlerts 返回库存低于最低库存的商品（全仓汇总口径）。
func (r *ERPMasterRepository) ListStockAlerts(ctx context.Context, limit int) ([]domain.Stock, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.warehouse_id, w.name, s.goods_id, g.code, g.name, g.unit, s.qty, s.updated_at
		FROM erp_stocks s
		JOIN erp_goods g ON g.id = s.goods_id
		JOIN erp_warehouses w ON w.id = s.warehouse_id
		WHERE g.min_qty > 0
		  AND (SELECT SUM(x.qty) FROM erp_stocks x WHERE x.goods_id = s.goods_id) < g.min_qty
		ORDER BY s.qty ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.Stock, 0)
	for rows.Next() {
		var s domain.Stock
		if err := rows.Scan(&s.WarehouseID, &s.WarehouseName, &s.GoodsID, &s.GoodsCode,
			&s.GoodsName, &s.Unit, &s.Qty, &s.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// CountStockAlerts 统计低于最低库存的商品数量（去重到商品）。
func (r *ERPMasterRepository) CountStockAlerts(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM erp_goods g
		WHERE g.min_qty > 0
		  AND COALESCE((SELECT SUM(x.qty) FROM erp_stocks x WHERE x.goods_id = g.id), 0) < g.min_qty`).Scan(&n)
	return n, err
}

func (r *ERPMasterRepository) ListStocks(ctx context.Context, f StockFilter, limit, offset int) ([]domain.Stock, error) {
	where, args := stockWhere(f, "s")
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.warehouse_id, w.name, s.goods_id, g.code, g.name, g.unit, s.qty, s.updated_at
		FROM erp_stocks s
		JOIN erp_goods g ON g.id = s.goods_id
		JOIN erp_warehouses w ON w.id = s.warehouse_id`+where+`
		ORDER BY s.updated_at DESC, s.goods_id LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.Stock, 0)
	for rows.Next() {
		var s domain.Stock
		if err := rows.Scan(&s.WarehouseID, &s.WarehouseName, &s.GoodsID, &s.GoodsCode,
			&s.GoodsName, &s.Unit, &s.Qty, &s.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *ERPMasterRepository) CountStockLogs(ctx context.Context, f StockFilter) (int64, error) {
	where, args := stockWhere(f, "l")
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM erp_stock_logs l
		JOIN erp_goods g ON g.id = l.goods_id`+where, args...).Scan(&n)
	return n, err
}

func (r *ERPMasterRepository) ListStockLogs(ctx context.Context, f StockFilter, limit, offset int) ([]domain.StockLog, error) {
	where, args := stockWhere(f, "l")
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, `
		SELECT l.id, l.created_at, l.warehouse_id, l.goods_id, g.name, l.change_qty, l.biz_type, l.ref_no
		FROM erp_stock_logs l
		JOIN erp_goods g ON g.id = l.goods_id`+where+`
		ORDER BY l.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.StockLog, 0)
	for rows.Next() {
		var l domain.StockLog
		if err := rows.Scan(&l.ID, &l.CreatedAt, &l.WarehouseID, &l.GoodsID, &l.GoodsName,
			&l.ChangeQty, &l.BizType, &l.RefNo); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}

// ---------- 报表 ----------

// StockFlowRow 是「入库/出库统计」的一行：按日期区间汇总流水。
type StockFlowRow struct {
	GoodsID   int64   `json:"goods_id"`
	GoodsCode string  `json:"goods_code"`
	GoodsName string  `json:"goods_name"`
	Unit      string  `json:"unit"`
	InQty     float64 `json:"in_qty"`
	OutQty    float64 `json:"out_qty"`
	NetQty    float64 `json:"net_qty"`
}

// StockFlowReport 汇总一段时间内的出入库数量。正数流水计为入库，负数计为出库。
func (r *ERPMasterRepository) StockFlowReport(ctx context.Context, from, to string, warehouseID int64) ([]StockFlowRow, error) {
	conds := []string{"l.created_at >= ?", "l.created_at < ?"}
	args := []any{from, to}
	if warehouseID > 0 {
		conds = append(conds, "l.warehouse_id = ?")
		args = append(args, warehouseID)
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT g.id, g.code, g.name, g.unit,
		       COALESCE(SUM(CASE WHEN l.change_qty > 0 THEN l.change_qty ELSE 0 END), 0) AS in_qty,
		       COALESCE(SUM(CASE WHEN l.change_qty < 0 THEN -l.change_qty ELSE 0 END), 0) AS out_qty,
		       COALESCE(SUM(l.change_qty), 0) AS net_qty
		FROM erp_stock_logs l
		JOIN erp_goods g ON g.id = l.goods_id
		WHERE `+strings.Join(conds, " AND ")+`
		GROUP BY g.id, g.code, g.name, g.unit
		ORDER BY g.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]StockFlowRow, 0)
	for rows.Next() {
		var v StockFlowRow
		if err := rows.Scan(&v.GoodsID, &v.GoodsCode, &v.GoodsName, &v.Unit,
			&v.InQty, &v.OutQty, &v.NetQty); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

// ---------- 开关 ----------

func (r *ERPMasterRepository) GetSwitchConfig(ctx context.Context, scope string, scopeID int64) ([]byte, error) {
	var raw []byte
	err := r.db.QueryRowContext(ctx,
		`SELECT config_json FROM erp_switch_configs WHERE scope = ? AND scope_id = ?`,
		scope, scopeID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return raw, err
}

type SwitchRecord struct {
	Scope     string    `json:"scope"`
	ScopeID   int64     `json:"scope_id"`
	Config    []byte    `json:"config"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r *ERPMasterRepository) ListSwitchConfigs(ctx context.Context) ([]SwitchRecord, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT scope, scope_id, config_json, updated_at FROM erp_switch_configs ORDER BY scope, scope_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]SwitchRecord, 0)
	for rows.Next() {
		var rec SwitchRecord
		if err := rows.Scan(&rec.Scope, &rec.ScopeID, &rec.Config, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, rec)
	}
	return list, rows.Err()
}

// SetSwitchConfig 覆盖某作用域配置（平台下发时整份替换，避免合并歧义）。
func (r *ERPMasterRepository) SetSwitchConfig(ctx context.Context, scope string, scopeID int64, config []byte) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO erp_switch_configs (scope, scope_id, config_json) VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE config_json = VALUES(config_json), updated_at = CURRENT_TIMESTAMP(3)`,
		scope, scopeID, config)
	return err
}

// Exists 判断某张表里是否有指定主键的记录。表名只允许内部常量，不接受用户输入。
func (r *ERPMasterRepository) Exists(ctx context.Context, table string, id int64) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE id = ?)`, table), id).Scan(&ok)
	return ok, err
}

// CodeExists 判断编码是否被占用（excludeID=0 表示新建）。
func (r *ERPMasterRepository) CodeExists(ctx context.Context, table, code string, excludeID int64) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE code = ? AND id <> ?)`, table), code, excludeID).Scan(&ok)
	return ok, err
}

// ---------- 内部辅助 ----------

func (r *ERPMasterRepository) updateSimple(ctx context.Context, table, set string, args []any) error {
	res, err := r.db.ExecContext(ctx, `UPDATE `+table+` SET `+set+` WHERE id = ?`, args...)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (r *ERPMasterRepository) deleteSimple(ctx context.Context, table string, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM `+table+` WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (r *ERPMasterRepository) countRef(ctx context.Context, table, col string, id int64) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s = ?`, table, col), id).Scan(&n)
	return n, err
}
