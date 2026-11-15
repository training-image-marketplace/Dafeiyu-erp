<!-- 有问题欢迎联系 hot_kun@hotmail.com -->
<script lang="ts">
  import { onMount } from 'svelte'

  import { api } from '$lib/api/client'
  import type {
    AuditBill,
    AuditBillType,
    BillOperation,
    OrderStatus,
    PurchaseOrder,
    ReturnOrder,
    SalesOrder,
    StockAdjustment,
    StockCheck,
    StockTransfer,
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

  type Direction = 'in' | 'out'
  type BillDetail = PurchaseOrder | SalesOrder | ReturnOrder | StockAdjustment | StockTransfer | StockCheck

  const statuses: OrderStatus[] = ['draft', 'submitted', 'approved', 'completed', 'rejected']
  const auditPerm = 'erp:audit:approve'

  const list = createListState<AuditBill, 'direction' | 'status' | 'keyword'>({
    endpoint: '/audits',
    filterKeys: ['direction', 'status', 'keyword'],
  })

  let { direction }: { direction: Direction } = $props()

  let statusFilter = $state('submitted')
  let keyword = $state('')

  const statusKey = (s: OrderStatus): MessageKey => `erp.status.${s}`
  const typeKey = (tp: AuditBillType): MessageKey => `erp.audit.type.${tp}` as MessageKey
  const actionKey = (action: string): MessageKey => `erp.action.${action}` as MessageKey

  onMount(() => {
    list.filters.direction = direction
    statusFilter = list.filters.status || 'submitted'
    keyword = list.filters.keyword
    reload()
  })

  function reload() {
    list.filters.direction = direction
    list.filters.status = statusFilter
    list.filters.keyword = keyword.trim()
    list.search()
  }


  function resetFilters() {
    statusFilter = 'submitted'
    keyword = ''
    reload()
  }

  // ---- 详情 ----

  let detailOpen = $state(false)
  let detailLoading = $state(false)
  let detail = $state<BillDetail | null>(null)
  let detailMeta = $state<AuditBill | null>(null)
  let operations = $state<BillOperation[]>([])

  function detailPath(bill: AuditBill) {
    switch (bill.bill_type) {
      case 'purchase':
        return `/purchases/${bill.bill_id}`
      case 'sales':
        return `/sales/${bill.bill_id}`
      case 'purchase-return':
        return `/purchase-returns/${bill.bill_id}`
      case 'sale-return':
        return `/sales-returns/${bill.bill_id}`
      case 'stock-adjustment':
        return `/stock-adjustments/${bill.bill_id}`
      case 'transfer':
        return `/transfers/${bill.bill_id}`
      case 'stock-check':
        return `/stock-checks/${bill.bill_id}`
    }
  }

  async function openDetail(bill: AuditBill) {
    detailOpen = true
    detailLoading = true
    detail = null
    detailMeta = bill
    operations = []
    try {
      const [full, ops] = await Promise.all([
        api.get<BillDetail>(detailPath(bill)),
        api.get<{ list: BillOperation[] }>(`/bills/${bill.bill_type}/${bill.bill_id}/operations`),
      ])
      detail = full
      operations = ops.list
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      detailLoading = false
    }
  }

  const partnerOf = (bill: BillDetail) =>
    (bill as PurchaseOrder).supplier_name ?? (bill as SalesOrder).customer_name ?? '—'

  // ---- 审核 ----

  let busyID = $state('')

  async function approve(bill: AuditBill) {
    if (!(await confirm({ titleKey: 'erp.audit.passConfirm' }))) return
    const key = `${bill.bill_type}-${bill.bill_id}`
    busyID = key
    try {
      await api.post(`/audits/${bill.bill_type}/${bill.bill_id}`, { pass: true })
      notifySuccess('notify.saved')
      await list.load()
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      busyID = ''
    }
  }

  let rejectOpen = $state(false)
  let rejectTarget = $state<AuditBill | null>(null)
  let rejectComment = $state('')
  let rejectError = $state(false)
  let rejectBusy = $state(false)

  function openReject(bill: AuditBill) {
    rejectTarget = bill
    rejectComment = ''
    rejectError = false
    rejectOpen = true
  }

  async function submitReject(event: SubmitEvent) {
    event.preventDefault()
    if (!rejectTarget) return
    if (!rejectComment.trim()) {
      rejectError = true
      return
    }
    rejectBusy = true
    try {
      await api.post(`/audits/${rejectTarget.bill_type}/${rejectTarget.bill_id}`, {
        pass: false,
        comment: rejectComment.trim(),
      })
      notifySuccess('notify.saved')
      rejectOpen = false
      await list.load()
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      rejectBusy = false
    }
  }
</script>

<div class="flex flex-col gap-4">
  <ListHeader total={list.total} />

  <div class="flex flex-wrap items-center gap-2">
    <form
      class="flex flex-wrap items-end gap-2"
      onsubmit={(e) => {
        e.preventDefault()
        reload()
      }}
    >
      <label class="flex flex-col gap-1 text-xs">
        <span class="text-muted-foreground">{t('common.status')}</span>
        <NativeSelect class="w-36" bind:value={statusFilter}>
          <option value="">{t('erp.audit.allStatus')}</option>
          {#each statuses as s (s)}
            <option value={s}>{t(statusKey(s))}</option>
          {/each}
        </NativeSelect>
      </label>
      <label class="flex flex-col gap-1 text-xs">
        <span class="text-muted-foreground">{t('erp.audit.keyword')}</span>
        <Input class="w-56" bind:value={keyword} />
      </label>
      <Button type="submit" variant="outline" size="sm">{t('common.search')}</Button>
      <Button type="button" variant="ghost" size="sm" onclick={resetFilters}>{t('common.reset')}</Button>
    </form>
    <p class="ml-auto text-xs text-muted-foreground">{t('erp.audit.submittedHint')}</p>
  </div>

  <div class="overflow-x-auto rounded-lg border bg-card">
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('erp.audit.type')}</TableHead>
          <TableHead>{t('erp.order.no')}</TableHead>
          <TableHead>{t('erp.audit.partner')}</TableHead>
          <TableHead>{t('erp.order.warehouse')}</TableHead>
          <TableHead>{t('erp.audit.amountOrQty')}</TableHead>
          <TableHead>{t('common.status')}</TableHead>
          <TableHead>{t('erp.order.creator')}</TableHead>
          <TableHead>{t('erp.order.submittedAt')}</TableHead>
          <TableHead class="text-right">{t('common.actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {#if list.rows.length === 0}
          <TableEmpty loading={list.loading} colspan={9} />
        {/if}
        {#each list.rows as bill (`${bill.bill_type}-${bill.bill_id}`)}
          {@const key = `${bill.bill_type}-${bill.bill_id}`}
          <TableRow>
            <TableCell>
              <Badge variant="outline">{t(typeKey(bill.bill_type))}</Badge>
            </TableCell>
            <TableCell class="font-mono text-sm">{bill.bill_no}</TableCell>
            <TableCell>{bill.partner_name || '—'}</TableCell>
            <TableCell>{bill.warehouse_name}</TableCell>
            <TableCell class="text-right tabular-nums">
              {#if bill.bill_type === 'stock-adjustment'}
                {bill.direction === 'in' ? '+' : '−'}{bill.qty}
              {:else if bill.bill_type === 'transfer' || bill.bill_type === 'stock-check'}
                —
              {:else}
                {bill.total_amount.toFixed(2)}
              {/if}
            </TableCell>
            <TableCell>
              <Badge
                variant={bill.status === 'rejected'
                  ? 'destructive'
                  : bill.status === 'completed'
                    ? 'default'
                    : 'secondary'}
              >
                {t(statusKey(bill.status))}
              </Badge>
            </TableCell>
            <TableCell>{bill.creator_name || '—'}</TableCell>
            <TableCell class="text-xs text-muted-foreground">{bill.submitted_at ?? '—'}</TableCell>
            <TableCell class="text-right whitespace-nowrap">
              <Button variant="ghost" size="sm" onclick={() => openDetail(bill)}>{t('erp.order.detail')}</Button>
              {#if session.can(auditPerm) && bill.status === 'submitted'}
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={busyID === key}
                  onclick={() => approve(bill)}
                >
                  {t('erp.audit.pass')}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  class="text-destructive"
                  disabled={busyID === key}
                  onclick={() => openReject(bill)}
                >
                  {t('erp.audit.reject')}
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

<Dialog bind:open={detailOpen}>
  <DialogContent class="sm:max-w-2xl">
    <DialogHeader>
      <DialogTitle>{detailMeta?.bill_no ?? t('erp.order.detail')}</DialogTitle>
    </DialogHeader>
    <div class="flex max-h-[70vh] flex-col gap-4 overflow-y-auto pr-1 text-sm">
      {#if detailLoading}
        <p class="text-sm text-muted-foreground">{t('common.loading')}</p>
      {:else if detail && detailMeta}
        <div class="grid grid-cols-2 gap-2 text-xs text-muted-foreground">
          <div>{t('erp.audit.type')}: {t(typeKey(detailMeta.bill_type))}</div>
          {#if 'direction' in detail}
            <div>{t('erp.order.warehouse')}: {detail.warehouse_name}</div>
            <div>{t('erp.goods.name')}: {detail.goods_name}</div>
            <div>{t('erp.stock.direction')}: {t(`erp.stock.direction.${detail.direction}` as const)}</div>
            <div>{t('erp.stock.qty')}: {detail.qty} {detail.unit}</div>
            <div>{t('erp.stock.reason')}: {detail.reason || '—'}</div>
          {:else if 'from_warehouse_name' in detail}
            <div>{t('erp.transfer.from')}: {detail.from_warehouse_name}</div>
            <div>{t('erp.transfer.to')}: {detail.to_warehouse_name}</div>
            <div>{t('field.remark')}: {detail.remark || '—'}</div>
          {:else if 'check_no' in detail}
            <div>{t('erp.order.warehouse')}: {detail.warehouse_name}</div>
            <div>{t('field.remark')}: {detail.remark || '—'}</div>
          {:else}
            <div>{t('erp.order.warehouse')}: {detail.warehouse_name}</div>
            <div>{t('erp.audit.partner')}: {detailMeta.partner_name || partnerOf(detail)}</div>
            <div>{t('erp.order.total')}: {detail.total_amount.toFixed(2)}</div>
            <div>{t('erp.order.remark')}: {'remark' in detail ? detail.remark || '—' : '—'}</div>
          {/if}
          <div>{t('common.status')}: {t(statusKey(detail.status))}</div>
          <div>{t('erp.order.creator')}: {detailMeta.creator_name || '—'}</div>
          <div>{t('erp.order.approver')}: {detailMeta.approver_name || '—'}</div>
        </div>

        {#if 'direction' in detail}
          <!-- 库存调整：单商品，无明细表 -->
        {:else if 'from_warehouse_name' in detail}
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('erp.goods.name')}</TableHead>
                <TableHead>{t('field.unit')}</TableHead>
                <TableHead class="text-right">{t('field.qty')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {#each detail.items ?? [] as it (it.id)}
                <TableRow>
                  <TableCell>{it.goods_name}</TableCell>
                  <TableCell>{it.unit}</TableCell>
                  <TableCell class="text-right tabular-nums">{it.qty}</TableCell>
                </TableRow>
              {/each}
            </TableBody>
          </Table>
        {:else if 'check_no' in detail}
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('erp.goods.name')}</TableHead>
                <TableHead class="text-right">{t('erp.stockCheck.bookQty')}</TableHead>
                <TableHead class="text-right">{t('erp.stockCheck.actualQty')}</TableHead>
                <TableHead class="text-right">{t('erp.stockCheck.diffQty')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {#each detail.items ?? [] as it (it.id)}
                <TableRow>
                  <TableCell>{it.goods_name}</TableCell>
                  <TableCell class="text-right tabular-nums">{it.book_qty}</TableCell>
                  <TableCell class="text-right tabular-nums">{it.actual_qty}</TableCell>
                  <TableCell class="text-right tabular-nums">{it.diff_qty}</TableCell>
                </TableRow>
              {/each}
            </TableBody>
          </Table>
        {:else}
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
        {/if}

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

<Dialog bind:open={rejectOpen}>
  <DialogContent class="sm:max-w-md">
    <DialogHeader>
      <DialogTitle>{t('erp.audit.rejectTitle')}</DialogTitle>
    </DialogHeader>
    <form class="flex flex-col gap-4" onsubmit={submitReject}>
      <p class="text-sm text-muted-foreground">
        {rejectTarget?.bill_no} · {rejectTarget ? t(typeKey(rejectTarget.bill_type)) : ''}
      </p>
      <label class="flex flex-col gap-1.5 text-sm">
        <Label for="audit-reject-comment">{t('erp.order.rejectReason')}</Label>
        <textarea
          id="audit-reject-comment"
          class="min-h-24 w-full rounded-md border bg-transparent px-3 py-2 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50"
          maxlength={500}
          bind:value={rejectComment}
          oninput={() => (rejectError = false)}
          required
        ></textarea>
        <span class="text-xs text-muted-foreground">{t('erp.order.rejectReasonHint')}</span>
        {#if rejectError}
          <span class="text-xs text-destructive">{t('erp.order.rejectRequired')}</span>
        {/if}
      </label>
      <DialogFooter>
        <Button type="button" variant="outline" onclick={() => (rejectOpen = false)}>{t('common.cancel')}</Button>
        <Button type="submit" variant="destructive" disabled={rejectBusy}>{t('erp.audit.reject')}</Button>
      </DialogFooter>
    </form>
  </DialogContent>
</Dialog>
