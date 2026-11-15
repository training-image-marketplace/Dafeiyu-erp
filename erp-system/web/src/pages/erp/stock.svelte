<!-- 有问题欢迎联系 hot_kun@hotmail.com -->
<script lang="ts">
  import { onMount } from 'svelte'

  import { ApiError, api } from '$lib/api/client'
  import type { Goods, Page, StockAdjustment, StockLog, StockRow, Warehouse } from '$lib/api/types'
  import FilterBar from '$lib/components/filter-bar.svelte'
  import ListHeader from '$lib/components/list-header.svelte'
  import Pagination from '$lib/components/pagination.svelte'
  import TableEmpty from '$lib/components/table-empty.svelte'
  import { Button } from '$lib/components/ui/button'
  import {
    Dialog,
    DialogContent,
    DialogFooter,
    DialogHeader,
    DialogTitle,
  } from '$lib/components/ui/dialog'
  import { Input } from '$lib/components/ui/input'
  import { Label } from '$lib/components/ui/label'
  import { NativeSelect } from '$lib/components/ui/native-select'
  import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
  } from '$lib/components/ui/table'
  import { t } from '$lib/i18n/index.svelte'
  import { createListState, errKey } from '$lib/list-state.svelte'
  import { notifyError, notifySuccess } from '$lib/stores/notify.svelte'
  import { session } from '$lib/stores/session.svelte'

  const list = createListState<StockRow, 'warehouse_id' | 'keyword'>({
    endpoint: '/stocks',
    filterKeys: ['warehouse_id', 'keyword'],
  })

  let warehouses = $state<Warehouse[]>([])
  let goods = $state<Goods[]>([])

  onMount(async () => {
    await Promise.all([list.load(), loadOptions()])
  })

  async function loadOptions() {
    try {
      const [w, g] = await Promise.all([
        api.get<Page<Warehouse>>('/warehouses?page=1&page_size=200&status=active'),
        api.get<Page<Goods>>('/goods?page=1&page_size=200&status=active'),
      ])
      warehouses = w.list
      goods = g.list
    } catch (err) {
      notifyError(errKey(err))
    }
  }

  // ---- 调整 ----

  let adjustOpen = $state(false)
  let saving = $state(false)
  let adjustment = $state<StockAdjustment | null>(null)
  let form = $state({ warehouse_id: 0, goods_id: 0, direction: 'in' as 'in' | 'out', qty: 0, reason: '' })

  function openAdjust() {
    adjustment = null
    form = {
      warehouse_id: warehouses[0]?.id ?? 0,
      goods_id: goods[0]?.id ?? 0,
      direction: 'in',
      qty: 0,
      reason: '',
    }
    adjustOpen = true
  }

  async function adjust(event: SubmitEvent) {
    event.preventDefault()
    saving = true
    try {
      adjustment = await api.post<StockAdjustment>('/stock-adjustments', {
        warehouse_id: Number(form.warehouse_id),
        goods_id: Number(form.goods_id),
        direction: form.direction,
        qty: Number(form.qty),
        reason: form.reason,
      })
      notifySuccess('notify.saved')
      await list.load()
    } catch (err) {
      notifyError(err instanceof ApiError ? err.key : 'error.internal')
    } finally {
      saving = false
    }
  }

  async function submitAdjustment() {
    if (!adjustment) return
    saving = true
    try {
      await api.post(`/stock-adjustments/${adjustment.id}/submit`, {})
      adjustment = await api.get<StockAdjustment>(`/stock-adjustments/${adjustment.id}`)
      notifySuccess('notify.saved')
      await list.load()
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      saving = false
    }
  }

  async function deleteAdjustment() {
    if (!adjustment) return
    saving = true
    try {
      await api.del(`/stock-adjustments/${adjustment.id}`)
      adjustment = null
      adjustOpen = false
      await list.load()
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      saving = false
    }
  }

  // ---- 流水 ----

  let showLogs = $state(false)
  let logs = $state<StockLog[]>([])
  let logPage = $state(1)
  let logTotal = $state(0)
  let logLoading = $state(false)

  async function loadLogs() {
    logLoading = true
    try {
      const q = new URLSearchParams({ logs: '1', page: String(logPage), page_size: '20' })
      if (list.filters.warehouse_id) q.set('warehouse_id', list.filters.warehouse_id)
      if (list.filters.keyword) q.set('keyword', list.filters.keyword)
      const res = await api.get<Page<StockLog>>(`/stocks?${q}`)
      logs = res.list
      logTotal = res.total
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      logLoading = false
    }
  }

  function toggleLogs() {
    showLogs = !showLogs
    if (showLogs) {
      logPage = 1
      void loadLogs()
    }
  }

  const logPages = $derived(Math.max(1, Math.ceil(logTotal / 20)))
