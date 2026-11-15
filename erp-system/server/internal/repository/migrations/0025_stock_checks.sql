-- 0025 库存盘点单：按仓库盘点，记录账面数与实盘数，差异审核后生成库存调整单。
--
-- 设计要点：盘点单本身不动库存；审核通过时按差异生成一张「已审核」的库存调整单，
-- 库存流水因此始终有明确来源单号，盘点与调整不出现两套口径。

CREATE TABLE IF NOT EXISTS erp_stock_checks (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    check_no     VARCHAR(64) NOT NULL,
    warehouse_id BIGINT UNSIGNED NOT NULL,
    status       VARCHAR(16) NOT NULL DEFAULT 'draft',
    remark       VARCHAR(255) NOT NULL DEFAULT '',
    created_by   BIGINT UNSIGNED NOT NULL,
    submitted_at DATETIME(3) NULL,
    approved_by  BIGINT UNSIGNED NULL,
    approved_at  DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_stock_check_no (check_no),
    KEY idx_stock_check_status (status, id),
    CONSTRAINT ck_stock_check_status CHECK (status IN ('draft','submitted','approved','rejected'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_stock_check_items (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    check_id    BIGINT UNSIGNED NOT NULL,
    goods_id    BIGINT UNSIGNED NOT NULL,
    book_qty    DECIMAL(18,3) NOT NULL,
    actual_qty  DECIMAL(18,3) NOT NULL,
    diff_qty    DECIMAL(18,3) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_stock_check_items_check (check_id),
    CONSTRAINT fk_stock_check_items_check FOREIGN KEY (check_id) REFERENCES erp_stock_checks (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
