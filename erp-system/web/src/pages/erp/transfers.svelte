<!-- 有问题欢迎联系 hot_kun@hotmail.com -->
<script lang="ts">
  import { onMount } from 'svelte'

  import { api } from '$lib/api/client'
  import type { Goods, Page, StockTransfer, Warehouse } from '$lib/api/types'
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

  const list = createListState<StockTransfer, 'status'>({ endpoint: '/transfers', filterKeys: ['status'] })

  let warehouses = $state<Warehouse[]>([])
  let goods = $state<Goods[]>([])
  let formOpen = $state(false)
  let saving = $state(false)
  let busyID = $state<number | null>(null)
  let form = $state({ from: 0, to: 0, remark: '', goods_id: 0, qty: 0 })

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
    form = { from: warehouses[0]?.id ?? 0, to: warehouses[1]?.id ?? 0, remark: '', goods_id: goods[0]?.id ?? 0, qty: 0 }
    formOpen = true
  }

  async function save(event: SubmitEvent) {
    event.preventDefault()
    saving = true
    try {
      await api.post('/transfers', {
        from_warehouse_id: Number(form.from),
        to_warehouse_id: Number(form.to),
        remark: form.remark,
        items: [{ goods_id: Number(form.goods_id), qty: Number(form.qty) }],
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

  async function act(tr: StockTransfer, action: 'submit' | 'complete') {
    busyID = tr.id
    try {
      await api.post(`/transfers/${tr.id}/${action}`, {})
      notifySuccess('notify.saved')
      await list.load()
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      busyID = null
    }
  }

  const statusVariant = (s: StockTransfer['status']) =>
    s === 'completed' ? 'default' : s === 'rejected' ? 'destructive' : 'secondary'
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
          <TableHead>{t('erp.transfer.from')}</TableHead>
          <TableHead>{t('erp.transfer.to')}</TableHead>
          <TableHead>{t('common.status')}</TableHead>
          <TableHead>{t('field.remark')}</TableHead>
          <TableHead class="text-right">{t('common.actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {#if list.rows.length === 0}
          <TableEmpty loading={list.loading} colspan={6} />
        {/if}
        {#each list.rows as tr (tr.id)}
          <TableRow>
            <TableCell class="font-mono text-sm">{tr.transfer_no}</TableCell>
            <TableCell>{tr.from_warehouse_name}</TableCell>
            <TableCell>{tr.to_warehouse_name}</TableCell>
            <TableCell>
              <Badge variant={statusVariant(tr.status)}>{t(`erp.status.${tr.status}` as const)}</Badge>
            </TableCell>
            <TableCell class="max-w-48 truncate text-muted-foreground">{tr.remark || '—'}</TableCell>
            <TableCell class="text-right whitespace-nowrap">
              {#if session.can('erp:stock:edit') && tr.status === 'draft'}
                <Button variant="ghost" size="sm" disabled={busyID === tr.id} onclick={() => act(tr, 'submit')}>
                  {t('erp.order.submit')}
                </Button>
              {/if}
              {#if session.can('erp:stock:edit') && tr.status === 'approved'}
                <Button variant="ghost" size="sm" disabled={busyID === tr.id} onclick={() => act(tr, 'complete')}>
                  {t('erp.transfer.complete')}
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
      <DialogTitle>{t('erp.transfer.create')}</DialogTitle>
    </DialogHeader>
    <form class="flex flex-col gap-4" onsubmit={save}>
      <label class="flex flex-col gap-1.5 text-sm">
        <Label>{t('erp.transfer.from')}</Label>
        <NativeSelect class="w-full" bind:value={form.from}>
          {#each warehouses as w (w.id)}
            <option value={w.id}>{w.name}</option>
          {/each}
        </NativeSelect>
      </label>
      <label class="flex flex-col gap-1.5 text-sm">
        <Label>{t('erp.transfer.to')}</Label>
        <NativeSelect class="w-full" bind:value={form.to}>
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
        <Label>{t('erp.stock.qty')}</Label>
        <Input type="number" min="0.001" step="0.001" bind:value={form.qty} />
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
