-- 0024 库存调拨单：仓库之间转移库存，调出与调入各写一条流水。
--
-- 与采购/销售一样是「单据 + 状态机」：草稿 → 提交 → 审核 → 完成（实际调拨）。
-- 审核与调拨分两步，便于定位问题：审核只改状态，完成才动库存。

CREATE TABLE IF NOT EXISTS erp_transfers (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    transfer_no    VARCHAR(64) NOT NULL,
    from_warehouse_id BIGINT UNSIGNED NOT NULL,
    to_warehouse_id   BIGINT UNSIGNED NOT NULL,
    status         VARCHAR(16) NOT NULL DEFAULT 'draft',
    remark         VARCHAR(255) NOT NULL DEFAULT '',
    created_by     BIGINT UNSIGNED NOT NULL,
    submitted_at   DATETIME(3) NULL,
    approved_by    BIGINT UNSIGNED NULL,
    approved_at    DATETIME(3) NULL,
    completed_at   DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_transfer_no (transfer_no),
    KEY idx_transfer_status (status, id),
    CONSTRAINT ck_transfer_status CHECK (status IN ('draft','submitted','approved','completed','rejected'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS erp_transfer_items (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    transfer_id BIGINT UNSIGNED NOT NULL,
    goods_id    BIGINT UNSIGNED NOT NULL,
    qty         DECIMAL(18,3) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_transfer_items_transfer (transfer_id),
    CONSTRAINT fk_transfer_items_transfer FOREIGN KEY (transfer_id) REFERENCES erp_transfers (id) ON DELETE CASCADE,
    CONSTRAINT ck_transfer_items_qty CHECK (qty > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
