<!-- 有问题欢迎联系 hot_kun@hotmail.com -->
<script lang="ts">
  import { onMount } from 'svelte'

  import { ApiError, api } from '$lib/api/client'
  import type { Brand, Category } from '$lib/api/types'
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
  import { t } from '$lib/i18n/index.svelte'
  import { errKey } from '$lib/list-state.svelte'
  import { confirm } from '$lib/stores/confirm.svelte'
  import { notifyError, notifySuccess } from '$lib/stores/notify.svelte'
  import { session } from '$lib/stores/session.svelte'

  let categories = $state<Category[]>([])
  let brands = $state<Brand[]>([])
  let categoryName = $state('')
  let categorySort = $state(0)
  let brandName = $state('')
  let saving = $state(false)

  type EditTarget = 'category' | 'brand'
  let editTarget = $state<EditTarget>('category')
  let editID = $state(0)
  let editName = $state('')
  let editSort = $state(0)
  let editOpen = $state(false)
  let editSaving = $state(false)

  function openEditCategory(c: Category) {
    editTarget = 'category'
    editID = c.id
    editName = c.name
    editSort = c.sort
    editOpen = true
  }

  function openEditBrand(b: Brand) {
    editTarget = 'brand'
    editID = b.id
    editName = b.name
    editOpen = true
  }

  async function saveEdit(event: SubmitEvent) {
    event.preventDefault()
    editSaving = true
    try {
      if (editTarget === 'category') {
        await api.put(`/categories/${editID}`, { name: editName, sort: Number(editSort) })
      } else {
        await api.put(`/brands/${editID}`, { name: editName })
      }
      notifySuccess('notify.saved')
      editOpen = false
      await refresh()
    } catch (err) {
      notifyError(err instanceof ApiError ? err.key : 'error.internal')
    } finally {
      editSaving = false
    }
  }

  async function remove(target: EditTarget, id: number) {
    if (!(await confirm({ titleKey: 'confirm.deleteTitle', bodyKey: 'confirm.deleteBody', danger: true }))) {
      return
    }
    try {
      await api.del(`/${target === 'category' ? 'categories' : 'brands'}/${id}`)
      notifySuccess('notify.deleted')
      await refresh()
    } catch (err) {
      notifyError(errKey(err))
    }
  }

  onMount(async () => {
    await refresh()
  })

  async function refresh() {
    try {
      const [c, b] = await Promise.all([
        api.get<{ list: Category[] }>('/categories'),
        api.get<{ list: Brand[] }>('/brands'),
      ])
      categories = c.list
      brands = b.list
    } catch (err) {
      notifyError(errKey(err))
    }
  }

  async function addCategory(event: SubmitEvent) {
    event.preventDefault()
    saving = true
    try {
      await api.post('/categories', { name: categoryName, sort: Number(categorySort) })
      notifySuccess('notify.saved')
      categoryName = ''
      categorySort = 0
      await refresh()
    } catch (err) {
      notifyError(err instanceof ApiError ? err.key : 'error.internal')
    } finally {
      saving = false
    }
  }

  async function addBrand(event: SubmitEvent) {
    event.preventDefault()
    saving = true
    try {
      await api.post('/brands', { name: brandName })
      notifySuccess('notify.saved')
      brandName = ''
      await refresh()
    } catch (err) {
      notifyError(err instanceof ApiError ? err.key : 'error.internal')
    } finally {
      saving = false
    }
  }
</script>

<div class="grid gap-4 lg:grid-cols-2">
  <section class="rounded-lg border bg-card p-4">
    <h2 class="mb-3 text-sm font-medium">{t('erp.category.title')}</h2>
    {#if categories.length === 0}
      <p class="py-3 text-sm text-muted-foreground">{t('common.empty')}</p>
    {:else}
      <ul class="flex flex-col gap-1">
        {#each categories as c (c.id)}
          <li class="flex items-center justify-between rounded border px-3 py-2 text-sm">
            <span>{c.name} <span class="text-xs text-muted-foreground">#{c.sort}</span></span>
            {#if session.can('erp:master:edit')}
              <span class="flex gap-1">
                <Button variant="ghost" size="sm" onclick={() => openEditCategory(c)}>{t('common.edit')}</Button>
                <Button variant="ghost" size="sm" class="text-destructive" onclick={() => remove('category', c.id)}>
                  {t('common.delete')}
                </Button>
              </span>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
    {#if session.can('erp:master:edit')}
      <form class="mt-4 flex items-end gap-2 border-t pt-4" onsubmit={addCategory}>
        <label class="flex flex-1 flex-col gap-1.5 text-sm">
          <Label>{t('erp.category.name')}</Label>
          <Input bind:value={categoryName} required />
        </label>
        <label class="flex w-24 flex-col gap-1.5 text-sm">
          <Label>{t('field.sort')}</Label>
          <Input type="number" bind:value={categorySort} min={0} />
        </label>
        <Button type="submit" size="sm" disabled={saving}>{t('common.create')}</Button>
      </form>
    {/if}
  </section>

  <section class="rounded-lg border bg-card p-4">
    <h2 class="mb-3 text-sm font-medium">{t('erp.brand.title')}</h2>
    {#if brands.length === 0}
      <p class="py-3 text-sm text-muted-foreground">{t('common.empty')}</p>
    {:else}
      <ul class="flex flex-col gap-1">
        {#each brands as b (b.id)}
          <li class="flex items-center justify-between rounded border px-3 py-2 text-sm">
            <span>{b.name}</span>
            {#if session.can('erp:master:edit')}
              <span class="flex gap-1">
                <Button variant="ghost" size="sm" onclick={() => openEditBrand(b)}>{t('common.edit')}</Button>
                <Button variant="ghost" size="sm" class="text-destructive" onclick={() => remove('brand', b.id)}>
                  {t('common.delete')}
                </Button>
              </span>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
    {#if session.can('erp:master:edit')}
      <form class="mt-4 flex items-end gap-2 border-t pt-4" onsubmit={addBrand}>
        <label class="flex flex-1 flex-col gap-1.5 text-sm">
          <Label>{t('erp.brand.name')}</Label>
          <Input bind:value={brandName} required />
        </label>
        <Button type="submit" size="sm" disabled={saving}>{t('common.create')}</Button>
      </form>
    {/if}
  </section>
</div>

<Dialog bind:open={editOpen}>
  <DialogContent class="sm:max-w-md">
    <DialogHeader>
      <DialogTitle>{editTarget === 'category' ? t('erp.category.title') : t('erp.brand.title')}</DialogTitle>
    </DialogHeader>
    <form class="flex flex-col gap-4" onsubmit={saveEdit}>
      <label class="flex flex-col gap-1.5 text-sm">
        <Label>{editTarget === 'category' ? t('erp.category.name') : t('erp.brand.name')}</Label>
        <Input bind:value={editName} required />
      </label>
      {#if editTarget === 'category'}
        <label class="flex flex-col gap-1.5 text-sm">
          <Label>{t('field.sort')}</Label>
          <Input type="number" bind:value={editSort} min={0} />
        </label>
      {/if}
      <DialogFooter>
        <Button type="button" variant="outline" onclick={() => (editOpen = false)}>{t('common.cancel')}</Button>
        <Button type="submit" disabled={editSaving}>{t('common.save')}</Button>
      </DialogFooter>
    </form>
  </DialogContent>
</Dialog>
