-- 0019 ERP 仪表盘菜单
--
-- 仪表盘是正常菜单项，不是特殊路由：排序 0 放在最前，并授权给所有已有角色。
-- 前端仍保留一份内置路由兜底，给「角色后来新建、还没配菜单」的账号用，
-- 与 training-platform 的 dashboard 处理方式一致。
--
-- 注意：它挂在根目录，不占 ERP 业务目录的子菜单位。
INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT NULL, 0, 'menu', 'menu.erp.dashboard', '/erp/dashboard', 'erp/dashboard', 'layout-dashboard'
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM sys_menus WHERE title_key = 'menu.erp.dashboard');

INSERT INTO sys_role_menus (role_id, menu_id)
SELECT r.id, m.id
FROM sys_roles r
JOIN sys_menus m ON m.title_key = 'menu.erp.dashboard'
WHERE NOT EXISTS (
    SELECT 1 FROM sys_role_menus rm WHERE rm.role_id = r.id AND rm.menu_id = m.id
);
