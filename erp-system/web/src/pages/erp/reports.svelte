<!-- 有问题欢迎联系 hot_kun@hotmail.com -->
<script lang="ts">
  import { onMount } from 'svelte'

  import { api } from '$lib/api/client'
  import type { Page, StockFlowRow, StockRow, Warehouse } from '$lib/api/types'
  import ListHeader from '$lib/components/list-header.svelte'
  import TableEmpty from '$lib/components/table-empty.svelte'
  import { Button } from '$lib/components/ui/button'
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
  import { errKey } from '$lib/list-state.svelte'
  import { notifyError } from '$lib/stores/notify.svelte'

  type Mode = 'flow' | 'summary'

  let mode = $state<Mode>('flow')
  let from = $state(today(-7))
  let to = $state(today(1))
  let warehouseID = $state<number | ''>('')
  let warehouses = $state<Warehouse[]>([])
  let flow = $state<StockFlowRow[]>([])
  let summary = $state<StockRow[]>([])
  let loading = $state(false)

  function today(offsetDays: number): string {
    const d = new Date()
    d.setDate(d.getDate() + offsetDays)
    return d.toISOString().slice(0, 10)
  }

  onMount(async () => {
    try {
      const res = await api.get<Page<Warehouse>>('/warehouses?page=1&page_size=200&status=active')
      warehouses = res.list
    } catch (err) {
      notifyError(errKey(err))
    }
    await load()
  })

  async function load() {
    loading = true
    try {
      if (mode === 'flow') {
        const q = new URLSearchParams({ from, to })
        if (warehouseID) q.set('warehouse_id', String(warehouseID))
        const res = await api.get<{ list: StockFlowRow[] }>(`/reports/stock-flow?${q}`)
        flow = res.list
      } else {
        const q = new URLSearchParams({ page: '1', page_size: '200' })
        if (warehouseID) q.set('warehouse_id', String(warehouseID))
        const res = await api.get<Page<StockRow>>(`/reports/stock-summary?${q}`)
        summary = res.list
      }
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      loading = false
    }
  }

  const flowTotal = $derived(
    flow.reduce(
      (acc, r) => ({ inQty: acc.inQty + r.in_qty, outQty: acc.outQty + r.out_qty }),
      { inQty: 0, outQty: 0 },
    ),
  )
</script>

<div class="flex flex-col gap-4">
  <div class="flex flex-wrap items-end gap-3">
    <div class="flex flex-col gap-1 text-xs">
      <span class="text-muted-foreground">{t('report.mode')}</span>
      <div class="flex gap-2">
        <Button size="sm" variant={mode === 'flow' ? 'default' : 'outline'} onclick={() => { mode = 'flow'; void load() }}>
          {t('report.flow')}
        </Button>
        <Button size="sm" variant={mode === 'summary' ? 'default' : 'outline'} onclick={() => { mode = 'summary'; void load() }}>
          {t('report.summary')}
        </Button>
      </div>
    </div>
    {#if mode === 'flow'}
      <label class="flex flex-col gap-1 text-xs">
        <span class="text-muted-foreground">{t('report.from')}</span>
        <Input type="date" bind:value={from} />
      </label>
      <label class="flex flex-col gap-1 text-xs">
        <span class="text-muted-foreground">{t('report.to')}</span>
        <Input type="date" bind:value={to} />
      </label>
    {/if}
    <label class="flex flex-col gap-1 text-xs">
      <span class="text-muted-foreground">{t('erp.stock.warehouse')}</span>
      <NativeSelect class="w-48" bind:value={warehouseID}>
        <option value="">{t('common.all')}</option>
        {#each warehouses as w (w.id)}
          <option value={w.id}>{w.name}</option>
        {/each}
      </NativeSelect>
    </label>
    <Button size="sm" onclick={load} disabled={loading}>{t('common.search')}</Button>
  </div>

  <ListHeader total={mode === 'flow' ? flow.length : summary.length}>
    {#if mode === 'flow'}
      <span class="text-xs text-muted-foreground">
        {t('report.flowTotal', { in: flowTotal.inQty, out: flowTotal.outQty })}
      </span>
    {/if}
  </ListHeader>

  <div class="rounded-lg border bg-card">
    <Table>
      {#if mode === 'flow'}
        <TableHeader>
          <TableRow>
            <TableHead>{t('erp.goods.name')}</TableHead>
            <TableHead>{t('field.unit')}</TableHead>
            <TableHead class="text-right">{t('report.inQty')}</TableHead>
            <TableHead class="text-right">{t('report.outQty')}</TableHead>
            <TableHead class="text-right">{t('report.netQty')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {#if flow.length === 0}
            <TableEmpty loading={loading} colspan={5} />
          {/if}
          {#each flow as r (r.goods_id)}
            <TableRow>
              <TableCell>
                <span class="font-mono text-xs text-muted-foreground">{r.goods_code}</span>
                <span class="ml-2 font-medium">{r.goods_name}</span>
              </TableCell>
              <TableCell>{r.unit}</TableCell>
              <TableCell class="text-right tabular-nums">{r.in_qty}</TableCell>
              <TableCell class="text-right tabular-nums">{r.out_qty}</TableCell>
              <TableCell class="text-right tabular-nums">{r.net_qty}</TableCell>
            </TableRow>
          {/each}
        </TableBody>
      {:else}
        <TableHeader>
          <TableRow>
            <TableHead>{t('erp.stock.warehouse')}</TableHead>
            <TableHead>{t('erp.goods.name')}</TableHead>
            <TableHead>{t('field.unit')}</TableHead>
            <TableHead class="text-right">{t('erp.stock.qty')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {#if summary.length === 0}
            <TableEmpty loading={loading} colspan={4} />
          {/if}
          {#each summary as s (`${s.warehouse_id}-${s.goods_id}`)}
            <TableRow>
              <TableCell>{s.warehouse_name}</TableCell>
              <TableCell>
                <span class="font-mono text-xs text-muted-foreground">{s.goods_code}</span>
                <span class="ml-2 font-medium">{s.goods_name}</span>
              </TableCell>
              <TableCell>{s.unit}</TableCell>
              <TableCell class="text-right tabular-nums">{s.qty}</TableCell>
            </TableRow>
          {/each}
        </TableBody>
      {/if}
    </Table>
  </div>
</div>
