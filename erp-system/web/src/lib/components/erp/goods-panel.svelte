<!-- 有问题欢迎联系 hot_kun@hotmail.com -->
<script lang="ts">
  import { onMount } from 'svelte'

  import { ApiError, api } from '$lib/api/client'
  import type { FieldError, Goods, Status, Unit } from '$lib/api/types'
  import FilterBar from '$lib/components/filter-bar.svelte'
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
  import { fieldErrorOf } from '$lib/i18n/errors'
  import { t } from '$lib/i18n/index.svelte'
  import { createListState, errKey } from '$lib/list-state.svelte'
  import { confirm } from '$lib/stores/confirm.svelte'
  import { notifyError, notifySuccess } from '$lib/stores/notify.svelte'
  import { session } from '$lib/stores/session.svelte'

  const list = createListState<Goods, 'keyword' | 'status'>({
    endpoint: '/goods',
    filterKeys: ['keyword', 'status'],
  })

  let units = $state<Unit[]>([])

  onMount(async () => {
    await list.load()
    try {
      const res = await api.get<{ list: Unit[] }>('/units?status=active')
      units = res.list
    } catch (err) {
      notifyError(errKey(err))
    }
  })

  type Form = {
    id: number | null
    code: string
    name: string
    spec: string
    unit_id: number | null
    unit: string
    price: number
    cost: number
    purchase_price: number
    sale_price: number
    min_qty: number
    status: Status
  }
  const emptyForm = (): Form => ({
    id: null,
    code: '',
    name: '',
    spec: '',
    unit_id: null,
    unit: '件',
    price: 0,
    cost: 0,
    purchase_price: 0,
    sale_price: 0,
    min_qty: 0,
    status: 'active',
  })

  let form = $state<Form>(emptyForm())
  let formOpen = $state(false)
  let saving = $state(false)
  let fieldErrors = $state<FieldError[]>([])
  let busyID = $state<number | null>(null)

  function openCreate() {
    form = emptyForm()
    fieldErrors = []
    formOpen = true
  }

  function openEdit(g: Goods) {
    form = {
      id: g.id,
      code: g.code,
      name: g.name,
      spec: g.spec,
      unit_id: g.unit_id,
      unit: g.unit,
      price: g.price,
      cost: g.cost,
      purchase_price: g.purchase_price,
      sale_price: g.sale_price,
      min_qty: g.min_qty,
      status: g.status,
    }
    fieldErrors = []
    formOpen = true
  }

  async function save(event: SubmitEvent) {
    event.preventDefault()
    saving = true
    fieldErrors = []
    const payload = {
      code: form.code,
      name: form.name,
      spec: form.spec,
      unit_id: form.unit_id,
      unit: form.unit,
      price: Number(form.price),
      cost: Number(form.cost),
      purchase_price: Number(form.purchase_price),
      sale_price: Number(form.sale_price),
      min_qty: Number(form.min_qty),
      status: form.status,
    }
    try {
      if (form.id === null) await api.post('/goods', payload)
      else await api.put(`/goods/${form.id}`, payload)
      notifySuccess('notify.saved')
      formOpen = false
      await list.load()
    } catch (err) {
      if (err instanceof ApiError) {
        fieldErrors = err.fieldErrors
        if (fieldErrors.length === 0) notifyError(err.key)
      } else {
        notifyError('error.internal')
      }
    } finally {
      saving = false
    }
  }

  async function remove(g: Goods) {
    if (!(await confirm({ titleKey: 'confirm.deleteTitle', bodyKey: 'confirm.deleteBody', danger: true }))) {
      return
    }
    busyID = g.id
    try {
      await api.del(`/goods/${g.id}`)
      notifySuccess('notify.deleted')
      await list.refreshAfterDelete()
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      busyID = null
    }
  }
</script>

