<!-- 有问题欢迎联系 hot_kun@hotmail.com -->
<script lang="ts">
  import { onMount } from 'svelte'

  import { api } from '$lib/api/client'
  import type { Goods, Page, StockCheck, Warehouse } from '$lib/api/types'
  import ListHeader from '$lib/components/list-header.svelte'
  import Pagination from '$lib/components/pagination.svelte'
  import TableEmpty from '$lib/components/table-empty.svelte'
  import { Badge } from '$lib/components/ui/badge'
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

  const list = createListState<StockCheck, 'status'>({ endpoint: '/stock-checks', filterKeys: ['status'] })

  let warehouses = $state<Warehouse[]>([])
  let goods = $state<Goods[]>([])
  let formOpen = $state(false)
  let saving = $state(false)
  let busyID = $state<number | null>(null)
  let form = $state({ warehouse_id: 0, goods_id: 0, actual_qty: 0, remark: '' })

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

  function openCreate() {
    form = { warehouse_id: warehouses[0]?.id ?? 0, goods_id: goods[0]?.id ?? 0, actual_qty: 0, remark: '' }
    formOpen = true
  }

  async function save(event: SubmitEvent) {
    event.preventDefault()
    saving = true
    try {
      await api.post('/stock-checks', {
        warehouse_id: Number(form.warehouse_id),
        remark: form.remark,
        items: [{ goods_id: Number(form.goods_id), actual_qty: Number(form.actual_qty) }],
      })
      notifySuccess('notify.saved')
      formOpen = false
      await list.load()
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      saving = false
    }
  }

  async function act(sc: StockCheck) {
    busyID = sc.id
    try {
      await api.post(`/stock-checks/${sc.id}/submit`, {})
      notifySuccess('notify.saved')
      await list.load()
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      busyID = null
    }
  }

  const statusVariant = (s: StockCheck['status']) =>
    s === 'approved' ? 'default' : s === 'rejected' ? 'destructive' : 'secondary'
</script>

<div class="flex flex-col gap-4">
  <ListHeader total={list.total}>
    {#if session.can('erp:stock:edit')}
      <Button size="sm" onclick={openCreate}>{t('common.create')}</Button>
    {/if}
  </ListHeader>

  <div class="rounded-lg border bg-card">
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('erp.order.no')}</TableHead>
          <TableHead>{t('erp.stock.warehouse')}</TableHead>
          <TableHead>{t('common.status')}</TableHead>
          <TableHead>{t('field.remark')}</TableHead>
          <TableHead class="text-right">{t('common.actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {#if list.rows.length === 0}
          <TableEmpty loading={list.loading} colspan={5} />
        {/if}
        {#each list.rows as sc (sc.id)}
          <TableRow>
            <TableCell class="font-mono text-sm">{sc.check_no}</TableCell>
            <TableCell>{sc.warehouse_name}</TableCell>
            <TableCell>
              <Badge variant={statusVariant(sc.status)}>{t(`erp.status.${sc.status}` as const)}</Badge>
            </TableCell>
            <TableCell class="max-w-48 truncate text-muted-foreground">{sc.remark || '—'}</TableCell>
            <TableCell class="text-right whitespace-nowrap">
              {#if session.can('erp:stock:edit') && sc.status === 'draft'}
                <Button variant="ghost" size="sm" disabled={busyID === sc.id} onclick={() => act(sc)}>
                  {t('erp.order.submit')}
                </Button>
              {/if}
            </TableCell>
          </TableRow>
        {/each}
      </TableBody>
    </Table>
  </div>

  <Pagination page={list.page} totalPages={list.totalPages} onGo={(p) => list.goPage(p)} />
</div>

<Dialog bind:open={formOpen}>
  <DialogContent class="sm:max-w-md">
    <DialogHeader>
      <DialogTitle>{t('erp.stockCheck.create')}</DialogTitle>
    </DialogHeader>
    <form class="flex flex-col gap-4" onsubmit={save}>
      <label class="flex flex-col gap-1.5 text-sm">
        <Label>{t('erp.stock.warehouse')}</Label>
        <NativeSelect class="w-full" bind:value={form.warehouse_id}>
          {#each warehouses as w (w.id)}
            <option value={w.id}>{w.name}</option>
          {/each}
        </NativeSelect>
      </label>
      <label class="flex flex-col gap-1.5 text-sm">
        <Label>{t('erp.goods.name')}</Label>
        <NativeSelect class="w-full" bind:value={form.goods_id}>
          {#each goods as g (g.id)}
            <option value={g.id}>{g.name} · {g.unit}</option>
          {/each}
        </NativeSelect>
      </label>
      <label class="flex flex-col gap-1.5 text-sm">
        <Label>{t('erp.stockCheck.actualQty')}</Label>
        <Input type="number" min="0" step="0.001" bind:value={form.actual_qty} />
        <span class="text-xs text-muted-foreground">{t('erp.stockCheck.bookHint')}</span>
      </label>
      <label class="flex flex-col gap-1.5 text-sm">
        <Label>{t('field.remark')}</Label>
        <Input bind:value={form.remark} maxlength={255} />
      </label>
      <DialogFooter>
        <Button type="button" variant="outline" onclick={() => (formOpen = false)}>{t('common.cancel')}</Button>
        <Button type="submit" disabled={saving}>{t('common.save')}</Button>
      </DialogFooter>
    </form>
  </DialogContent>
</Dialog>
