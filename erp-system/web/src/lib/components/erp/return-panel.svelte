<!-- 有问题欢迎联系 hot_kun@hotmail.com -->
<script lang="ts">
  import { onMount } from 'svelte'

  import { ApiError, api } from '$lib/api/client'
  import type {
    BillOperation,
    OrderItem,
    OrderStatus,
    Page,
    PurchaseOrder,
    ReturnOrder,
    SalesOrder,
  } from '$lib/api/types'
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
  import { t, type MessageKey } from '$lib/i18n/index.svelte'
  import { createListState, errKey } from '$lib/list-state.svelte'
  import { confirm } from '$lib/stores/confirm.svelte'
  import { notifyError, notifySuccess } from '$lib/stores/notify.svelte'
  import { session } from '$lib/stores/session.svelte'

  type SourceOrder = PurchaseOrder | SalesOrder

  /** 建单/改单时的一行：来源数量、其它未驳回退货已占数量、本次可退数量。 */
  type ReturnLine = {
    goods_id: number
    goods_name: string
    price: number
    source_qty: number
    claimed: number
    qty: number
  }

  let { kind }: { kind: 'purchase' | 'sales' } = $props()

  // svelte-ignore state_referenced_locally
  const isPurchase = kind === 'purchase'
  const endpoint = isPurchase ? '/purchase-returns' : '/sales-returns'
  const sourceEndpoint = isPurchase ? '/purchases' : '/sales'
  const billType = isPurchase ? 'purchase-return' : 'sale-return'
  const partyLabel: MessageKey = isPurchase ? 'erp.order.supplier' : 'erp.order.customer'
  const titleKey: MessageKey = isPurchase ? 'erp.return.purchase.title' : 'erp.return.sales.title'
  const completeConfirm: MessageKey = isPurchase
    ? 'erp.return.purchaseComplete'
    : 'erp.return.salesComplete'
  const editPerm = 'erp:return:edit'

  const statuses: OrderStatus[] = ['draft', 'submitted', 'approved', 'completed', 'rejected']

  const list = createListState<ReturnOrder, 'status' | 'keyword'>({
    endpoint,
    filterKeys: ['status', 'keyword'],
  })

  let statusFilter = $state('')
  let keyword = $state('')

  const statusKey = (s: OrderStatus): MessageKey => `erp.status.${s}`
  const actionKey = (action: string): MessageKey => `erp.action.${action}` as MessageKey
  const partyOf = (r: ReturnOrder) => (isPurchase ? r.supplier_name : r.customer_name) ?? '—'
  const sourceParty = (s: SourceOrder) =>
    (isPurchase ? (s as PurchaseOrder).supplier_name : (s as SalesOrder).customer_name) ?? '—'

  onMount(() => {
    statusFilter = list.filters.status
    keyword = list.filters.keyword
    void list.load()
  })

  function search() {
    list.filters.status = statusFilter
    list.filters.keyword = keyword.trim()
    list.search()
  }

  function resetFilters() {
    statusFilter = ''
    keyword = ''
    list.reset()
  }

  // ---- 建单 / 改单 ----

  let formOpen = $state(false)
  let editingID = $state<number | null>(null)
  let saving = $state(false)
  let formError = $state('')
  let sources = $state<SourceOrder[]>([])
  let sourceLoading = $state(false)
  let selectedSourceID = $state(0)
  let lines = $state<ReturnLine[]>([])
  let remark = $state('')

  const draftTotal = $derived(
    lines.reduce((sum, line) => sum + Number(line.qty || 0) * Number(line.price), 0),
  )

  async function loadSources() {
    sourceLoading = true
    try {
      const res = await api.get<Page<SourceOrder>>(
        `${sourceEndpoint}?page=1&page_size=200&status=completed`,
      )
      sources = res.list
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      sourceLoading = false
    }
  }

  /** 载入来源单明细，并汇总「其它未驳回退货」已占数量。 */
  async function selectSource(id: number, preset?: OrderItem[]) {
    selectedSourceID = id
    formError = ''
    if (!id) {
      lines = []
      return
    }
    try {
      const [detail, returns] = await Promise.all([
        api.get<SourceOrder>(`${sourceEndpoint}/${id}`),
        api.get<Page<ReturnOrder>>(`${endpoint}?page=1&page_size=200&source_order_id=${id}`),
      ])
      // 列表行不带明细：逐单取详情汇总已退量。一个来源单的退货单通常只有几张。
      const active = returns.list.filter((r) => r.status !== 'draft' && r.status !== 'rejected')
      const details = await Promise.all(
        active.map((r) => api.get<ReturnOrder>(`${endpoint}/${r.id}`)),
      )
      const claimed = new Map<number, number>()
      for (const ret of details) {
        for (const it of ret.items ?? []) {
          claimed.set(it.goods_id, (claimed.get(it.goods_id) ?? 0) + it.qty)
        }
      }

      const presetQty = new Map<number, number>()
      for (const it of preset ?? []) presetQty.set(it.goods_id, it.qty)

      lines = detail.items.map((it) => {
        const used = claimed.get(it.goods_id) ?? 0
        return {
          goods_id: it.goods_id,
          goods_name: it.goods_name ?? '',
          price: it.price,
          source_qty: it.qty,
          claimed: used,
          qty: presetQty.get(it.goods_id) ?? 0,
        }
      })
    } catch (err) {
      notifyError(errKey(err))
    }
  }

  async function openCreate() {
    editingID = null
    remark = ''
    formError = ''
    formOpen = true
    if (sources.length === 0) await loadSources()
    await selectSource(sources[0]?.id ?? 0)
  }

  async function openEdit(ret: ReturnOrder) {
    editingID = ret.id
    remark = ret.remark
    formError = ''
    formOpen = true
    if (sources.length === 0) await loadSources()
    // 来源单在编辑时不可改；用它带出现有明细与已退量。
    const detail = await api.get<ReturnOrder>(`${endpoint}/${ret.id}`)
    selectedSourceID = detail.source_order_id
    await selectSource(detail.source_order_id, detail.items ?? [])
  }

  async function saveReturn(event: SubmitEvent) {
    event.preventDefault()
    formError = ''
    if (!selectedSourceID) {
      formError = t('erp.return.sourceEmpty')
      return
    }
    const payloadItems = lines
      .filter((line) => Number(line.qty) > 0)
      .map((line) => ({ goods_id: line.goods_id, qty: Number(line.qty) }))
    if (payloadItems.length === 0) {
      formError = t('erp.return.atLeastOne')
      return
    }
    for (const line of lines) {
      const remaining = Math.max(0, line.source_qty - line.claimed)
      if (Number(line.qty) > remaining + 1e-9) {
        formError = `${line.goods_name}: ${t('erp.return.remainingQty')} ${remaining}`
        return
      }
    }

    saving = true
    try {
      if (editingID === null) {
        await api.post(endpoint, {
          source_order_id: selectedSourceID,
          items: payloadItems,
          remark: remark.trim(),
        })
      } else {
        await api.put(`${endpoint}/${editingID}`, { items: payloadItems, remark: remark.trim() })
      }
      notifySuccess('notify.saved')
      formOpen = false
      await list.load()
    } catch (err) {
      notifyError(err instanceof ApiError ? err.key : 'error.internal')
    } finally {
      saving = false
    }
  }

  // ---- 状态操作 ----

  let busyID = $state<number | null>(null)

  async function act(ret: ReturnOrder, action: string, confirmTitle?: MessageKey) {
    if (confirmTitle && !(await confirm({ titleKey: confirmTitle }))) return
    busyID = ret.id
    try {
      await api.post(`${endpoint}/${ret.id}/${action}`, {})
      notifySuccess('notify.saved')
      await list.load()
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      busyID = null
    }
  }

  async function removeDraft(ret: ReturnOrder) {
    if (!(await confirm({ titleKey: 'erp.order.deleteConfirm', danger: true }))) return
    busyID = ret.id
    try {
      await api.del(`${endpoint}/${ret.id}`)
      notifySuccess('notify.deleted')
      await list.refreshAfterDelete()
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      busyID = null
    }
  }

  // ---- 详情 ----

  let detailOpen = $state(false)
  let detail = $state<ReturnOrder | null>(null)
  let operations = $state<BillOperation[]>([])
  let detailLoading = $state(false)

  async function openDetail(ret: ReturnOrder) {
    detailOpen = true
    detailLoading = true
    detail = null
    operations = []
    try {
      const [full, ops] = await Promise.all([
        api.get<ReturnOrder>(`${endpoint}/${ret.id}`),
        api.get<{ list: BillOperation[] }>(`/bills/${billType}/${ret.id}/operations`),
      ])
      detail = full
      operations = ops.list
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      detailLoading = false
    }
  }
