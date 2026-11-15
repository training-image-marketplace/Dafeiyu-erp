// 有问题欢迎联系 hot_kun@hotmail.com
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"practical-training/erp-system/internal/domain"
	"practical-training/erp-system/internal/repository"
)

// SwitchService 解析并下发缺陷/菜单开关。
//
// 优先级：personal > group > all。当前实现按请求声明的单一作用域叠加 all；
// 平台如果要在一次请求里叠加 group + personal，应先把合并结果下发到对应作用域。
type SwitchService struct {
	repo *repository.ERPMasterRepository
}

func NewSwitchService(repo *repository.ERPMasterRepository) *SwitchService {
	return &SwitchService{repo: repo}
}

// Config 返回某作用域生效后的配置（all 打底，scope 覆盖）。
func (s *SwitchService) Config(ctx context.Context, scope string, scopeID int64) (domain.SwitchConfig, error) {
	cfg := domain.SwitchConfig{Bugs: map[string]bool{}, Menus: map[string]bool{}}

	layers := []struct {
		scope string
		id    int64
	}{{"all", 0}}
	if scope != "" && scope != "all" {
		layers = append(layers, struct {
			scope string
			id    int64
		}{scope, scopeID})
	}

	for _, l := range layers {
		raw, err := s.repo.GetSwitchConfig(ctx, l.scope, l.id)
		if errors.Is(err, domain.ErrNotFound) {
			continue
		}
		if err != nil {
			return domain.SwitchConfig{}, fmt.Errorf("load %s/%d switch: %w", l.scope, l.id, err)
		}
		var part domain.SwitchConfig
		if err := json.Unmarshal(raw, &part); err != nil {
			return domain.SwitchConfig{}, fmt.Errorf("parse %s/%d switch: %w", l.scope, l.id, err)
		}
		for k, v := range part.Bugs {
			cfg.Bugs[k] = v
		}
		for k, v := range part.Menus {
			cfg.Menus[k] = v
		}
	}
	return cfg, nil
}

// Set 覆盖某作用域的完整配置。
func (s *SwitchService) Set(ctx context.Context, scope string, scopeID int64, cfg domain.SwitchConfig) error {
	switch scope {
	case "all", "group", "personal":
	default:
		return fmt.Errorf("unknown scope %q: %w", scope, domain.ErrInvalidParent)
	}
	if scope == "all" {
		scopeID = 0
	}
	if scope != "all" && scopeID <= 0 {
		return fmt.Errorf("%s scope requires scope_id: %w", scope, domain.ErrInvalidParent)
	}
	if cfg.Bugs == nil {
		cfg.Bugs = map[string]bool{}
	}
	if cfg.Menus == nil {
		cfg.Menus = map[string]bool{}
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal switch: %w", err)
	}
	return s.repo.SetSwitchConfig(ctx, scope, scopeID, raw)
}

func (s *SwitchService) List(ctx context.Context) ([]repository.SwitchRecord, error) {
	return s.repo.ListSwitchConfigs(ctx)
}

func (s *SwitchService) BugEnabled(ctx context.Context, code string, scope string, scopeID int64) bool {
	cfg, err := s.Config(ctx, scope, scopeID)
	if err != nil {
		// 开关读失败时按「关闭」处理：宁可少一个缺陷，也不要让业务随机 500。
		return false
	}
	return cfg.Bugs[code]
}

func (s *SwitchService) MenuEnabled(ctx context.Context, code string, scope string, scopeID int64) bool {
	cfg, err := s.Config(ctx, scope, scopeID)
	if err != nil {
		return true // 菜单读失败时放行，避免整站不可用
	}
	// 未配置的菜单默认可见。
	if _, ok := cfg.Menus[code]; !ok {
		return true
	}
	return cfg.Menus[code]
}
