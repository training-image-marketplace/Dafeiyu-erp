<!-- 有问题欢迎联系 hot_kun@hotmail.com -->
<script lang="ts">
  import { onMount } from 'svelte'

  import { ApiError, api } from '$lib/api/client'
  import type { Customer, FieldError, Status } from '$lib/api/types'
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
  import { t, type MessageKey } from '$lib/i18n/index.svelte'
  import { createListState, errKey } from '$lib/list-state.svelte'
  import { confirm } from '$lib/stores/confirm.svelte'
  import { notifyError, notifySuccess } from '$lib/stores/notify.svelte'
  import { session } from '$lib/stores/session.svelte'

  let { endpoint, titleKey }: { endpoint: string; titleKey: MessageKey } = $props()

  // svelte-ignore state_referenced_locally
  // endpoint 在一次挂载内不会变：客户/供应商各用一个实例。
  const list = createListState<Customer, 'keyword' | 'status'>({
    endpoint,
    filterKeys: ['keyword', 'status'],
  })

  onMount(() => {
    void list.load()
  })

  type Form = {
    id: number | null
    code: string
    name: string
    contact: string
    phone: string
    address: string
    payment_method: string
    status: Status
  }
  const emptyForm = (): Form => ({
    id: null,
    code: '',
    name: '',
    contact: '',
    phone: '',
    address: '',
    payment_method: '',
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

  function openEdit(item: Customer) {
    form = {
      id: item.id,
      code: item.code,
      name: item.name,
      contact: item.contact,
      phone: item.phone,
      address: item.address,
      payment_method: item.payment_method,
      status: item.status,
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
      contact: form.contact,
      phone: form.phone,
      address: form.address,
      payment_method: form.payment_method,
      status: form.status,
    }
    try {
      if (form.id === null) await api.post(endpoint, payload)
      else await api.put(`${endpoint}/${form.id}`, payload)
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

  async function remove(item: Customer) {
    if (!(await confirm({ titleKey: 'confirm.deleteTitle', bodyKey: 'confirm.deleteBody', danger: true }))) {
      return
    }
    busyID = item.id
    try {
      await api.del(`${endpoint}/${item.id}`)
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
          <TableHead>{t('field.contact')}</TableHead>
          <TableHead>{t('field.phone')}</TableHead>
          <TableHead>{t('common.status')}</TableHead>
          <TableHead class="text-right">{t('common.actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {#if list.rows.length === 0}
          <TableEmpty loading={list.loading} colspan={6} />
        {/if}
        {#each list.rows as item (item.id)}
          <TableRow>
            <TableCell class="font-mono text-sm">{item.code}</TableCell>
            <TableCell class="font-medium">{item.name}</TableCell>
            <TableCell class="text-muted-foreground">{item.contact || '—'}</TableCell>
            <TableCell class="text-muted-foreground">{item.phone || '—'}</TableCell>
            <TableCell>
              <Badge variant={item.status === 'active' ? 'default' : 'secondary'}>
                {t(item.status === 'active' ? 'common.status.active' : 'common.status.inactive')}
              </Badge>
            </TableCell>
            <TableCell class="text-right whitespace-nowrap">
              {#if session.can('erp:master:edit')}
                <Button variant="ghost" size="sm" onclick={() => openEdit(item)}>{t('common.edit')}</Button>
                <Button
                  variant="ghost"
                  size="sm"
                  class="text-destructive"
                  disabled={busyID === item.id}
                  onclick={() => remove(item)}
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
      <DialogTitle>{form.id === null ? t(titleKey) : t('common.edit')}</DialogTitle>
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
          <Label>{t('field.contact')}</Label>
          <Input bind:value={form.contact} />
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t('field.phone')}</Label>
          <Input bind:value={form.phone} />
        </label>
      </div>
      <label class="flex flex-col gap-1.5 text-sm">
        <Label>{t('field.address')}</Label>
        <Input bind:value={form.address} />
      </label>
      <label class="flex flex-col gap-1.5 text-sm">
        <Label>{t('field.paymentMethod')}</Label>
        <NativeSelect class="w-full" bind:value={form.payment_method}>
          <option value="">{t('erp.party.paymentNone')}</option>
          <option value="现结">{t('erp.party.paymentCash')}</option>
          <option value="月结">{t('erp.party.paymentMonthly')}</option>
          <option value="货到付款">{t('erp.party.paymentCod')}</option>
        </NativeSelect>
      </label>
      <label class="flex flex-col gap-1.5 text-sm">
        <Label>{t('common.status')}</Label>
        <NativeSelect class="w-full" bind:value={form.status}>
          <option value="active">{t('common.status.active')}</option>
          <option value="inactive">{t('common.status.inactive')}</option>
        </NativeSelect>
      </label>
      <DialogFooter>
        <Button type="button" variant="outline" onclick={() => (formOpen = false)}>{t('common.cancel')}</Button>
        <Button type="submit" disabled={saving}>{t('common.save')}</Button>
      </DialogFooter>
    </form>
  </DialogContent>
</Dialog>
