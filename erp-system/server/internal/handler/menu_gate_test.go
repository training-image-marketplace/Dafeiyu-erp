// 有问题欢迎联系 hot_kun@hotmail.com
package handler

import "testing"

// TestMenuCodeOfPath 固定「路径 → 菜单 key」的映射。
//
// 这里最容易踩的坑是 sales 与 sales-returns 共享前缀：历史上的前缀匹配 + map
// 迭代顺序会让两者随机串味，关掉销售退货却把销售也拦了（或反之）。用用例
// 把每个资源段都钉死，谁改回前缀匹配就会红。
func TestMenuCodeOfPath(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/api/v1/goods", "goods"},
		{"/api/v1/goods/12", "goods"},
		{"/api/v1/categories/3", "goods"},
		{"/api/v1/brands", "goods"},
		{"/api/v1/units/1", "goods"},
		{"/api/v1/customers", "customers"},
		{"/api/v1/suppliers/7", "suppliers"},
		{"/api/v1/warehouses", "warehouses"},
		{"/api/v1/stocks", "stock"},
		{"/api/v1/stocks?logs=1", "stock"},
		{"/api/v1/stock-adjustments/4/submit", "stock"},
		{"/api/v1/transfers", "stock"},
		{"/api/v1/stock-checks/2", "stock"},
		{"/api/v1/reports/stock-flow", "reports"},
		{"/api/v1/purchases/9/approve", "purchase"},
		{"/api/v1/purchase-returns", "purchase_return"},
		// 下面两条是本次修复的核心：彼此前缀重叠，必须各归各的 key。
		{"/api/v1/sales", "sales"},
		{"/api/v1/sales/5/ship", "sales"},
		{"/api/v1/sales-returns", "sales_return"},
		{"/api/v1/sales-returns/5/complete", "sales_return"},
		{"/api/v1/audits", "audit"},
		{"/api/v1/audits/purchase/1", "audit"},
		// 不归任何菜单管的路径：不能让它们被误拦。
		{"/api/v1/auth/me", ""},
		{"/api/v1/files", ""},
		{"/internal/switches", ""},
		{"/healthz", ""},
		{"/api/v1/", ""},
	}

	for _, tc := range cases {
		if got := menuCodeOfPath(tc.path); got != tc.want {
			t.Errorf("menuCodeOfPath(%q) = %q，期望 %q", tc.path, got, tc.want)
		}
	}
}
