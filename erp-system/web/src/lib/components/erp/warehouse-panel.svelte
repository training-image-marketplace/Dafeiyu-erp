<!-- 有问题欢迎联系 hot_kun@hotmail.com -->
<script lang="ts">
  import { onMount } from 'svelte'

  import { ApiError, api } from '$lib/api/client'
  import type { FieldError, Status, Warehouse } from '$lib/api/types'
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

  const list = createListState<Warehouse, 'keyword' | 'status'>({
    endpoint: '/warehouses',
    filterKeys: ['keyword', 'status'],
  })

  onMount(() => {
    void list.load()
  })

  type Form = {
    id: number | null
    code: string
    name: string
    location: string
    keeper: string
    phone: string
    address: string
    status: Status
  }
  const emptyForm = (): Form => ({
    id: null,
    code: '',
    name: '',
    location: '',
    keeper: '',
    phone: '',
    address: '',
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

  function openEdit(w: Warehouse) {
    form = {
      id: w.id,
      code: w.code,
      name: w.name,
      location: w.location,
      keeper: w.keeper,
      phone: w.phone,
      address: w.address,
      status: w.status,
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
      location: form.location,
      keeper: form.keeper,
      phone: form.phone,
      address: form.address,
      status: form.status,
    }
    try {
      if (form.id === null) await api.post('/warehouses', payload)
      else await api.put(`/warehouses/${form.id}`, payload)
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

  async function remove(w: Warehouse) {
    if (!(await confirm({ titleKey: 'confirm.deleteTitle', bodyKey: 'confirm.deleteBody', danger: true }))) {
      return
    }
    busyID = w.id
    try {
      await api.del(`/warehouses/${w.id}`)
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
          <TableHead>{t('field.location')}</TableHead>
          <TableHead>{t('common.status')}</TableHead>
          <TableHead class="text-right">{t('common.actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {#if list.rows.length === 0}
          <TableEmpty loading={list.loading} colspan={5} />
        {/if}
        {#each list.rows as w (w.id)}
          <TableRow>
            <TableCell class="font-mono text-sm">{w.code}</TableCell>
            <TableCell class="font-medium">{w.name}</TableCell>
            <TableCell class="text-muted-foreground">{w.location || '—'}</TableCell>
            <TableCell>
              <Badge variant={w.status === 'active' ? 'default' : 'secondary'}>
                {t(w.status === 'active' ? 'common.status.active' : 'common.status.inactive')}
              </Badge>
            </TableCell>
            <TableCell class="text-right whitespace-nowrap">
              {#if session.can('erp:master:edit')}
                <Button variant="ghost" size="sm" onclick={() => openEdit(w)}>{t('common.edit')}</Button>
                <Button
                  variant="ghost"
                  size="sm"
                  class="text-destructive"
                  disabled={busyID === w.id}
                  onclick={() => remove(w)}
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
      <DialogTitle>{form.id === null ? t('erp.master.createWarehouse') : t('erp.master.editWarehouse')}</DialogTitle>
    </DialogHeader>
    <form class="flex flex-col gap-4" onsubmit={save}>
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
      </label>
      <label class="flex flex-col gap-1.5 text-sm">
        <Label>{t('field.location')}</Label>
        <Input bind:value={form.location} />
      </label>
      <label class="flex flex-col gap-1.5 text-sm">
        <Label>{t('field.keeper')}</Label>
        <Input bind:value={form.keeper} />
      </label>
      <label class="flex flex-col gap-1.5 text-sm">
        <Label>{t('field.phone')}</Label>
        <Input bind:value={form.phone} />
      </label>
      <label class="flex flex-col gap-1.5 text-sm">
        <Label>{t('field.address')}</Label>
        <Input bind:value={form.address} />
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
