-- 0026 新增业务菜单：库存调拨、库存盘点、报表统计
--
-- 权限沿用现有码（erp:stock:list / erp:stock:edit / erp:audit:approve），
-- 不新增权限体系；这三个页面都是库存业务的延伸。

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 6, 'menu', 'menu.erp.transfers', '/erp/transfers', 'erp/transfers', 'arrow-left-right'
FROM sys_menus WHERE title_key = 'menu.erp.biz'
  AND NOT EXISTS (SELECT 1 FROM sys_menus WHERE title_key = 'menu.erp.transfers');

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 7, 'menu', 'menu.erp.stockChecks', '/erp/stock-checks', 'erp/stock-checks', 'clipboard-check'
FROM sys_menus WHERE title_key = 'menu.erp.biz'
  AND NOT EXISTS (SELECT 1 FROM sys_menus WHERE title_key = 'menu.erp.stockChecks');

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 8, 'menu', 'menu.erp.reports', '/erp/reports', 'erp/reports', 'chart-column'
FROM sys_menus WHERE title_key = 'menu.erp.biz'
  AND NOT EXISTS (SELECT 1 FROM sys_menus WHERE title_key = 'menu.erp.reports');

-- 已经能看到库存页的角色，继续能看到这三个新页面（同一批库存权限）。
INSERT INTO sys_role_menus (role_id, menu_id)
SELECT rm.role_id, m.id
FROM sys_role_menus rm
JOIN sys_menus s ON s.id = rm.menu_id AND s.title_key = 'menu.erp.stock'
JOIN sys_menus m ON m.title_key IN ('menu.erp.transfers', 'menu.erp.stockChecks', 'menu.erp.reports')
ON DUPLICATE KEY UPDATE role_id = VALUES(role_id);
