-- 0015 ERP 业务菜单
--
-- 第 4 个根目录，下挂主数据、库存、采购、销售四个页面。
-- component 与 web/src/pages 下的文件一一对应，menu_seed_test 会校验。

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT NULL, 4, 'directory', 'menu.erp', '/erp', '', 'package' FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM sys_menus WHERE title_key = 'menu.erp');

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 1, 'menu', 'menu.erp.master', '/erp/master', 'erp/master', 'database'
FROM sys_menus WHERE title_key = 'menu.erp';

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 2, 'menu', 'menu.erp.stock', '/erp/stock', 'erp/stock', 'boxes'
FROM sys_menus WHERE title_key = 'menu.erp';

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 3, 'menu', 'menu.erp.purchase', '/erp/purchase', 'erp/purchase', 'shopping-cart'
FROM sys_menus WHERE title_key = 'menu.erp';

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 4, 'menu', 'menu.erp.sales', '/erp/sales', 'erp/sales', 'trending-up'
FROM sys_menus WHERE title_key = 'menu.erp';
