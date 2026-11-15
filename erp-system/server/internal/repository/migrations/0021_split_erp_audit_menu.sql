-- 0021 审核待办拆分为入库/出库两个菜单
--
-- 原来「审核中心」一个页面里用按钮切入库/出库。审核员每天只盯一个方向，
-- 切换既多一步，也无法给「只审入库」和「只审出库」分别授权。
--
-- 这里把 menu.erp.audit 转成目录「审核中心」，下面挂两个页面菜单：
--   入库待办 /erp/audit/in  -> 采购、采购退货
--   出库待办 /erp/audit/out -> 销售、销售退货

UPDATE sys_menus
SET menu_type = 'directory',
    title_key = 'menu.erp.audit',
    path = '/erp/audit',
    component = '',
    icon = 'clipboard-check',
    sort = 3
WHERE title_key = 'menu.erp.audit';

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 1, 'menu', 'menu.erp.auditIn', '/erp/audit/in', 'erp/audit-in', 'inbox'
FROM sys_menus WHERE title_key = 'menu.erp.audit'
  AND NOT EXISTS (SELECT 1 FROM sys_menus WHERE title_key = 'menu.erp.auditIn');

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 2, 'menu', 'menu.erp.auditOut', '/erp/audit/out', 'erp/audit-out', 'send'
FROM sys_menus WHERE title_key = 'menu.erp.audit'
  AND NOT EXISTS (SELECT 1 FROM sys_menus WHERE title_key = 'menu.erp.auditOut');

-- 原来授权了「审核中心」的角色，继续能看到两个方向的待办。
INSERT INTO sys_role_menus (role_id, menu_id)
SELECT rm.role_id, c.id
FROM sys_role_menus rm
JOIN sys_menus d ON d.id = rm.menu_id AND d.title_key = 'menu.erp.audit'
JOIN sys_menus c ON c.title_key IN ('menu.erp.auditIn', 'menu.erp.auditOut')
ON DUPLICATE KEY UPDATE role_id = VALUES(role_id);
