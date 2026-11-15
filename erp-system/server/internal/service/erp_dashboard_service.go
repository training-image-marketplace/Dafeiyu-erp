// 有问题欢迎联系 hot_kun@hotmail.com
package service

import (
	"context"
	"fmt"
	"sort"

	"practical-training/erp-system/internal/domain"
	"practical-training/erp-system/internal/repository"
)

// dashboardPendingLimit 是仪表盘待办展示条数。
//
// 仪表盘只负责「有没有待办、大概是什么」，完整列表和筛选在审核中心，
// 所以不在首页做分页，多出来的直接截断。
const dashboardPendingLimit = 5

// Dashboard 汇总首页指标。
//
// 任何登录角色都可以调用：这里只有计数与待办公开字段，不返回成本、
// 库存明细等敏感数据，也不需要「看单据列表」的权限码。
func (s *ERPService) Dashboard(ctx context.Context) (domain.Dashboard, error) {
	goods, err := s.master.CountGoods(ctx, repository.MasterFilter{})
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("count goods: %w", err)
	}
	customers, err := s.master.CountCustomers(ctx, repository.MasterFilter{})
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("count customers: %w", err)
	}
	suppliers, err := s.master.CountSuppliers(ctx, repository.MasterFilter{})
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("count suppliers: %w", err)
	}
	warehouses, err := s.master.CountWarehouses(ctx, repository.MasterFilter{})
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("count warehouses: %w", err)
	}
	stockGoods, stockTotal, err := s.master.StockTotals(ctx)
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("stock totals: %w", err)
	}

	alerts, err := s.master.CountStockAlerts(ctx)
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("count stock alerts: %w", err)
	}

	filter := repository.AuditFilter{Status: "submitted"}
	pendingIn, err := s.orders.CountAuditBills(ctx, "in", filter)
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("count inbound pending: %w", err)
	}
	pendingOut, err := s.orders.CountAuditBills(ctx, "out", filter)
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("count outbound pending: %w", err)
	}

	// 两个方向各取前 N 条，再按提交时间合并排序。
	// 两次查询都走审核视图的现有索引，不需要为首页再加一套 UNION。
	inList, err := s.orders.ListAuditBills(ctx, "in", filter, dashboardPendingLimit, 0)
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("list inbound pending: %w", err)
	}
	outList, err := s.orders.ListAuditBills(ctx, "out", filter, dashboardPendingLimit, 0)
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("list outbound pending: %w", err)
	}
	pending := append(inList, outList...)
	sort.SliceStable(pending, func(i, j int) bool {
		a, b := pending[i].SubmittedAt, pending[j].SubmittedAt
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		return a.After(*b)
	})
	if len(pending) > dashboardPendingLimit {
		pending = pending[:dashboardPendingLimit]
	}
	if pending == nil {
		pending = []domain.AuditBill{}
	}

	return domain.Dashboard{
		GoodsCount:      goods,
		CustomerCount:   customers,
		SupplierCount:   suppliers,
		WarehouseCount:  warehouses,
		StockGoodsCount: stockGoods,
		StockTotal:      stockTotal,
		PendingTotal:    pendingIn + pendingOut,
		PendingIn:       pendingIn,
		PendingOut:      pendingOut,
		StockAlerts:     alerts,
		Pending:         pending,
	}, nil
}
