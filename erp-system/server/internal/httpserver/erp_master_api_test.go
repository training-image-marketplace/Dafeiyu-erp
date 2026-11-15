// 有问题欢迎联系 hot_kun@hotmail.com
package httpserver

import (
	"net/http"
	"testing"
)

// TestMasterListsReturnAllFields 防止「SELECT 加列但 Scan 没同步」这类 500。
// 每个主数据列表都必须能正常返回，且新字段出现在响应里。
func TestMasterListsReturnAllFields(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	setupMaster(t, r, admin)

	cases := []struct {
		path   string
		fields []string
	}{
		{"/api/v1/warehouses?page=1&page_size=200&status=active", []string{"keeper", "phone", "address"}},
		{"/api/v1/customers?page=1&page_size=200&status=active", []string{"address", "payment_method"}},
		{"/api/v1/suppliers?page=1&page_size=200&status=active", []string{"address", "payment_method"}},
		{"/api/v1/goods?page=1&page_size=200&status=active", []string{"spec", "unit_id", "purchase_price", "sale_price", "min_qty"}},
		{"/api/v1/units?status=active", []string{"name", "status"}},
	}
	for _, tc := range cases {
		w := call(t, r, http.MethodGet, tc.path, "", admin)
		if w.Code != http.StatusOK {
			t.Errorf("%s 返回 %d: %s", tc.path, w.Code, w.Body.String())
			continue
		}
		for _, f := range tc.fields {
			if !containsJSONKey(w.Body.String(), f) {
				t.Errorf("%s 的响应缺少字段 %q", tc.path, f)
			}
		}
	}
}

func containsJSONKey(body, key string) bool {
	for i := 0; i+len(key)+2 <= len(body); i++ {
		if body[i] == '"' && body[i+1:i+1+len(key)] == key && body[i+len(key)+1] == '"' {
			return true
		}
	}
	return false
}
