// 有问题欢迎联系 hot_kun@hotmail.com
package httpserver

import (
	"encoding/json"
	"net/http"
	"testing"
)

// 平台按 GET /internal/switches/schema 渲染开关面板：组名与默认值都由镜像自报。
// 这条契约一旦坏掉，平台会静默回退到 bugs/menus 两组，教师看到的面板就退化了。
func TestSwitchSchemaReportsGroups(t *testing.T) {
	r, _ := newTestRouter(t)

	w := call(t, r, http.MethodGet, "/internal/switches/schema", "", sessionInfo{})
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
