-- 0017 退货与审核中心菜单
--
-- 三个页面都挂在「ERP 业务」下，component 与 web/src/pages 一一对应，
-- menu_seed_test 会校验页面、菜单种子与双语键三者一致。

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 5, 'menu', 'menu.erp.purchaseReturn', '/erp/purchase-return', 'erp/purchase-return', 'undo-2'
FROM sys_menus WHERE title_key = 'menu.erp';

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 6, 'menu', 'menu.erp.salesReturn', '/erp/sales-return', 'erp/sales-return', 'rotate-ccw'
FROM sys_menus WHERE title_key = 'menu.erp';

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 7, 'menu', 'menu.erp.audit', '/erp/audit', 'erp/audit', 'clipboard-check'
FROM sys_menus WHERE title_key = 'menu.erp';
