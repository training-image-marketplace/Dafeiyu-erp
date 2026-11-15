-- 0028 菜单按钮级权限：sys_menus 增加 buttons_json
--
-- 每个页面声明自己有哪些操作按钮（权限码 + 文案键），前端授权页按页面展示，
-- 角色授权仍然落到 sys_role_perms 的权限码上，接口侧用同一个码校验，
-- 因此「界面上能勾选」和「API 真正拦截」是同一套码。

ALTER TABLE sys_menus
    ADD COLUMN buttons_json JSON NULL AFTER icon;

-- 基础数据：新增/编辑/删除共用一个 master:edit
UPDATE sys_menus SET buttons_json = JSON_ARRAY(
    JSON_OBJECT('code', 'erp:master:edit', 'label_key', 'perm.erp.master.edit'))
WHERE title_key IN ('menu.erp.goods', 'menu.erp.customers', 'menu.erp.suppliers',
                    'menu.erp.warehouses', 'menu.erp.classifications');

-- 库存：调整单的新建/提交/删除
UPDATE sys_menus SET buttons_json = JSON_ARRAY(
    JSON_OBJECT('code', 'erp:stock:edit', 'label_key', 'perm.erp.stock.edit'))
WHERE title_key IN ('menu.erp.stock', 'menu.erp.transfers', 'menu.erp.stockChecks');

-- 采购/销售：建单改单提交 + 独立审批码
UPDATE sys_menus SET buttons_json = JSON_ARRAY(
    JSON_OBJECT('code', 'erp:purchase:edit', 'label_key', 'perm.erp.purchase.edit'))
WHERE title_key = 'menu.erp.purchase';

UPDATE sys_menus SET buttons_json = JSON_ARRAY(
    JSON_OBJECT('code', 'erp:sales:edit', 'label_key', 'perm.erp.sales.edit'))
WHERE title_key = 'menu.erp.sales';

-- 退货：建单 + 统一审核码
UPDATE sys_menus SET buttons_json = JSON_ARRAY(
    JSON_OBJECT('code', 'erp:return:edit', 'label_key', 'perm.erp.return.edit'),
    JSON_OBJECT('code', 'erp:audit:approve', 'label_key', 'perm.erp.audit.approve'))
WHERE title_key IN ('menu.erp.purchaseReturn', 'menu.erp.salesReturn');

-- 审核中心：只放审核动作
UPDATE sys_menus SET buttons_json = JSON_ARRAY(
    JSON_OBJECT('code', 'erp:audit:approve', 'label_key', 'perm.erp.audit.approve'))
WHERE title_key IN ('menu.erp.auditIn', 'menu.erp.auditOut');
