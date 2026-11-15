-- 0014 ERP 业务表：主数据、库存、单据、开关
--
-- 设计要点：
--   * 库存用「余额表 + 流水表」：余额供查询，流水供追溯，两者同事务更新
--   * 单据分主子表；状态机 draft → submitted → approved → completed
--   * 单号由 erp_bill_no_seq 取号，用行锁保证并发唯一，不依赖自增 id
--   * 开关配置按作用域存一份 JSON：all / group / personal，
--     平台是配置真源，这里只负责执行

CREATE TABLE IF NOT EXISTS erp_customers (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    code       VARCHAR(32)  NOT NULL,
    name       VARCHAR(128) NOT NULL,
    contact    VARCHAR(64)  NOT NULL DEFAULT '',
    phone      VARCHAR(32)  NOT NULL DEFAULT '',
    status     VARCHAR(16)  NOT NULL DEFAULT 'active',
    PRIMARY KEY (id),
    UNIQUE KEY uk_customers_code (code),
    CONSTRAINT ck_customers_status CHECK (status IN ('active','inactive'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_suppliers (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    code       VARCHAR(32)  NOT NULL,
    name       VARCHAR(128) NOT NULL,
    contact    VARCHAR(64)  NOT NULL DEFAULT '',
    phone      VARCHAR(32)  NOT NULL DEFAULT '',
    status     VARCHAR(16)  NOT NULL DEFAULT 'active',
    PRIMARY KEY (id),
    UNIQUE KEY uk_suppliers_code (code),
    CONSTRAINT ck_suppliers_status CHECK (status IN ('active','inactive'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_categories (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    parent_id  BIGINT UNSIGNED NULL,
    name       VARCHAR(128) NOT NULL,
    sort       INT NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    KEY idx_categories_parent (parent_id, sort),
    CONSTRAINT fk_categories_parent FOREIGN KEY (parent_id) REFERENCES erp_categories (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_brands (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    name       VARCHAR(128) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_brands_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_goods (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    code        VARCHAR(32)  NOT NULL,
    name        VARCHAR(128) NOT NULL,
    category_id BIGINT UNSIGNED NULL,
    brand_id    BIGINT UNSIGNED NULL,
    unit        VARCHAR(16)  NOT NULL DEFAULT '件',
    price       DECIMAL(18,2) NOT NULL DEFAULT 0,
    cost        DECIMAL(18,2) NOT NULL DEFAULT 0,
    status      VARCHAR(16)  NOT NULL DEFAULT 'active',
    PRIMARY KEY (id),
    UNIQUE KEY uk_goods_code (code),
    KEY idx_goods_category (category_id),
    CONSTRAINT fk_goods_category FOREIGN KEY (category_id) REFERENCES erp_categories (id),
    CONSTRAINT fk_goods_brand FOREIGN KEY (brand_id) REFERENCES erp_brands (id),
    CONSTRAINT ck_goods_status CHECK (status IN ('active','inactive')),
    CONSTRAINT ck_goods_price CHECK (price >= 0 AND cost >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_warehouses (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    code       VARCHAR(32)  NOT NULL,
    name       VARCHAR(128) NOT NULL,
    location   VARCHAR(255) NOT NULL DEFAULT '',
    status     VARCHAR(16)  NOT NULL DEFAULT 'active',
    PRIMARY KEY (id),
    UNIQUE KEY uk_warehouses_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_stocks (
    warehouse_id BIGINT UNSIGNED NOT NULL,
    goods_id     BIGINT UNSIGNED NOT NULL,
    qty          DECIMAL(18,3) NOT NULL DEFAULT 0,
    updated_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (warehouse_id, goods_id),
    CONSTRAINT fk_stocks_wh FOREIGN KEY (warehouse_id) REFERENCES erp_warehouses (id),
    CONSTRAINT fk_stocks_goods FOREIGN KEY (goods_id) REFERENCES erp_goods (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_stock_logs (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    warehouse_id BIGINT UNSIGNED NOT NULL,
    goods_id     BIGINT UNSIGNED NOT NULL,
    change_qty   DECIMAL(18,3) NOT NULL,
    biz_type     VARCHAR(16) NOT NULL,
    ref_no       VARCHAR(64) NOT NULL DEFAULT '',
    PRIMARY KEY (id),
    KEY idx_stock_logs_goods (goods_id, created_at),
    KEY idx_stock_logs_ref (ref_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_purchase_orders (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    order_no     VARCHAR(64) NOT NULL,
    supplier_id  BIGINT UNSIGNED NOT NULL,
    warehouse_id BIGINT UNSIGNED NOT NULL,
    status       VARCHAR(16) NOT NULL DEFAULT 'draft',
    total_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
    created_by   BIGINT UNSIGNED NOT NULL,
    submitted_at DATETIME(3) NULL,
    approved_by  BIGINT UNSIGNED NULL,
    approved_at  DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_po_no (order_no),
    KEY idx_po_status (status, id),
    CONSTRAINT fk_po_supplier FOREIGN KEY (supplier_id) REFERENCES erp_suppliers (id),
    CONSTRAINT fk_po_wh FOREIGN KEY (warehouse_id) REFERENCES erp_warehouses (id),
    CONSTRAINT ck_po_status CHECK (status IN ('draft','submitted','approved','completed','rejected'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_purchase_order_items (
    id       BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    order_id BIGINT UNSIGNED NOT NULL,
    goods_id BIGINT UNSIGNED NOT NULL,
    qty      DECIMAL(18,3) NOT NULL,
    price    DECIMAL(18,2) NOT NULL,
    amount   DECIMAL(18,2) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_poi_order (order_id),
    CONSTRAINT fk_poi_order FOREIGN KEY (order_id) REFERENCES erp_purchase_orders (id) ON DELETE CASCADE,
    CONSTRAINT fk_poi_goods FOREIGN KEY (goods_id) REFERENCES erp_goods (id),
    CONSTRAINT ck_poi_qty CHECK (qty > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_sales_orders (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    order_no     VARCHAR(64) NOT NULL,
    customer_id  BIGINT UNSIGNED NOT NULL,
    warehouse_id BIGINT UNSIGNED NOT NULL,
    status       VARCHAR(16) NOT NULL DEFAULT 'draft',
    total_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
    created_by   BIGINT UNSIGNED NOT NULL,
    submitted_at DATETIME(3) NULL,
    approved_by  BIGINT UNSIGNED NULL,
    approved_at  DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_so_no (order_no),
    KEY idx_so_status (status, id),
    CONSTRAINT fk_so_customer FOREIGN KEY (customer_id) REFERENCES erp_customers (id),
    CONSTRAINT fk_so_wh FOREIGN KEY (warehouse_id) REFERENCES erp_warehouses (id),
    CONSTRAINT ck_so_status CHECK (status IN ('draft','submitted','approved','completed','rejected'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_sales_order_items (
    id       BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    order_id BIGINT UNSIGNED NOT NULL,
    goods_id BIGINT UNSIGNED NOT NULL,
    qty      DECIMAL(18,3) NOT NULL,
    price    DECIMAL(18,2) NOT NULL,
    amount   DECIMAL(18,2) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_soi_order (order_id),
    CONSTRAINT fk_soi_order FOREIGN KEY (order_id) REFERENCES erp_sales_orders (id) ON DELETE CASCADE,
    CONSTRAINT fk_soi_goods FOREIGN KEY (goods_id) REFERENCES erp_goods (id),
    CONSTRAINT ck_soi_qty CHECK (qty > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_bill_operations (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    bill_type     VARCHAR(16) NOT NULL,
    bill_id       BIGINT UNSIGNED NOT NULL,
    action        VARCHAR(16) NOT NULL,
    operator_id   BIGINT UNSIGNED NOT NULL,
    operator_name VARCHAR(128) NOT NULL,
    comment       VARCHAR(500) NOT NULL DEFAULT '',
    PRIMARY KEY (id),
    KEY idx_bill_ops (bill_type, bill_id, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_bill_no_seq (
    prefix   VARCHAR(16) NOT NULL,
    next_val BIGINT UNSIGNED NOT NULL DEFAULT 1,
    PRIMARY KEY (prefix)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 开关配置：一个作用域一份。scope_id 对 all 固定为 0。
CREATE TABLE IF NOT EXISTS erp_switch_configs (
    scope      VARCHAR(16) NOT NULL,
    scope_id   BIGINT UNSIGNED NOT NULL DEFAULT 0,
    config_json JSON NOT NULL,
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (scope, scope_id),
    CONSTRAINT ck_switch_scope CHECK (scope IN ('all','group','personal'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO erp_bill_no_seq (prefix, next_val) VALUES ('PO', 1), ('SO', 1)
ON DUPLICATE KEY UPDATE prefix = prefix;
