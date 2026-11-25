// 有问题欢迎联系 hot_kun@hotmail.com
package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 平台按 GET /internal/switches/schema 渲染开关面板：组名与默认值都由镜像自报。
// 这条契约一旦坏掉，平台会静默回退到 bugs/menus 两组，教师看到的面板就退化了。
// The platform sends the exact groups advertised by the schema, not a flattened bugs map.
func TestSwitchSchemaGroupsApplyToBusinessBehavior(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")
	supplierID, warehouseID, goodsID, _ := setupMaster(t, r, admin)
	internal := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Internal-Token", "test-internal-token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	payload := `{"scope":"all","scope_id":0,"config":{"bugs-docs":{"BUG-001":true,"BUG-005":true},"bugs-approval":{"BUG-003":true},"bugs-stock":{"BUG-002":true},"menus":{"sales":false}}}`
	w := internal(http.MethodPut, "/internal/switches", payload)
	if w.Code != http.StatusOK {
		t.Fatalf("apply schema groups: %d %s", w.Code, w.Body.String())
	}
	w = internal(http.MethodGet, "/internal/switches", "")
	var current struct {
		Config struct {
			Bugs  map[string]bool `json:"bugs"`
			Menus map[string]bool `json:"menus"`
		} `json:"config"`
	}
	decodeData(t, w.Body.Bytes(), &current)
	for _, code := range []string{"BUG-001", "BUG-002", "BUG-003", "BUG-005"} {
		if !current.Config.Bugs[code] {
			t.Errorf("schema group lost defect %s: %s", code, w.Body.String())
		}
	}
	body := fmt.Sprintf(`{"supplier_id":%d,"warehouse_id":%d,"items":[{"goods_id":%d,"qty":10,"price":5}]}`, supplierID, warehouseID, goodsID)
	w = call(t, r, http.MethodPost, "/api/v1/purchases", body, admin)
	if w.Code != http.StatusOK {
		t.Fatalf("create purchase: %d %s", w.Code, w.Body.String())
	}
	var purchase struct {
		TotalAmount float64 `json:"total_amount"`
	}
	decodeData(t, w.Body.Bytes(), &purchase)
	if purchase.TotalAmount != 49.99 {
		t.Fatalf("grouped BUG-001 must affect business total: got %.2f want 49.99", purchase.TotalAmount)
	}
	w = call(t, r, http.MethodGet, "/api/v1/sales", "", admin)
	if w.Code != http.StatusForbidden {
		t.Fatalf("grouped menus gate: %d", w.Code)
	}
	// A grouped false overrides a stale legacy true, avoiding a defect that cannot be disabled.
	w = internal(http.MethodPut, "/internal/switches", `{"scope":"all","config":{"bugs":{"BUG-001":true},"bugs-docs":{"BUG-001":false}}}`)
	if w.Code != http.StatusOK {
		t.Fatal(w.Body.String())
	}
	w = call(t, r, http.MethodPost, "/api/v1/purchases", body, admin)
	decodeData(t, w.Body.Bytes(), &purchase)
	if purchase.TotalAmount != 50 {
		t.Fatalf("grouped false did not disable legacy defect: %.2f", purchase.TotalAmount)
	}
	// Legacy payloads remain accepted for the ERP's own switch UI.
	w = internal(http.MethodPut, "/internal/switches", `{"scope":"all","config":{"bugs":{"BUG-001":true},"menus":{"sales":true}}}`)
	if w.Code != http.StatusOK {
		t.Fatal(w.Body.String())
	}
	w = call(t, r, http.MethodPost, "/api/v1/purchases", body, admin)
	decodeData(t, w.Body.Bytes(), &purchase)
	if purchase.TotalAmount != 49.99 {
		t.Fatalf("legacy defect payload regressed: %.2f", purchase.TotalAmount)
	}
}

func TestSwitchSchemaReportsGroups(t *testing.T) {
	r, _ := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/internal/switches/schema", nil)
	req.Header.Set("X-Internal-Token", "test-internal-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			List []struct {
				Code    string `json:"code"`
				Name    string `json:"name"`
				NameEN  string `json:"name_en"`
				Default *bool  `json:"default"`
				Items   []struct {
					Code string `json:"code"`
					Name string `json:"name"`
				} `json:"items"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v, body = %s", err, w.Body.String())
	}

	groups := resp.Data.List
	if len(groups) < 2 {
		t.Fatalf("期望至少 2 个开关组，实际 %d：%+v", len(groups), groups)
	}

	byCode := make(map[string]int, len(groups))
	for i, g := range groups {
		if g.Code == "" || g.Name == "" {
			t.Fatalf("组缺少 code/name: %+v", g)
		}
		if len(g.Items) == 0 {
			t.Fatalf("出现空组 %q，平台面板里会是空白项", g.Code)
		}
		byCode[g.Code] = i
	}

	// 缺陷默认关、菜单默认开：这是平台按「清单里有但没配过」时的行为
	if i, ok := byCode["bugs-docs"]; !ok {
		t.Fatalf("缺少单据类缺陷组，实际组：%v", byCode)
	} else if d := groups[i].Default; d == nil || *d {
		t.Fatalf("单据类缺陷组 default 期望 false，实际 %v", d)
	}
	if i, ok := byCode["menus"]; !ok {
		t.Fatalf("缺少菜单组，实际组：%v", byCode)
	} else if d := groups[i].Default; d == nil || !*d {
		t.Fatalf("菜单组 default 期望 true，实际 %v", d)
	}

	// 已接线的缺陷必须都能在 schema 里找到，否则教师无法在平台上打开它
	wantBugs := []string{"BUG-001", "BUG-002", "BUG-003", "BUG-005"}
	found := map[string]bool{}
	for _, g := range groups {
		for _, it := range g.Items {
			found[it.Code] = true
		}
	}
	for _, code := range wantBugs {
		if !found[code] {
			t.Errorf("schema 缺少已接线的缺陷 %s", code)
		}
	}

	// 菜单项也要齐全
	for _, code := range []string{"goods", "customers", "stock", "audit"} {
		if !found[code] {
			t.Errorf("schema 缺少菜单开关 %s", code)
		}
	}
}
