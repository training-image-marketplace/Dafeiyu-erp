-- 0030 缺陷开关页与 Android 客户端页
--
-- 两页都挂在「ERP 业务」目录下：
--   /erp/switches 缺陷注入开关，需要 erp:switch:list，默认只有内置管理员有
--   /erp/android  Android 客户端介绍与 APK 下载，任何登录用户都能看
--
-- 开关页的按钮码与接口权限码是同一个（erp:switch:edit），
-- 授权页勾选它等于放开 /switches/bug 的写权限。

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon, buttons_json)
SELECT id, 4, 'menu', 'menu.erp.switches', '/erp/switches', 'erp/switches', 'toggle-left',
       JSON_ARRAY(JSON_OBJECT('code', 'erp:switch:edit', 'label_key', 'perm.erp.switch.edit'))
FROM sys_menus WHERE title_key = 'menu.erp'
  AND NOT EXISTS (SELECT 1 FROM sys_menus WHERE title_key = 'menu.erp.switches');

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon, buttons_json)
SELECT id, 5, 'menu', 'menu.erp.android', '/erp/android', 'erp/android', 'smartphone', JSON_ARRAY()
FROM sys_menus WHERE title_key = 'menu.erp'
  AND NOT EXISTS (SELECT 1 FROM sys_menus WHERE title_key = 'menu.erp.android');

-- 演示角色也能看到 Android 下载入口；开关页只留给管理员（内置管理员天然可见全部菜单）
INSERT INTO sys_role_menus (role_id, menu_id)
SELECT r.id, m.id
FROM sys_roles r JOIN sys_menus m ON m.title_key = 'menu.erp.android'
WHERE r.code IN ('warehouse_manager', 'warehouse_keeper', 'buyer', 'seller')
ON DUPLICATE KEY UPDATE role_id = VALUES(role_id);
