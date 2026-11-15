-- 0020 ERP 菜单分组
--
-- 原来 ERP 业务目录下 11 个页面平铺，基础数据、业务单据、审核混在一起。
-- 这里在 ERP 业务目录下再分两层：
--
--   ERP 业务
--   ├── 基础数据（商品/客户/供应商/仓库/分类品牌）
--   ├── ERP 数据（库存/采购/销售/采购退货/销售退货）
--   └── 审核中心
--
-- 菜单树本身支持任意层级（buildMenuTree 是递归的），前端侧边栏这次也改成
-- 递归渲染；在此之前只渲染一层目录。

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 1, 'directory', 'menu.erp.basic', '/erp/basic', '', 'database'
FROM sys_menus WHERE title_key = 'menu.erp'
  AND NOT EXISTS (SELECT 1 FROM sys_menus WHERE title_key = 'menu.erp.basic');

INSERT INTO sys_menus (parent_id, sort, menu_type, title_key, path, component, icon)
SELECT id, 2, 'directory', 'menu.erp.biz', '/erp/biz', '', 'boxes'
FROM sys_menus WHERE title_key = 'menu.erp'
  AND NOT EXISTS (SELECT 1 FROM sys_menus WHERE title_key = 'menu.erp.biz');

-- 基础数据：五个档案页面
UPDATE sys_menus c JOIN sys_menus p ON p.title_key = 'menu.erp.basic'
SET c.parent_id = p.id, c.sort = 1
WHERE c.title_key = 'menu.erp.goods';

UPDATE sys_menus c JOIN sys_menus p ON p.title_key = 'menu.erp.basic'
SET c.parent_id = p.id, c.sort = 2
WHERE c.title_key = 'menu.erp.customers';

UPDATE sys_menus c JOIN sys_menus p ON p.title_key = 'menu.erp.basic'
SET c.parent_id = p.id, c.sort = 3
WHERE c.title_key = 'menu.erp.suppliers';

UPDATE sys_menus c JOIN sys_menus p ON p.title_key = 'menu.erp.basic'
SET c.parent_id = p.id, c.sort = 4
WHERE c.title_key = 'menu.erp.warehouses';

UPDATE sys_menus c JOIN sys_menus p ON p.title_key = 'menu.erp.basic'
SET c.parent_id = p.id, c.sort = 5
WHERE c.title_key = 'menu.erp.classifications';

-- ERP 数据：库存与四类单据
UPDATE sys_menus c JOIN sys_menus p ON p.title_key = 'menu.erp.biz'
SET c.parent_id = p.id, c.sort = 1
WHERE c.title_key = 'menu.erp.stock';

UPDATE sys_menus c JOIN sys_menus p ON p.title_key = 'menu.erp.biz'
SET c.parent_id = p.id, c.sort = 2
WHERE c.title_key = 'menu.erp.purchase';

UPDATE sys_menus c JOIN sys_menus p ON p.title_key = 'menu.erp.biz'
SET c.parent_id = p.id, c.sort = 3
WHERE c.title_key = 'menu.erp.sales';

UPDATE sys_menus c JOIN sys_menus p ON p.title_key = 'menu.erp.biz'
SET c.parent_id = p.id, c.sort = 4
WHERE c.title_key = 'menu.erp.purchaseReturn';

UPDATE sys_menus c JOIN sys_menus p ON p.title_key = 'menu.erp.biz'
SET c.parent_id = p.id, c.sort = 5
WHERE c.title_key = 'menu.erp.salesReturn';

-- 审核从业务单据里独立出来，直接挂在 ERP 业务下。
UPDATE sys_menus c JOIN sys_menus p ON p.title_key = 'menu.erp'
SET c.parent_id = p.id, c.sort = 3
WHERE c.title_key = 'menu.erp.audit';
