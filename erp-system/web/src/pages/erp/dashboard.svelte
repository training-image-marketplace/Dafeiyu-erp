<!-- 有问题欢迎联系 hot_kun@hotmail.com -->
<script lang="ts">
  import { onMount } from 'svelte'

  import AlertTriangle from '@lucide/svelte/icons/triangle-alert'
  import Boxes from '@lucide/svelte/icons/boxes'
  import ClipboardList from '@lucide/svelte/icons/clipboard-list'
  import Package from '@lucide/svelte/icons/package'
  import RefreshCw from '@lucide/svelte/icons/refresh-cw'
  import Truck from '@lucide/svelte/icons/truck'
  import Users from '@lucide/svelte/icons/users'
  import Warehouse from '@lucide/svelte/icons/warehouse'

  import { api } from '$lib/api/client'
  import type { AuditBillType, Dashboard } from '$lib/api/types'
  import { Badge } from '$lib/components/ui/badge'
  import { Button } from '$lib/components/ui/button'
  import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
  } from '$lib/components/ui/table'
  import { t, type MessageKey } from '$lib/i18n/index.svelte'
  import { errKey } from '$lib/list-state.svelte'
  import { navigate } from '$lib/router.svelte'
  import { notifyError } from '$lib/stores/notify.svelte'
  import { session } from '$lib/stores/session.svelte'

  let stats = $state<Dashboard | null>(null)
  let loading = $state(true)
  let loadError = $state('')

  async function load() {
    loading = true
    loadError = ''
    try {
      stats = await api.get<Dashboard>('/dashboard')
    } catch (err) {
      loadError = t(errKey(err))
      notifyError(errKey(err))
    } finally {
      loading = false
    }
  }

  onMount(load)

  const fmtQty = (n: number) => (Number.isInteger(n) ? String(n) : n.toFixed(2))
  const shortTime = (s: string | null) => (s ? s.replace('T', ' ').slice(0, 16) : '—')
  const typeKey = (tp: AuditBillType): MessageKey => `erp.audit.type.${tp}` as MessageKey

  const cards = $derived(
    stats
      ? [
          {
            label: 'dashboard.goods' as MessageKey,
            value: fmtQty(stats.goods_count),
            icon: Package,
            tone: 'bg-sky-50 text-sky-600 dark:bg-sky-500/10 dark:text-sky-400',
          },
          {
            label: 'dashboard.customers' as MessageKey,
            value: fmtQty(stats.customer_count),
            icon: Users,
            tone: 'bg-violet-50 text-violet-600 dark:bg-violet-500/10 dark:text-violet-400',
          },
          {
            label: 'dashboard.suppliers' as MessageKey,
            value: fmtQty(stats.supplier_count),
            icon: Truck,
            tone: 'bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-400',
          },
          {
            label: 'dashboard.warehouses' as MessageKey,
            value: fmtQty(stats.warehouse_count),
            icon: Warehouse,
            tone: 'bg-rose-50 text-rose-600 dark:bg-rose-500/10 dark:text-rose-400',
          },
        ]
      : [],
  )

  const quickLinks = $derived(
    [
      { label: 'menu.erp.purchase' as MessageKey, path: '/erp/purchase', perm: 'erp:purchase:list' },
      { label: 'menu.erp.sales' as MessageKey, path: '/erp/sales', perm: 'erp:sales:list' },
      { label: 'menu.erp.stock' as MessageKey, path: '/erp/stock', perm: 'erp:stock:list' },
      { label: 'menu.erp.auditIn' as MessageKey, path: '/erp/audit/in', perm: 'erp:audit:list' },
      { label: 'menu.erp.auditOut' as MessageKey, path: '/erp/audit/out', perm: 'erp:audit:list' },
    ].filter((link) => session.can(link.perm)),
  )
</script>

