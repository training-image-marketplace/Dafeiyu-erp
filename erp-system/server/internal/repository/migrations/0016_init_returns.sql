-- 0016 退货单：采购退货（出库 + 回冲）与销售退货（入库）
--
-- 设计要点：
--   * 退货必须挂在已完成的来源单上：采购退货冲采购入库，销售退货冲销售出库
--   * 往来单位与仓库从来源单带出，不接受前端传入，避免改单后数据不一致
--   * 数量校验分两层：建单时按「来源数量 - 其它未驳回退货」算可退量，
--     提交时在事务内用行锁再算一次，防止两张草稿各自通过、一起提交后超退
--   * 状态机与采购/销售单一致：draft → submitted → approved → completed / rejected
--   * 不建实际外键（约定见 AGENTS.md §6），关联只建索引

CREATE TABLE IF NOT EXISTS erp_purchase_returns (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    return_no       VARCHAR(64) NOT NULL,
    source_order_id BIGINT UNSIGNED NOT NULL,
    source_order_no VARCHAR(64) NOT NULL,
    supplier_id     BIGINT UNSIGNED NOT NULL,
    warehouse_id    BIGINT UNSIGNED NOT NULL,
    status          VARCHAR(16) NOT NULL DEFAULT 'draft',
    total_amount    DECIMAL(18,2) NOT NULL DEFAULT 0,
    remark          VARCHAR(255) NOT NULL DEFAULT '',
    created_by      BIGINT UNSIGNED NOT NULL,
    submitted_at    DATETIME(3) NULL,
    approved_by     BIGINT UNSIGNED NULL,
    approved_at     DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_purchase_returns_no (return_no),
    KEY idx_purchase_returns_status (status, id),
    KEY idx_purchase_returns_source (source_order_id),
    CONSTRAINT ck_purchase_returns_status
        CHECK (status IN ('draft','submitted','approved','completed','rejected'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_purchase_return_items (
    id        BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    return_id BIGINT UNSIGNED NOT NULL,
    goods_id  BIGINT UNSIGNED NOT NULL,
    qty       DECIMAL(18,3) NOT NULL,
    price     DECIMAL(18,2) NOT NULL,
    amount    DECIMAL(18,2) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_purchase_return_items_return (return_id),
    CONSTRAINT ck_purchase_return_items_qty CHECK (qty > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_sale_returns (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    return_no       VARCHAR(64) NOT NULL,
    source_order_id BIGINT UNSIGNED NOT NULL,
    source_order_no VARCHAR(64) NOT NULL,
    customer_id     BIGINT UNSIGNED NOT NULL,
    warehouse_id    BIGINT UNSIGNED NOT NULL,
    status          VARCHAR(16) NOT NULL DEFAULT 'draft',
    total_amount    DECIMAL(18,2) NOT NULL DEFAULT 0,
    remark          VARCHAR(255) NOT NULL DEFAULT '',
    created_by      BIGINT UNSIGNED NOT NULL,
    submitted_at    DATETIME(3) NULL,
    approved_by     BIGINT UNSIGNED NULL,
    approved_at     DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_sale_returns_no (return_no),
    KEY idx_sale_returns_status (status, id),
    KEY idx_sale_returns_source (source_order_id),
    CONSTRAINT ck_sale_returns_status
        CHECK (status IN ('draft','submitted','approved','completed','rejected'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_sale_return_items (
    id        BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    return_id BIGINT UNSIGNED NOT NULL,
    goods_id  BIGINT UNSIGNED NOT NULL,
    qty       DECIMAL(18,3) NOT NULL,
    price     DECIMAL(18,2) NOT NULL,
    amount    DECIMAL(18,2) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_sale_return_items_return (return_id),
    CONSTRAINT ck_sale_return_items_qty CHECK (qty > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO erp_bill_no_seq (prefix, next_val) VALUES ('PR', 1), ('SR', 1)
ON DUPLICATE KEY UPDATE prefix = prefix;
