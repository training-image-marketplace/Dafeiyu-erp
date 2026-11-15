-- 0005 API 权限清单菜单
--
-- 这个页面是只读的：权限点由代码声明（internal/perm），界面不提供新增与修改。
-- 新增一个权限点必须同时有「代码声明 + 路由引用 + 后端校验」三样东西。

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 4, 'menu', 'menu.system.apis', '/system/apis', 'system/apis', 'plug'
FROM sys_menus WHERE title_key = 'menu.system';
