// 有问题欢迎联系 hot_kun@hotmail.com
package domain

import "time"

// Role 对应 sys_roles 表。
type Role struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Status    string    `json:"status"`
	Code      string    `json:"code"` // 稳定标识，如 admin
	Name      string    `json:"name"`
	Remark    string    `json:"remark"`
}

func (r *Role) IsBuiltin() bool { return r.Code == RoleCodeAdmin }

// MenuButton 是页面上的一个操作按钮。Code 与接口权限码一致，
// LabelKey 是 i18n 键——授权页据此渲染勾选项，不再另造一套按钮码。
type MenuButton struct {
	Code     string `json:"code"`
	LabelKey string `json:"label_key"`
}

// Menu 对应 sys_menus 表。ParentID 为 nil 表示根节点。
type Menu struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Status    string    `json:"status"`
	ParentID  *int64    `json:"parent_id"`
	Sort      int       `json:"sort"`
	MenuType  string    `json:"menu_type"`
	TitleKey  string    `json:"title_key"` // i18n 键，文案由前端渲染
	Path      string    `json:"path"`
	Component string    `json:"component"`
	Icon      string    `json:"icon"`
	// Buttons 是该页面的按钮权限，逐条授予；空数组表示无按钮。
	Buttons []MenuButton `json:"buttons"`
}

// MenuNode 是菜单树节点。children 为空时序列化成 []，前端不必特判 null。
type MenuNode struct {
	Menu
	Children []*MenuNode `json:"children"`
}
