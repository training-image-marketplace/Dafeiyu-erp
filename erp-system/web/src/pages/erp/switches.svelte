<script lang="ts">
  import { onMount } from 'svelte'

  import { api } from '$lib/api/client'
  import type { BugSwitch } from '$lib/api/types'
  import { Badge } from '$lib/components/ui/badge'
  import { Button } from '$lib/components/ui/button'
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
  import { notifyError, notifySuccess } from '$lib/stores/notify.svelte'
  import { session } from '$lib/stores/session.svelte'

  // 缺陷目录是静态声明，当前开关状态来自 /switches 的 config.bugs。
  let catalog = $state<BugSwitch[]>([])
  let enabled = $state<Record<string, boolean>>({})
  let scope = $state('all')
  let loading = $state(true)
  let busyCode = $state('')

  const canEdit = $derived(session.can('erp:switch:edit'))
  const wiredCount = $derived(catalog.filter((b) => b.wired).length)

  onMount(load)

  async function load() {
    loading = true
    try {
      const [bugs, cfg] = await Promise.all([
        api.get<{ list: BugSwitch[] }>('/switches/bugs'),
        api.get<{ scope: string; config: { bugs?: Record<string, boolean> } }>('/switches'),
      ])
      catalog = bugs.list
      enabled = cfg.config.bugs ?? {}
      scope = cfg.scope
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      loading = false
    }
  }

  async function toggle(bug: BugSwitch) {
    busyCode = bug.code
    try {
      await api.post('/switches/bug', { bug_code: bug.code, enabled: !enabled[bug.code] })
      notifySuccess('notify.saved')
      await load()
    } catch (err) {
      notifyError(errKey(err))
    } finally {
      busyCode = ''
    }
  }
</script>

<div class="flex flex-col gap-4">
  <div class="rounded-xl border border-amber-300/60 bg-amber-50 px-4 py-3 text-sm text-amber-900">
    {t('switch.warning')}
  </div>

  <div class="flex flex-wrap items-center justify-between gap-3 text-sm">
    <p class="text-muted-foreground">
      {t('switch.scope')}: <span class="font-medium text-foreground">{t('switch.scopeAll')}</span>
      <span class="ml-1 font-mono text-xs">({scope})</span>
      <span class="ml-4">{t('switch.total', { count: catalog.length, wired: wiredCount })}</span>
    </p>
    {#if !canEdit}
      <p class="text-muted-foreground">{t('switch.readonly')}</p>
    {/if}
  </div>

  <div class="overflow-x-auto rounded-lg border bg-card">
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('switch.code')}</TableHead>
          <TableHead>{t('switch.name')}</TableHead>
          <TableHead>{t('switch.area')}</TableHead>
          <TableHead>{t('switch.state')}</TableHead>
          <TableHead class="text-right">{t('common.actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {#if catalog.length === 0}
          <TableRow>
            <TableCell colspan={5} class="py-10 text-center text-sm text-muted-foreground">
              {loading ? t('common.loading') : t('common.empty')}
            </TableCell>
          </TableRow>
        {/if}
        {#each catalog as bug (bug.code)}
          <TableRow>
            <TableCell class="font-mono text-xs">{bug.code}</TableCell>
            <TableCell>
              {bug.name}
              {#if !bug.wired}
                <span class="mt-1 block text-xs text-muted-foreground">{t('switch.unwiredHint')}</span>
              {/if}
            </TableCell>
            <TableCell class="font-mono text-xs text-muted-foreground">{bug.area}</TableCell>
            <TableCell>
              {#if !bug.wired}
                <Badge variant="outline">{t('switch.unwired')}</Badge>
              {:else if enabled[bug.code]}
                <Badge variant="destructive">{t('switch.on')}</Badge>
              {:else}
                <Badge variant="secondary">{t('switch.off')}</Badge>
              {/if}
            </TableCell>
            <TableCell class="text-right">
              {#if canEdit && bug.wired}
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={busyCode === bug.code}
                  onclick={() => toggle(bug)}
                >
                  {enabled[bug.code] ? t('switch.disable') : t('switch.enable')}
                </Button>
              {/if}
            </TableCell>
          </TableRow>
        {/each}
      </TableBody>
    </Table>
  </div>
</div>
