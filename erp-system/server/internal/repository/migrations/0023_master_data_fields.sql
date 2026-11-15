-- 0023 主数据补齐：计量单位、地址与结账方式、仓库专员、商品规格与库存预警线
--
-- 对照简化版 WMS 清单：单位要能统一维护（不能只是商品上的自由文本），
-- 客户/供应商要有地址与结账方式，仓库要有专员/电话/地址，商品要有规格、
-- 建议采购价/建议销售价和最低库存（库存预警的依据）。

CREATE TABLE IF NOT EXISTS erp_units (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    name       VARCHAR(16)  NOT NULL,
    status     VARCHAR(16)  NOT NULL DEFAULT 'active',
    PRIMARY KEY (id),
    UNIQUE KEY uk_units_name (name),
    CONSTRAINT ck_units_status CHECK (status IN ('active','inactive'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO erp_units (name) VALUES ('个'), ('箱'), ('件'), ('公斤'), ('套')
ON DUPLICATE KEY UPDATE name = VALUES(name);

ALTER TABLE erp_customers
    ADD COLUMN address VARCHAR(255) NOT NULL DEFAULT '' AFTER phone,
    ADD COLUMN payment_method VARCHAR(32) NOT NULL DEFAULT '' AFTER address;

ALTER TABLE erp_suppliers
    ADD COLUMN address VARCHAR(255) NOT NULL DEFAULT '' AFTER phone,
    ADD COLUMN payment_method VARCHAR(32) NOT NULL DEFAULT '' AFTER address;

ALTER TABLE erp_warehouses
    ADD COLUMN keeper VARCHAR(64) NOT NULL DEFAULT '' AFTER location,
    ADD COLUMN phone VARCHAR(32) NOT NULL DEFAULT '' AFTER keeper,
    ADD COLUMN address VARCHAR(255) NOT NULL DEFAULT '' AFTER phone;

ALTER TABLE erp_goods
    ADD COLUMN spec VARCHAR(128) NOT NULL DEFAULT '' AFTER name,
    ADD COLUMN unit_id BIGINT UNSIGNED NULL AFTER brand_id,
    ADD COLUMN purchase_price DECIMAL(18,2) NOT NULL DEFAULT 0 AFTER price,
    ADD COLUMN sale_price DECIMAL(18,2) NOT NULL DEFAULT 0 AFTER purchase_price,
    ADD COLUMN min_qty DECIMAL(18,3) NOT NULL DEFAULT 0 AFTER sale_price,
    ADD KEY idx_goods_unit (unit_id);

-- 现有商品的 unit 文本回填到单位表引用；找不到的保留文本，unit_id 留空。
UPDATE erp_goods g JOIN erp_units u ON u.name = g.unit SET g.unit_id = u.id;

-- 建议价从现有售价/成本回填，避免老数据为空。
UPDATE erp_goods SET sale_price = price, purchase_price = cost;