</script>

<div class="flex flex-col gap-4">
  <FilterBar
    onSearch={() => {
      list.search()
      if (showLogs) void loadLogs()
    }}
    onReset={() => {
      list.reset()
      if (showLogs) void loadLogs()
    }}
  >
    <label class="flex w-52 flex-col gap-1.5 text-sm">
      <span class="text-muted-foreground">{t('erp.stock.warehouse')}</span>
      <NativeSelect class="w-full" bind:value={list.filters.warehouse_id}>
        <option value="">{t('common.all')}</option>
        {#each warehouses as w (w.id)}
          <option value={String(w.id)}>{w.name}</option>
        {/each}
      </NativeSelect>
    </label>
    <label class="flex w-56 flex-col gap-1.5 text-sm">
      <span class="text-muted-foreground">{t('erp.keyword')}</span>
      <Input placeholder={t('erp.keywordHint')} bind:value={list.filters.keyword} />
    </label>
  </FilterBar>

  <ListHeader total={showLogs ? logTotal : list.total}>
    <Button variant="outline" size="sm" onclick={toggleLogs}>
      {showLogs ? t('erp.stock.balance') : t('erp.stock.logs')}
    </Button>
    {#if session.can('erp:stock:edit')}
      <Button size="sm" onclick={openAdjust}>{t('erp.stock.adjust')}</Button>
    {/if}
  </ListHeader>

  <div class="rounded-lg border bg-card">
    {#if showLogs}
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('common.time')}</TableHead>
            <TableHead>{t('erp.goods.name')}</TableHead>
            <TableHead class="text-right">{t('erp.stock.change')}</TableHead>
            <TableHead>{t('erp.stock.bizType')}</TableHead>
            <TableHead>{t('erp.stock.refNo')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {#if logs.length === 0}
            <TableEmpty loading={logLoading} colspan={5} />
          {/if}
          {#each logs as l (l.id)}
            <TableRow>
              <TableCell class="text-xs text-muted-foreground">{l.created_at}</TableCell>
              <TableCell>{l.goods_name}</TableCell>
              <TableCell class="text-right tabular-nums">
                {l.change_qty > 0 ? `+${l.change_qty}` : l.change_qty}
              </TableCell>
              <TableCell>{l.biz_type}</TableCell>
              <TableCell class="font-mono text-xs">{l.ref_no || '—'}</TableCell>
            </TableRow>
          {/each}
        </TableBody>
      </Table>
    {:else}
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('erp.stock.warehouse')}</TableHead>
            <TableHead>{t('erp.goods.name')}</TableHead>
            <TableHead>{t('field.unit')}</TableHead>
            <TableHead class="text-right">{t('erp.stock.qty')}</TableHead>
            <TableHead>{t('common.time')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {#if list.rows.length === 0}
            <TableEmpty loading={list.loading} colspan={5} />
          {/if}
          {#each list.rows as s (`${s.warehouse_id}-${s.goods_id}`)}
            <TableRow>
              <TableCell>{s.warehouse_name}</TableCell>
              <TableCell>
                <span class="font-mono text-xs text-muted-foreground">{s.goods_code}</span>
                <span class="ml-2 font-medium">{s.goods_name}</span>
              </TableCell>
              <TableCell>{s.unit}</TableCell>
              <TableCell class="text-right tabular-nums">{s.qty}</TableCell>
              <TableCell class="text-xs text-muted-foreground">{s.updated_at}</TableCell>
            </TableRow>
          {/each}
        </TableBody>
      </Table>
    {/if}
  </div>

  {#if showLogs}
    <Pagination
      page={logPage}
      totalPages={logPages}
      onGo={(p) => {
        logPage = p
        void loadLogs()
      }}
    />
  {:else}
    <Pagination page={list.page} totalPages={list.totalPages} onGo={(p) => list.goPage(p)} />
  {/if}
</div>

<Dialog bind:open={adjustOpen}>
  <DialogContent class="sm:max-w-md">
    <DialogHeader>
      <DialogTitle>{t('erp.stock.adjust')}</DialogTitle>
    </DialogHeader>
    {#if adjustment}
      <div class="flex flex-col gap-4 text-sm">
        <dl class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-2 text-xs">
          <dt class="text-muted-foreground">{t('common.status')}</dt>
          <dd>{t(`erp.status.${adjustment.status}` as const)}</dd>
          <dt class="text-muted-foreground">{t('erp.order.no')}</dt>
          <dd class="font-mono">{adjustment.adjustment_no}</dd>
          <dt class="text-muted-foreground">{t('erp.stock.warehouse')}</dt>
          <dd>{adjustment.warehouse_name}</dd>
          <dt class="text-muted-foreground">{t('erp.goods.name')}</dt>
          <dd>{adjustment.goods_name}</dd>
          <dt class="text-muted-foreground">{t('erp.stock.direction')}</dt>
          <dd>{t(`erp.stock.direction.${adjustment.direction}` as const)}</dd>
          <dt class="text-muted-foreground">{t('erp.stock.qty')}</dt>
          <dd class="font-mono">{adjustment.direction === 'in' ? '+' : '−'}{adjustment.qty} {adjustment.unit}</dd>
          <dt class="text-muted-foreground">{t('erp.stock.reason')}</dt>
          <dd>{adjustment.reason || '—'}</dd>
        </dl>
        <p class="text-xs text-muted-foreground">{t('erp.stock.pendingApprovalHint')}</p>
        <DialogFooter>
          {#if adjustment.status === 'draft'}
            <Button type="button" variant="outline" disabled={saving} onclick={deleteAdjustment}>{t('common.delete')}</Button>
            <Button type="button" disabled={saving} onclick={submitAdjustment}>{t('erp.order.submit')}</Button>
          {:else}
            <Button type="button" variant="outline" onclick={() => (adjustOpen = false)}>{t('common.close')}</Button>
          {/if}
        </DialogFooter>
      </div>
    {:else}
      <form class="flex flex-col gap-4" onsubmit={adjust}>
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t('erp.stock.warehouse')}</Label>
          <NativeSelect
            class="w-full"
            value={String(form.warehouse_id)}
            onchange={(e) => (form.warehouse_id = Number(e.currentTarget.value))}
            required
          >
            {#each warehouses as w (w.id)}
              <option value={String(w.id)}>{w.name}</option>
            {/each}
          </NativeSelect>
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t('erp.goods.name')}</Label>
          <NativeSelect
            class="w-full"
            value={String(form.goods_id)}
            onchange={(e) => (form.goods_id = Number(e.currentTarget.value))}
            required
          >
            {#each goods as g (g.id)}
              <option value={String(g.id)}>{g.name} · {g.unit}</option>
            {/each}
          </NativeSelect>
        </label>
        <fieldset class="flex flex-col gap-1.5 text-sm">
          <legend class="text-sm font-medium">{t('erp.stock.direction')}</legend>
          <div class="grid grid-cols-2 gap-2">
            <Button type="button" variant={form.direction === 'in' ? 'default' : 'outline'} onclick={() => (form.direction = 'in')}>
              {t('erp.stock.direction.in')}
            </Button>
            <Button type="button" variant={form.direction === 'out' ? 'default' : 'outline'} onclick={() => (form.direction = 'out')}>
              {t('erp.stock.direction.out')}
            </Button>
          </div>
        </fieldset>
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t('erp.stock.qty')}</Label>
          <Input type="number" min="0.001" step="0.001" bind:value={form.qty} oninput={(e) => (form.qty = Number(e.currentTarget.value))} required />
          <span class="text-xs text-muted-foreground">{t('erp.stock.effectPreview')}: {form.direction === 'in' ? '+' : '−'}{form.qty || 0}</span>
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t('erp.stock.reason')}</Label>
          <Input bind:value={form.reason} maxlength={255} />
        </label>
        <DialogFooter>
          <Button type="button" variant="outline" onclick={() => (adjustOpen = false)}>{t('common.cancel')}</Button>
          <Button type="submit" disabled={saving || Number(form.qty) <= 0}>{t('common.save')}</Button>
        </DialogFooter>
      </form>
    {/if}
  </DialogContent>
</Dialog>
