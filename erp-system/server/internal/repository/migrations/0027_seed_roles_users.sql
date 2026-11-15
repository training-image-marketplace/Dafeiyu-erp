-- 0027 角色模板与演示用户
--
-- 对照简化版 WMS 的角色矩阵：仓库主管审核、仓管员收发、采购管采购、销售管销售。
-- 密码都是弱口令，仅供教学演示，登录后应立即修改。
-- 权限码沿用现有集合，不新增码：主管拿 audit + stock，仓管拿 stock，
-- 采购拿 purchase + master，销售拿 sales + master。

INSERT INTO sys_roles (code, name, remark) VALUES
    ('warehouse_manager', '仓库主管', '审核出入库与调拨、盘点'),
    ('warehouse_keeper', '仓管员', '商品收发、库存操作'),
    ('buyer', '采购人员', '采购入库与供应商管理'),
    ('seller', '销售人员', '销售出库与客户管理')
ON DUPLICATE KEY UPDATE name = VALUES(name), remark = VALUES(remark);

INSERT INTO sys_users (username, password, nickname, email) VALUES
    ('manager', '$2a$10$x7ZOAu2rR46VPZ93weElquPft.KE03YQr7/eMjTkn5Ohlr6L6XYxq', '仓库主管', ''),
    ('keeper', '$2a$10$SGP0AwkT0/xJ9YBlyhi/AuDbMar82EKJnhu2s/6yYc4Fggcrc7jp.', '仓管员', ''),
    ('buyer', '$2a$10$stbP4kvIo9MIpkssscu15eV2jvnJE7Rf/pnUETZfgUu8BCTGb/KRm', '采购员', ''),
    ('seller', '$2a$10$EP4keVbCRhXYrcr9AkzKvOEaIGGmtoiu/nW2H5BfgaGzqYMYzNouy', '销售员', '')
ON DUPLICATE KEY UPDATE nickname = VALUES(nickname);

INSERT INTO sys_user_roles (user_id, role_id)
SELECT u.id, r.id FROM sys_users u JOIN sys_roles r ON r.code = 'warehouse_manager'
WHERE u.username = 'manager'
ON DUPLICATE KEY UPDATE user_id = VALUES(user_id);

INSERT INTO sys_user_roles (user_id, role_id)
SELECT u.id, r.id FROM sys_users u JOIN sys_roles r ON r.code = 'warehouse_keeper'
WHERE u.username = 'keeper'
ON DUPLICATE KEY UPDATE user_id = VALUES(user_id);

INSERT INTO sys_user_roles (user_id, role_id)
SELECT u.id, r.id FROM sys_users u JOIN sys_roles r ON r.code = 'buyer'
WHERE u.username = 'buyer'
ON DUPLICATE KEY UPDATE user_id = VALUES(user_id);

INSERT INTO sys_user_roles (user_id, role_id)
SELECT u.id, r.id FROM sys_users u JOIN sys_roles r ON r.code = 'seller'
WHERE u.username = 'seller'
ON DUPLICATE KEY UPDATE user_id = VALUES(user_id);

-- 主管：审核 + 库存 + 主数据查看
INSERT INTO sys_role_perms (role_id, perm_code)
SELECT r.id, c FROM sys_roles r, (
    SELECT 'erp:audit:list' AS c UNION ALL SELECT 'erp:audit:approve'
    UNION ALL SELECT 'erp:stock:list' UNION ALL SELECT 'erp:stock:edit'
    UNION ALL SELECT 'erp:master:list' UNION ALL SELECT 'erp:purchase:list'
    UNION ALL SELECT 'erp:sales:list' UNION ALL SELECT 'erp:return:list'
    UNION ALL SELECT 'tool:file:list'
) perms
WHERE r.code = 'warehouse_manager'
ON DUPLICATE KEY UPDATE role_id = VALUES(role_id);

-- 仓管员：收发与库存，不看审核
INSERT INTO sys_role_perms (role_id, perm_code)
SELECT r.id, c FROM sys_roles r, (
    SELECT 'erp:stock:list' AS c UNION ALL SELECT 'erp:stock:edit'
    UNION ALL SELECT 'erp:master:list' UNION ALL SELECT 'erp:purchase:list'
    UNION ALL SELECT 'erp:sales:list' UNION ALL SELECT 'erp:return:list'
    UNION ALL SELECT 'tool:file:list'
) perms
WHERE r.code = 'warehouse_keeper'
ON DUPLICATE KEY UPDATE role_id = VALUES(role_id);

-- 采购：采购单据 + 主数据
INSERT INTO sys_role_perms (role_id, perm_code)
SELECT r.id, c FROM sys_roles r, (
    SELECT 'erp:master:list' AS c UNION ALL SELECT 'erp:master:edit'
    UNION ALL SELECT 'erp:purchase:list' UNION ALL SELECT 'erp:purchase:edit'
    UNION ALL SELECT 'erp:stock:list'
    UNION ALL SELECT 'erp:return:list' UNION ALL SELECT 'erp:return:edit'
) perms
WHERE r.code = 'buyer'
ON DUPLICATE KEY UPDATE role_id = VALUES(role_id);

-- 销售：销售单据 + 主数据
INSERT INTO sys_role_perms (role_id, perm_code)
SELECT r.id, c FROM sys_roles r, (
    SELECT 'erp:master:list' AS c UNION ALL SELECT 'erp:master:edit'
    UNION ALL SELECT 'erp:sales:list' UNION ALL SELECT 'erp:sales:edit'
    UNION ALL SELECT 'erp:stock:list'
    UNION ALL SELECT 'erp:return:list' UNION ALL SELECT 'erp:return:edit'
) perms
WHERE r.code = 'seller'
ON DUPLICATE KEY UPDATE role_id = VALUES(role_id);

-- 菜单授权：按角色能看到的最小集合
INSERT INTO sys_role_menus (role_id, menu_id)
SELECT r.id, m.id
FROM sys_roles r
JOIN sys_menus m ON m.title_key IN (
    'menu.erp.goods', 'menu.erp.customers', 'menu.erp.suppliers',
    'menu.erp.warehouses', 'menu.erp.classifications', 'menu.erp.stock',
    'menu.erp.transfers', 'menu.erp.stockChecks', 'menu.erp.reports',
    'menu.erp.purchase', 'menu.erp.sales', 'menu.erp.purchaseReturn', 'menu.erp.salesReturn'
)
WHERE r.code IN ('warehouse_manager', 'warehouse_keeper', 'buyer', 'seller')
ON DUPLICATE KEY UPDATE role_id = VALUES(role_id);

-- 审核中心两个待办页只给主管
INSERT INTO sys_role_menus (role_id, menu_id)
SELECT r.id, m.id
FROM sys_roles r
JOIN sys_menus m ON m.title_key IN ('menu.erp.auditIn', 'menu.erp.auditOut')
WHERE r.code = 'warehouse_manager'
ON DUPLICATE KEY UPDATE role_id = VALUES(role_id);