<div class="flex flex-col gap-4">
  <FilterBar onSearch={() => list.search()} onReset={() => list.reset()}>
    <label class="flex w-56 flex-col gap-1.5 text-sm">
      <span class="text-muted-foreground">{t('erp.keyword')}</span>
      <Input placeholder={t('erp.keywordHint')} bind:value={list.filters.keyword} />
    </label>
    <label class="flex w-40 flex-col gap-1.5 text-sm">
      <span class="text-muted-foreground">{t('common.status')}</span>
      <NativeSelect class="w-full" bind:value={list.filters.status}>
        <option value="">{t('common.all')}</option>
        <option value="active">{t('common.status.active')}</option>
        <option value="inactive">{t('common.status.inactive')}</option>
      </NativeSelect>
    </label>
  </FilterBar>

  <ListHeader total={list.total}>
    {#if session.can('erp:master:edit')}
      <Button size="sm" onclick={openCreate}>{t('common.create')}</Button>
    {/if}
  </ListHeader>

  <div class="rounded-lg border bg-card">
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('field.code')}</TableHead>
          <TableHead>{t('field.name')}</TableHead>
          <TableHead>{t('field.unit')}</TableHead>
          <TableHead class="text-right">{t('field.price')}</TableHead>
          <TableHead class="text-right">{t('field.cost')}</TableHead>
          <TableHead class="text-right">{t('erp.goods.stock')}</TableHead>
          <TableHead>{t('common.status')}</TableHead>
          <TableHead class="text-right">{t('common.actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {#if list.rows.length === 0}
          <TableEmpty loading={list.loading} colspan={8} />
        {/if}
        {#each list.rows as g (g.id)}
          <TableRow>
            <TableCell class="font-mono text-sm">{g.code}</TableCell>
            <TableCell class="font-medium">{g.name}</TableCell>
            <TableCell>{g.unit}</TableCell>
            <TableCell class="text-right tabular-nums">{g.price.toFixed(2)}</TableCell>
            <TableCell class="text-right tabular-nums">{g.cost.toFixed(2)}</TableCell>
            <TableCell class="text-right tabular-nums">{g.stock_qty}</TableCell>
            <TableCell>
              <Badge variant={g.status === 'active' ? 'default' : 'secondary'}>
                {t(g.status === 'active' ? 'common.status.active' : 'common.status.inactive')}
              </Badge>
            </TableCell>
            <TableCell class="text-right whitespace-nowrap">
              {#if session.can('erp:master:edit')}
                <Button variant="ghost" size="sm" onclick={() => openEdit(g)}>{t('common.edit')}</Button>
                <Button
                  variant="ghost"
                  size="sm"
                  class="text-destructive"
                  disabled={busyID === g.id}
                  onclick={() => remove(g)}
                >
                  {t('common.delete')}
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
      <DialogTitle>{form.id === null ? t('erp.master.createGoods') : t('erp.master.editGoods')}</DialogTitle>
    </DialogHeader>
    <form class="flex flex-col gap-4" onsubmit={save}>
      <div class="grid grid-cols-2 gap-4">
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t('field.code')}</Label>
          <Input bind:value={form.code} disabled={form.id !== null} required />
          {#if fieldErrorOf(fieldErrors, 'code')}
            <span class="text-xs text-destructive">{fieldErrorOf(fieldErrors, 'code')}</span>
          {/if}
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t('field.name')}</Label>
          <Input bind:value={form.name} required />
          {#if fieldErrorOf(fieldErrors, 'name')}
            <span class="text-xs text-destructive">{fieldErrorOf(fieldErrors, 'name')}</span>
          {/if}
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t('field.spec')}</Label>
          <Input bind:value={form.spec} placeholder={t('erp.goods.specHint')} />
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t('field.unit')}</Label>
          <NativeSelect class="w-full" bind:value={form.unit_id}>
            <option value={null}>{t('erp.goods.unitFree')}</option>
            {#each units as u (u.id)}
              <option value={u.id}>{u.name}</option>
            {/each}
          </NativeSelect>
          <Input bind:value={form.unit} placeholder={t('erp.goods.unitHint')} />
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t('common.status')}</Label>
          <NativeSelect class="w-full" bind:value={form.status}>
            <option value="active">{t('common.status.active')}</option>
            <option value="inactive">{t('common.status.inactive')}</option>
          </NativeSelect>
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t('field.price')}</Label>
          <Input type="number" step="0.01" min={0} bind:value={form.price} />
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t('field.cost')}</Label>
          <Input type="number" step="0.01" min={0} bind:value={form.cost} />
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t('erp.goods.purchasePrice')}</Label>
          <Input type="number" step="0.01" min={0} bind:value={form.purchase_price} />
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t('erp.goods.salePrice')}</Label>
          <Input type="number" step="0.01" min={0} bind:value={form.sale_price} />
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t('erp.goods.minQty')}</Label>
          <Input type="number" step="0.001" min={0} bind:value={form.min_qty} />
          <span class="text-xs text-muted-foreground">{t('erp.goods.minQtyHint')}</span>
        </label>
      </div>
      <DialogFooter>
        <Button type="button" variant="outline" onclick={() => (formOpen = false)}>{t('common.cancel')}</Button>
        <Button type="submit" disabled={saving}>{t('common.save')}</Button>
      </DialogFooter>
    </form>
  </DialogContent>
</Dialog>