<div class="flex flex-col gap-6">
  <!-- 欢迎区：用与登录页一致的绿色系，给首页一个明确的主视觉 -->
  <section
    class="relative overflow-hidden rounded-xl border border-emerald-700/20 bg-gradient-to-br from-emerald-700 via-teal-700 to-cyan-800 p-6 text-white shadow-sm"
  >
    <div
      class="pointer-events-none absolute -top-24 -right-16 size-64 rounded-full bg-white/10 blur-2xl"
      aria-hidden="true"
    ></div>
    <div class="relative flex flex-wrap items-end justify-between gap-4">
      <div>
        <p class="text-xs font-medium tracking-[0.2em] text-emerald-100/80 uppercase">
          {t('dashboard.eyebrow')}
        </p>
        <h2 class="mt-2 text-2xl font-semibold">
          {t('dashboard.welcome', { name: session.user?.nickname || session.user?.username || '' })}
        </h2>
        <p class="mt-1.5 max-w-xl text-sm text-emerald-50/80">{t('dashboard.subtitle')}</p>
      </div>
      <div class="flex items-center gap-3">
        <div class="rounded-lg border border-white/15 bg-white/10 px-4 py-2.5 backdrop-blur-sm">
          <p class="text-xs text-emerald-50/75">{t('dashboard.pending')}</p>
          <p class="mt-0.5 text-2xl font-semibold tabular-nums">{stats?.pending_total ?? '—'}</p>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="border-white/25 bg-white/10 text-white hover:bg-white/20 hover:text-white"
          onclick={load}
          disabled={loading}
        >
          <RefreshCw class="size-4" />
          {t('dashboard.refresh')}
        </Button>
      </div>
    </div>
  </section>

  {#if loading && !stats}
    <div class="flex flex-col gap-3">
      <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {#each Array(4) as _, i (i)}
          <div class="h-28 animate-pulse rounded-xl border bg-card"></div>
        {/each}
      </div>
      <p class="py-8 text-center text-sm text-muted-foreground">{t('common.loading')}</p>
    </div>
  {:else if loadError && !stats}
    <div class="rounded-xl border border-dashed bg-card p-10 text-center">
      <p class="text-sm text-muted-foreground">{loadError}</p>
      <Button type="button" variant="outline" size="sm" class="mt-3" onclick={load}>
        {t('dashboard.retry')}
      </Button>
    </div>
  {:else if stats}
    <!-- 主数据概览 -->
    <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      {#each cards as card (card.label)}
        <div class="rounded-xl border bg-card p-4 transition-shadow hover:shadow-sm">
          <div class="flex items-start justify-between gap-3">
            <div>
              <p class="text-xs text-muted-foreground">{t(card.label)}</p>
              <p class="mt-2 text-2xl font-semibold tabular-nums">{card.value}</p>
            </div>
            <span class="grid size-10 place-items-center rounded-lg {card.tone}">
              <card.icon class="size-5" />
            </span>
          </div>
        </div>
      {/each}
    </div>

    <!-- 库存与待办 -->
    <div class="grid gap-3 md:grid-cols-3">
      <div class="rounded-xl border bg-card p-4">
        <div class="flex items-center justify-between">
          <p class="text-xs text-muted-foreground">{t('dashboard.stockTotal')}</p>
          <Boxes class="size-4 text-muted-foreground" />
        </div>
        <p class="mt-2 text-2xl font-semibold tabular-nums">{fmtQty(stats.stock_total)}</p>
        <p class="mt-1 text-xs text-muted-foreground">{t('dashboard.stockHint')}</p>
      </div>
      <div class="rounded-xl border bg-card p-4">
        <div class="flex items-center justify-between">
          <p class="text-xs text-muted-foreground">{t('dashboard.stockGoods')}</p>
          <Package class="size-4 text-muted-foreground" />
        </div>
        <p class="mt-2 text-2xl font-semibold tabular-nums">{stats.stock_goods_count}</p>
        <p class="mt-1 text-xs text-muted-foreground">{t('dashboard.stockGoodsHint')}</p>
      </div>
      <div class="rounded-xl border bg-card p-4">
        <div class="flex items-center justify-between">
          <p class="text-xs text-muted-foreground">{t('dashboard.pending')}</p>
          <ClipboardList class="size-4 text-muted-foreground" />
        </div>
        <p class="mt-2 text-2xl font-semibold tabular-nums">{stats.pending_total}</p>
        <p class="mt-1 text-xs text-muted-foreground">
          {t('dashboard.pendingSplit', { in: stats.pending_in, out: stats.pending_out })}
        </p>
      </div>
      <div class="rounded-xl border bg-card p-4">
        <div class="flex items-center justify-between">
          <p class="text-xs text-muted-foreground">{t('dashboard.stockAlerts')}</p>
          <AlertTriangle class="size-4 text-muted-foreground" />
        </div>
        <p class="mt-2 text-2xl font-semibold tabular-nums">{stats.stock_alerts}</p>
        <p class="mt-1 text-xs text-muted-foreground">{t('dashboard.stockAlertsHint')}</p>
      </div>
    </div>

    <div class="grid gap-4 lg:grid-cols-3">
      <!-- 待办列表 -->
      <section class="rounded-xl border bg-card lg:col-span-2">
        <div class="flex items-center justify-between gap-3 border-b px-4 py-3">
          <div>
            <h3 class="text-sm font-medium">{t('dashboard.pendingTitle')}</h3>
            <p class="mt-0.5 text-xs text-muted-foreground">{t('dashboard.pendingHint')}</p>
          </div>
          {#if session.can('erp:audit:list')}
            <Button variant="ghost" size="sm" onclick={() => navigate('/erp/audit/in')}>
              {t('dashboard.gotoAudit')}
            </Button>
          {/if}
        </div>
        {#if stats.pending.length === 0}
          <div class="flex flex-col items-center gap-2 px-4 py-12 text-center">
            <ClipboardList class="size-8 text-muted-foreground/60" />
            <p class="text-sm text-muted-foreground">{t('dashboard.pendingEmpty')}</p>
          </div>
        {:else}
          <div class="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('erp.audit.type')}</TableHead>
                  <TableHead>{t('erp.order.no')}</TableHead>
                  <TableHead>{t('erp.audit.partner')}</TableHead>
                  <TableHead class="text-right">{t('erp.order.total')}</TableHead>
                  <TableHead>{t('erp.order.submittedAt')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {#each stats.pending as bill (`${bill.bill_type}-${bill.bill_id}`)}
                  <TableRow>
                    <TableCell>
                      <Badge variant="outline">{t(typeKey(bill.bill_type))}</Badge>
                    </TableCell>
                    <TableCell class="font-mono text-xs">{bill.bill_no}</TableCell>
                    <TableCell>{bill.partner_name || '—'}</TableCell>
                    <TableCell class="text-right tabular-nums">{bill.total_amount.toFixed(2)}</TableCell>
                    <TableCell class="text-xs text-muted-foreground">{shortTime(bill.submitted_at)}</TableCell>
                  </TableRow>
                {/each}
              </TableBody>
            </Table>
          </div>
        {/if}
      </section>

      <!-- 快捷入口：只显示当前角色确实能进的页面 -->
      <aside class="rounded-xl border bg-card">
        <div class="border-b px-4 py-3">
          <h3 class="text-sm font-medium">{t('dashboard.quickLinks')}</h3>
          <p class="mt-0.5 text-xs text-muted-foreground">{t('dashboard.quickHint')}</p>
        </div>
        {#if quickLinks.length === 0}
          <p class="px-4 py-8 text-center text-sm text-muted-foreground">
            {t('dashboard.quickEmpty')}
          </p>
        {:else}
          <ul class="flex flex-col p-2">
            {#each quickLinks as link (link.path)}
              <li>
                <button
                  type="button"
                  class="flex w-full items-center justify-between rounded-md px-3 py-2.5 text-left text-sm transition-colors hover:bg-accent"
                  onclick={() => navigate(link.path)}
                >
                  <span>{t(link.label)}</span>
                  <span class="text-muted-foreground" aria-hidden="true">→</span>
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      </aside>
    </div>
  {/if}
</div>
