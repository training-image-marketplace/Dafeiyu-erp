<!-- 有问题欢迎联系 hot_kun@hotmail.com -->
<script lang="ts">
  import { onMount } from 'svelte'

  import { ApiError, api } from '$lib/api/client'
  import type {
    BillOperation,
    Customer,
    Goods,
    OrderItem,
    OrderStatus,
    Page,
    Supplier,
    Warehouse,
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

  type OrderRow = {
    id: number
    order_no: string
    status: OrderStatus
    total_amount: number
    warehouse_id: number
    warehouse_name: string
    items: OrderItem[]
    supplier_id?: number
    supplier_name?: string
    customer_id?: number
    customer_name?: string
  }

  let { kind }: { kind: 'purchase' | 'sales' } = $props()

  // svelte-ignore state_referenced_locally
  // kind 在一次挂载内不会变：采购/销售各自是一个页面实例。
  const isPurchase = kind === 'purchase'
  const endpoint = isPurchase ? '/purchases' : '/sales'
  const partyEndpoint = isPurchase ? '/suppliers' : '/customers'
  // svelte-ignore state_referenced_locally
  const billType = kind
  const partyLabel: MessageKey = isPurchase ? 'erp.order.supplier' : 'erp.order.customer'
  const completeLabel: MessageKey = isPurchase ? 'erp.order.receive' : 'erp.order.ship'
  const listPerm = isPurchase ? 'erp:purchase:list' : 'erp:sales:list'
  const editPerm = isPurchase ? 'erp:purchase:edit' : 'erp:sales:edit'

  const list = createListState<OrderRow, 'status'>({
    endpoint,
    filterKeys: ['status'],
  })

  let parties = $state<Array<Supplier | Customer>>([])
  let warehouses = $state<Warehouse[]>([])
  let goods = $state<Goods[]>([])

  onMount(async () => {
    await Promise.all([list.load(), loadOptions()])
  })

  async function loadOptions() {
    try {
      const [p, w, g] = await Promise.all([
        api.get<Page<Supplier | Customer>>(`${partyEndpoint}?page=1&page_size=200&status=active`),
        api.get<Page<Warehouse>>('/warehouses?page=1&page_size=200&status=active'),
        api.get<Page<Goods>>('/goods?page=1&page_size=200&status=active'),
      ])
      parties = p.list
      warehouses = w.list
      goods = g.list
    } catch (err) {
      notifyError(errKey(err))
    }
  }

  const partyOf = (o: OrderRow) => (isPurchase ? o.supplier_name : o.customer_name) ?? '—'
  const partyIDOf = (o: OrderRow) => (isPurchase ? o.supplier_id : o.customer_id) ?? 0

  const statusKey = (s: OrderStatus): MessageKey => `erp.status.${s}`
  const actionKey = (action: string): MessageKey => `erp.action.${action}` as MessageKey

  // ---- 建单 ----

  let createOpen = $state(false)
  let creating = $state(false)
  let editingID = $state<number | null>(null)
  let form = $state({ party_id: 0, warehouse_id: 0 })
  let items = $state<Array<{ goods_id: number; qty: number; price: number }>>([])

  // 采购/销售分别用建议采购价/建议销售价，缺失时回退到售价。
  function suggestedPrice(g: Goods | undefined): number {
    if (!g) return 0
    return isPurchase ? g.purchase_price || g.price : g.sale_price || g.price
  }

  function openCreate() {
    editingID = null
    form = { party_id: parties[0]?.id ?? 0, warehouse_id: warehouses[0]?.id ?? 0 }
    items = [{ goods_id: goods[0]?.id ?? 0, qty: 1, price: suggestedPrice(goods[0]) }]
    createOpen = true
  }

  // 列表接口只返回单头，明细要回查详情；否则 items 为 null，编辑弹窗会崩。
  async function openEdit(order: OrderRow) {
    editingID = order.id
    form = { party_id: partyIDOf(order), warehouse_id: order.warehouse_id }
    items = []
    createOpen = true
    try {
      const full = await api.get<OrderRow>(`${endpoint}/${order.id}`)
      items = (full.items ?? []).map((it) => ({ goods_id: it.goods_id, qty: it.qty, price: it.price }))
    } catch (err) {
      createOpen = false
      notifyError(errKey(err))
    }
  }

  function addItem() {
    items = [...items, { goods_id: goods[0]?.id ?? 0, qty: 1, price: suggestedPrice(goods[0]) }]
  }

  function removeItem(index: number) {
    items = items.filter((_, i) => i !== index)
  }

  function onGoodsChange(index: number, id: number) {
    const g = goods.find((x) => x.id === id)
    items = items.map((it, i) =>
      i === index ? { ...it, goods_id: id, price: g ? suggestedPrice(g) : it.price } : it,
    )
  }

  const draftTotal = $derived(
    items.reduce((sum, it) => sum + Number(it.qty) * Number(it.price), 0),
  )

  async function saveOrder(event: SubmitEvent) {
    event.preventDefault()
    creating = true
    const payload = {
      ...(isPurchase ? { supplier_id: Number(form.party_id) } : { customer_id: Number(form.party_id) }),
      warehouse_id: Number(form.warehouse_id),
      items: items.map((it) => ({
        goods_id: Number(it.goods_id),
        qty: Number(it.qty),
        price: Number(it.price),
      })),
    }
    try {
      if (editingID === null) await api.post(endpoint, payload)
      else await api.put(`${endpoint}/${editingID}`, payload)
      notifySuccess('notify.saved')
      createOpen = false
      await list.load()
    } catch (err) {
      notifyError(err instanceof ApiError ? err.key : 'error.internal')
    } finally {
      creating = false
    }
  }

  // ---- 状态操作 ----

  let busyID = $state<number | null>(null)

  async function act(order: OrderRow, action: string, confirmTitle?: MessageKey) {
    if (confirmTitle && !(await confirm({ titleKey: confirmTitle }))) return
    busyID = order.id
    try {
      await api.post(`${endpoint}/${order.id}/${action}`, {})
      notifySuccess('notify.saved')
      await list.load()
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      busyID = null
    }
  }

  async function withdraw(order: OrderRow) {
    if (!(await confirm({ titleKey: 'erp.order.withdrawConfirm' }))) return
    await act(order, 'withdraw')
  }

  async function removeDraft(order: OrderRow) {
    if (!(await confirm({ titleKey: 'erp.order.deleteConfirm', danger: true }))) return
    busyID = order.id
    try {
      await api.del(`${endpoint}/${order.id}`)
      notifySuccess('notify.deleted')
      await list.refreshAfterDelete()
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      busyID = null
    }
  }

  // ---- 详情 / 操作历史 ----

  let detailOpen = $state(false)
  let detail = $state<OrderRow | null>(null)
  let operations = $state<BillOperation[]>([])
  let detailLoading = $state(false)

  async function openDetail(order: OrderRow) {
    detailOpen = true
    detailLoading = true
    detail = order
    operations = []
    try {
      const res = await api.get<{ list: BillOperation[] }>(`/bills/${billType}/${order.id}/operations`)
      operations = res.list
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
      <Button size="sm" onclick={openCreate}>{t(isPurchase ? 'erp.purchase.create' : 'erp.sales.create')}</Button>
    {/if}
  </ListHeader>

  <div class="overflow-x-auto rounded-lg border bg-card">
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('erp.order.no')}</TableHead>
          <TableHead>{t(partyLabel)}</TableHead>
          <TableHead>{t('erp.order.warehouse')}</TableHead>
          <TableHead>{t('common.status')}</TableHead>
          <TableHead class="text-right">{t('erp.order.total')}</TableHead>
          <TableHead class="text-right">{t('common.actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {#if list.rows.length === 0}
          <TableEmpty loading={list.loading} colspan={6} />
        {/if}
        {#each list.rows as o (o.id)}
          <TableRow>
            <TableCell class="font-mono text-sm">{o.order_no}</TableCell>
            <TableCell>{partyOf(o)}</TableCell>
            <TableCell>{o.warehouse_name}</TableCell>
            <TableCell>
              <Badge
                variant={o.status === 'rejected'
                  ? 'destructive'
                  : o.status === 'completed'
                    ? 'default'
                    : 'secondary'}
              >
                {t(statusKey(o.status))}
              </Badge>
            </TableCell>
            <TableCell class="text-right tabular-nums">{o.total_amount.toFixed(2)}</TableCell>
            <TableCell class="text-right whitespace-nowrap">
              <Button variant="ghost" size="sm" onclick={() => openDetail(o)}>{t('erp.order.detail')}</Button>
              {#if session.can(editPerm) && o.status === 'draft'}
                <Button variant="ghost" size="sm" onclick={() => openEdit(o)}>{t('common.edit')}</Button>
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={busyID === o.id}
                  onclick={() => act(o, 'submit', 'erp.order.submitConfirm')}
                >
                  {t('erp.order.submit')}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  class="text-destructive"
                  disabled={busyID === o.id}
                  onclick={() => removeDraft(o)}
                >
                  {t('erp.order.deleteDraft')}
                </Button>
              {/if}
              {#if session.can(editPerm) && o.status === 'submitted'}
                <Button variant="ghost" size="sm" disabled={busyID === o.id} onclick={() => withdraw(o)}>
                  {t('erp.order.withdraw')}
                </Button>
              {/if}
              {#if session.can(editPerm) && o.status === 'approved'}
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={busyID === o.id}
                  onclick={() => act(o, isPurchase ? 'receive' : 'ship', 'erp.order.completeConfirm')}
                >
                  {t(completeLabel)}
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

<Dialog bind:open={createOpen}>
  <DialogContent class="sm:max-w-3xl">
    <DialogHeader>
      <DialogTitle>
        {editingID === null
          ? t(isPurchase ? 'erp.purchase.create' : 'erp.sales.create')
          : t('common.edit')}
      </DialogTitle>
    </DialogHeader>
    <form class="flex max-h-[72vh] flex-col gap-4 overflow-y-auto pr-1" onsubmit={saveOrder}>
      <div class="grid grid-cols-2 gap-4">
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t(partyLabel)}</Label>
          <NativeSelect
            class="w-full"
            value={String(form.party_id)}
            onchange={(e) => (form.party_id = Number(e.currentTarget.value))}
            required
          >
            {#each parties as p (p.id)}
              <option value={String(p.id)}>{p.name}</option>
            {/each}
          </NativeSelect>
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t('erp.order.warehouse')}</Label>
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
      </div>

      <div class="flex flex-col gap-2">
        <div class="flex items-center justify-between">
          <span class="text-sm font-medium">{t('erp.order.items')}</span>
          <Button type="button" variant="outline" size="sm" onclick={addItem}>{t('erp.order.addItem')}</Button>
        </div>
        {#each items as item, index (index)}
          <div class="grid grid-cols-[1fr_96px_112px_32px] items-end gap-2">
            <label class="flex flex-col gap-1 text-xs">
              <span class="text-muted-foreground">{t('erp.goods.name')}</span>
              <NativeSelect
                class="w-full"
                value={String(item.goods_id)}
                onchange={(e) => onGoodsChange(index, Number(e.currentTarget.value))}
              >
                {#each goods as g (g.id)}
                  <option value={String(g.id)}>{g.name}</option>
                {/each}
              </NativeSelect>
            </label>
            <label class="flex flex-col gap-1 text-xs">
              <span class="text-muted-foreground">{t('field.qty')}</span>
              <Input type="number" step="0.001" min={0} bind:value={item.qty} />
            </label>
            <label class="flex flex-col gap-1 text-xs">
              <span class="text-muted-foreground">{t('field.price')}</span>
              <Input type="number" step="0.01" min={0} bind:value={item.price} />
            </label>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              class="text-destructive"
              disabled={items.length === 1}
              onclick={() => removeItem(index)}
            >
              ×
            </Button>
          </div>
        {/each}
        <div class="text-right text-sm">
          {t('erp.order.total')}: <span class="font-semibold tabular-nums">{draftTotal.toFixed(2)}</span>
        </div>
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" onclick={() => (createOpen = false)}>{t('common.cancel')}</Button>
        <Button type="submit" disabled={creating || items.length === 0}>{t('common.save')}</Button>
      </DialogFooter>
    </form>
  </DialogContent>
</Dialog>


<Dialog bind:open={detailOpen}>
  <DialogContent class="sm:max-w-2xl">
    <DialogHeader>
      <DialogTitle>{detail?.order_no ?? t('erp.order.detail')}</DialogTitle>
    </DialogHeader>
    <div class="flex max-h-[70vh] flex-col gap-4 overflow-y-auto pr-1 text-sm">
      {#if detail}
        <div class="grid grid-cols-2 gap-2 text-xs text-muted-foreground">
          <div>{t(partyLabel)}: {partyOf(detail)}</div>
          <div>{t('erp.order.warehouse')}: {detail.warehouse_name}</div>
          <div>{t('common.status')}: {t(statusKey(detail.status))}</div>
          <div>{t('erp.order.total')}: {detail.total_amount.toFixed(2)}</div>
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
            {#each detail.items as it (it.id ?? `${it.goods_id}-${it.qty}`)}
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
          {#if detailLoading}
            <p class="text-sm text-muted-foreground">{t('common.loading')}</p>
          {:else if operations.length === 0}
            <p class="text-sm text-muted-foreground">{t('common.empty')}</p>
          {:else}
            <ul class="flex flex-col gap-1 text-xs">
              {#each operations as op (op.id)}
                <li class="flex justify-between rounded border px-2 py-1">
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
