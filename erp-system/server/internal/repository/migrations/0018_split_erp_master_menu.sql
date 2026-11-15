-- 0018 拆分 ERP 主数据菜单
--
-- 原先把商品/客户/供应商/仓库/分类品牌挤在一个「主数据」页的 Tab 里，
-- 导航上只有一个入口，找具体档案要先点进去再切 Tab。这里改成五个独立菜单，
-- component 与 web/src/pages/erp/*.svelte 一一对应。
--
-- 仪表盘是内置路由（lib/router.svelte.ts 的 builtinRoutes），不写菜单：
-- 它要对所有角色可见，而菜单需要逐角色授权，做不到「所有角色」。
--
-- 迁移只向前：先把旧的 menu.erp.master 原地改成「商品管理」，再补另外四个。

UPDATE sys_menus
SET title_key = 'menu.erp.goods',
    path = '/erp/goods',
    component = 'erp/goods',
    icon = 'package',
    sort = 1
WHERE title_key = 'menu.erp.master';

-- 兜底：如果旧菜单被人手工删过，也保证商品管理仍然存在。
INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 1, 'menu', 'menu.erp.goods', '/erp/goods', 'erp/goods', 'package'
FROM sys_menus WHERE title_key = 'menu.erp'
  AND NOT EXISTS (SELECT 1 FROM sys_menus WHERE title_key = 'menu.erp.goods');

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 2, 'menu', 'menu.erp.customers', '/erp/customers', 'erp/customers', 'users'
FROM sys_menus WHERE title_key = 'menu.erp'
  AND NOT EXISTS (SELECT 1 FROM sys_menus WHERE title_key = 'menu.erp.customers');

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 3, 'menu', 'menu.erp.suppliers', '/erp/suppliers', 'erp/suppliers', 'truck'
FROM sys_menus WHERE title_key = 'menu.erp'
  AND NOT EXISTS (SELECT 1 FROM sys_menus WHERE title_key = 'menu.erp.suppliers');

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 4, 'menu', 'menu.erp.warehouses', '/erp/warehouses', 'erp/warehouses', 'warehouse'
FROM sys_menus WHERE title_key = 'menu.erp'
  AND NOT EXISTS (SELECT 1 FROM sys_menus WHERE title_key = 'menu.erp.warehouses');

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 5, 'menu', 'menu.erp.classifications', '/erp/classifications', 'erp/classifications', 'tags'
FROM sys_menus WHERE title_key = 'menu.erp'
  AND NOT EXISTS (SELECT 1 FROM sys_menus WHERE title_key = 'menu.erp.classifications');

-- 原来只授权了「主数据」角色的，迁移后应当继续能看全部五个主数据页面。
-- 旧菜单行已原地变成商品管理，所以从商品管理的授权出发复制到另外四个。
INSERT INTO sys_role_menus (role_id, menu_id)
SELECT rm.role_id, m.id
FROM sys_role_menus rm
JOIN sys_menus g ON g.id = rm.menu_id AND g.title_key = 'menu.erp.goods'
JOIN sys_menus m ON m.title_key IN (
    'menu.erp.customers', 'menu.erp.suppliers', 'menu.erp.warehouses', 'menu.erp.classifications'
)
ON DUPLICATE KEY UPDATE role_id = VALUES(role_id);

-- 其余业务菜单顺延，保持在 ERP 目录下的稳定顺序。
UPDATE sys_menus SET sort = 6 WHERE title_key = 'menu.erp.stock';
UPDATE sys_menus SET sort = 7 WHERE title_key = 'menu.erp.purchase';
UPDATE sys_menus SET sort = 8 WHERE title_key = 'menu.erp.sales';
UPDATE sys_menus SET sort = 9 WHERE title_key = 'menu.erp.purchaseReturn';
UPDATE sys_menus SET sort = 10 WHERE title_key = 'menu.erp.salesReturn';
UPDATE sys_menus SET sort = 11 WHERE title_key = 'menu.erp.audit';
