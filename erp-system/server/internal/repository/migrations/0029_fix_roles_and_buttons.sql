-- 0029 修正角色权限与退货页按钮元数据
--
-- 0027 播种的仓管员只有列表权限，但收货/发货/退货完成走的是各自 edit 码；
-- 审核员只有 audit:list/approve，审核中心详情还要读各单据，同样缺 list 码。
-- 这里补齐，并从退货页按钮里移除已迁到审核中心的审批动作。

INSERT INTO sys_role_perms (role_id, perm_code)
SELECT r.id, c FROM sys_roles r, (
    SELECT 'erp:purchase:edit' AS c
    UNION ALL SELECT 'erp:sales:edit'
    UNION ALL SELECT 'erp:return:edit'
) perms
WHERE r.code = 'warehouse_keeper'
ON DUPLICATE KEY UPDATE role_id = VALUES(role_id);

INSERT INTO sys_role_perms (role_id, perm_code)
SELECT r.id, c FROM sys_roles r, (
    SELECT 'erp:purchase:list' AS c
    UNION ALL SELECT 'erp:sales:list'
    UNION ALL SELECT 'erp:return:list'
    UNION ALL SELECT 'erp:stock:list'
) perms
WHERE r.code = 'auditor'
ON DUPLICATE KEY UPDATE role_id = VALUES(role_id);

-- 审核员需要两个待办页面；此前只给主管授权。
INSERT INTO sys_role_menus (role_id, menu_id)
SELECT r.id, m.id
FROM sys_roles r
JOIN sys_menus m ON m.title_key IN ('menu.erp.auditIn', 'menu.erp.auditOut')
WHERE r.code = 'auditor'
ON DUPLICATE KEY UPDATE menu_id = VALUES(menu_id);

-- 退货页只保留建单/提交/完成，审批入口已在审核中心。
UPDATE sys_menus
SET buttons_json = JSON_ARRAY(
    JSON_OBJECT('code', 'erp:return:edit', 'label_key', 'perm.erp.return.edit'))
WHERE title_key IN ('menu.erp.purchaseReturn', 'menu.erp.salesReturn')
  AND buttons_json IS NOT NULL;