</script>

<div class="flex flex-col gap-4">
  <ListHeader total={list.total}>
    {#if session.can(editPerm)}
      <Button size="sm" onclick={openCreate}>{t('erp.return.create')}</Button>
    {/if}
  </ListHeader>

  <form
    class="flex flex-wrap items-end gap-2"
    onsubmit={(e) => {
      e.preventDefault()
      search()
    }}
  >
    <label class="flex flex-col gap-1 text-xs">
      <span class="text-muted-foreground">{t('common.status')}</span>
      <NativeSelect class="w-36" bind:value={statusFilter}>
        <option value="">{t('erp.order.allStatus')}</option>
        {#each statuses as s (s)}
          <option value={s}>{t(statusKey(s))}</option>
        {/each}
      </NativeSelect>
    </label>
    <label class="flex flex-col gap-1 text-xs">
      <span class="text-muted-foreground">{t('erp.order.keyword')}</span>
      <Input class="w-52" bind:value={keyword} placeholder={t('erp.order.source')} />
    </label>
    <Button type="submit" variant="outline" size="sm">{t('common.search')}</Button>
    <Button type="button" variant="ghost" size="sm" onclick={resetFilters}>{t('common.reset')}</Button>
  </form>

  <div class="overflow-x-auto rounded-lg border bg-card">
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('erp.order.no')}</TableHead>
          <TableHead>{t('erp.return.source')}</TableHead>
          <TableHead>{t(partyLabel)}</TableHead>
          <TableHead>{t('erp.order.warehouse')}</TableHead>
          <TableHead>{t('common.status')}</TableHead>
          <TableHead class="text-right">{t('erp.order.total')}</TableHead>
          <TableHead class="text-right">{t('common.actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {#if list.rows.length === 0}
          <TableEmpty loading={list.loading} colspan={7} />
        {/if}
        {#each list.rows as ret (ret.id)}
          <TableRow>
            <TableCell class="font-mono text-sm">{ret.return_no}</TableCell>
            <TableCell class="font-mono text-xs text-muted-foreground">{ret.source_order_no}</TableCell>
            <TableCell>{partyOf(ret)}</TableCell>
            <TableCell>{ret.warehouse_name}</TableCell>
            <TableCell>
              <Badge
                variant={ret.status === 'rejected'
                  ? 'destructive'
                  : ret.status === 'completed'
                    ? 'default'
                    : 'secondary'}
              >
                {t(statusKey(ret.status))}
              </Badge>
            </TableCell>
            <TableCell class="text-right tabular-nums">{ret.total_amount.toFixed(2)}</TableCell>
            <TableCell class="text-right whitespace-nowrap">
              <Button variant="ghost" size="sm" onclick={() => openDetail(ret)}>{t('erp.order.detail')}</Button>
              {#if session.can(editPerm) && ret.status === 'draft'}
                <Button variant="ghost" size="sm" onclick={() => openEdit(ret)}>{t('common.edit')}</Button>
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={busyID === ret.id}
                  onclick={() => act(ret, 'submit', 'erp.return.submitConfirm')}
                >
                  {t('erp.order.submit')}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  class="text-destructive"
                  disabled={busyID === ret.id}
                  onclick={() => removeDraft(ret)}
                >
                  {t('erp.order.deleteDraft')}
                </Button>
              {/if}
              {#if session.can(editPerm) && ret.status === 'submitted'}
                <Button variant="ghost" size="sm" disabled={busyID === ret.id} onclick={() => act(ret, 'withdraw')}>
                  {t('erp.order.withdraw')}
                </Button>
              {/if}
              {#if session.can(editPerm) && ret.status === 'approved'}
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={busyID === ret.id}
                  onclick={() => act(ret, 'complete', completeConfirm)}
                >
                  {t(completeConfirm)}
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
  <DialogContent class="sm:max-w-3xl">
    <DialogHeader>
      <DialogTitle>{editingID === null ? t('erp.return.create') : t('erp.return.edit')}</DialogTitle>
    </DialogHeader>
    <form class="flex max-h-[72vh] flex-col gap-4 overflow-y-auto pr-1" onsubmit={saveReturn}>
      <label class="flex flex-col gap-1.5 text-sm">
        <Label for="return-source">{t('erp.return.source')}</Label>
        <NativeSelect
          id="return-source"
          class="w-full"
          value={String(selectedSourceID)}
          disabled={editingID !== null}
          onchange={(e) => selectSource(Number(e.currentTarget.value))}
          required
        >
          {#if sources.length === 0}
            <option value="0">{t('erp.return.sourceEmpty')}</option>
          {/if}
          {#each sources as source (source.id)}
            <option value={String(source.id)}>{source.order_no} · {sourceParty(source)}</option>
          {/each}
        </NativeSelect>
        <span class="text-xs text-muted-foreground">{t('erp.return.sourceHint')}</span>
      </label>

      {#if lines.length > 0}
        <div class="flex flex-col gap-2">
          <div class="text-sm font-medium">{t('erp.order.items')}</div>
          <span class="text-xs text-muted-foreground">{t('erp.return.itemsHint')}</span>
          <div class="overflow-x-auto rounded border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('erp.goods.name')}</TableHead>
                  <TableHead class="text-right">{t('erp.return.sourceQty')}</TableHead>
                  <TableHead class="text-right">{t('erp.return.returnedQty')}</TableHead>
                  <TableHead class="text-right">{t('erp.return.remainingQty')}</TableHead>
                  <TableHead class="text-right">{t('field.price')}</TableHead>
                  <TableHead class="w-28 text-right">{t('erp.return.qty')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {#each lines as line (line.goods_id)}
                  {@const remaining = Math.max(0, line.source_qty - line.claimed)}
                  <TableRow>
                    <TableCell>{line.goods_name}</TableCell>
                    <TableCell class="text-right tabular-nums">{line.source_qty}</TableCell>
                    <TableCell class="text-right tabular-nums">{line.claimed}</TableCell>
                    <TableCell class="text-right tabular-nums">{remaining}</TableCell>
                    <TableCell class="text-right tabular-nums">{line.price.toFixed(2)}</TableCell>
                    <TableCell class="text-right">
                      <Input
                        type="number"
                        step="0.001"
                        min={0}
                        max={remaining}
                        class="h-8 text-right"
                        bind:value={line.qty}
                      />
                    </TableCell>
                  </TableRow>
                {/each}
              </TableBody>
            </Table>
          </div>
          <div class="text-right text-sm">
            {t('erp.order.total')}:
            <span class="font-semibold tabular-nums">{draftTotal.toFixed(2)}</span>
          </div>
        </div>
      {:else if !sourceLoading}
        <p class="text-sm text-muted-foreground">{t('erp.return.sourceEmpty')}</p>
      {/if}

      <label class="flex flex-col gap-1.5 text-sm">
        <Label for="return-remark">{t('erp.order.remark')}</Label>
        <Input id="return-remark" bind:value={remark} maxlength={255} placeholder={t('erp.order.remarkHint')} />
      </label>

      {#if formError}
        <p class="text-sm text-destructive">{formError}</p>
      {/if}

      <DialogFooter>
        <Button type="button" variant="outline" onclick={() => (formOpen = false)}>{t('common.cancel')}</Button>
        <Button type="submit" disabled={saving || lines.length === 0}>{t('common.save')}</Button>
      </DialogFooter>
    </form>
  </DialogContent>
</Dialog>


<Dialog bind:open={detailOpen}>
  <DialogContent class="sm:max-w-2xl">
    <DialogHeader>
      <DialogTitle>{detail?.return_no ?? t('erp.order.detail')}</DialogTitle>
    </DialogHeader>
    <div class="flex max-h-[70vh] flex-col gap-4 overflow-y-auto pr-1 text-sm">
      {#if detailLoading}
        <p class="text-sm text-muted-foreground">{t('common.loading')}</p>
      {:else if detail}
        <div class="grid grid-cols-2 gap-2 text-xs text-muted-foreground">
          <div>{t('erp.return.source')}: {detail.source_order_no}</div>
          <div>{t(partyLabel)}: {partyOf(detail)}</div>
          <div>{t('erp.order.warehouse')}: {detail.warehouse_name}</div>
          <div>{t('common.status')}: {t(statusKey(detail.status))}</div>
          <div>{t('erp.order.total')}: {detail.total_amount.toFixed(2)}</div>
          <div>{t('erp.order.remark')}: {detail.remark || '—'}</div>
        </div>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('erp.goods.name')}</TableHead>
              <TableHead class="text-right">{t('field.qty')}</TableHead>
              <TableHead class="text-right">{t('field.price')}</TableHead>
              <TableHead class="text-right">{t('erp.order.amount')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {#each detail.items ?? [] as it (it.id ?? `${it.goods_id}-${it.qty}`)}
              <TableRow>
                <TableCell>{it.goods_name}</TableCell>
                <TableCell class="text-right tabular-nums">{it.qty}</TableCell>
                <TableCell class="text-right tabular-nums">{it.price.toFixed(2)}</TableCell>
                <TableCell class="text-right tabular-nums">{it.amount?.toFixed(2)}</TableCell>
              </TableRow>
            {/each}
          </TableBody>
        </Table>

        <div>
          <div class="mb-2 text-sm font-medium">{t('erp.order.operations')}</div>
          {#if operations.length === 0}
            <p class="text-sm text-muted-foreground">{t('common.empty')}</p>
          {:else}
            <ul class="flex flex-col gap-1 text-xs">
              {#each operations as op (op.id)}
                <li class="flex justify-between gap-2 rounded border px-2 py-1">
                  <span>
                    {t(actionKey(op.action))} · {op.operator_name}
                    {#if op.comment}
                      <span class="text-muted-foreground">— {op.comment}</span>
                    {/if}
                  </span>
                  <span class="shrink-0 text-muted-foreground">{op.created_at}</span>
                </li>
              {/each}
            </ul>
          {/if}
        </div>
      {/if}
      <DialogFooter>
        <Button type="button" variant="outline" onclick={() => (detailOpen = false)}>{t('common.close')}</Button>
      </DialogFooter>
    </div>
  </DialogContent>
</Dialog>
