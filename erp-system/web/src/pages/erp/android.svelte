<script lang="ts">
  import { onMount } from 'svelte'

  import CheckCircle2 from '@lucide/svelte/icons/check-circle-2'
  import Download from '@lucide/svelte/icons/download'
  import ShieldAlert from '@lucide/svelte/icons/shield-alert'
  import Smartphone from '@lucide/svelte/icons/smartphone'

  import { Badge } from '$lib/components/ui/badge'
  import { Button } from '$lib/components/ui/button'
  import { t } from '$lib/i18n/index.svelte'

  /** APK 放在 web/public/apk/ 下，构建后由后端按 /apk 静态托管；
   *  部署后也可以直接丢进 dist/apk/，不用重新构建。 */
  const APK_PATH = '/apk/dafeiyu-erp.apk'

  let available = $state<boolean | null>(null)
  let sizeLabel = $state('')

  onMount(async () => {
    try {
      const res = await fetch(APK_PATH, { method: 'HEAD' })
      available = res.ok
      const bytes = Number(res.headers.get('content-length') ?? 0)
      if (bytes > 0) sizeLabel = formatBytes(bytes)
    } catch {
      available = false
    }
  })

  function formatBytes(bytes: number): string {
    if (bytes >= 1024 ** 2) return `${(bytes / 1024 ** 2).toFixed(1)} MB`
    return `${(bytes / 1024).toFixed(0)} KB`
  }

  const features = ['android.f1', 'android.f2', 'android.f3', 'android.f4', 'android.f5', 'android.f6'] as const
  const limits = ['android.l1', 'android.l2', 'android.l3'] as const
  const steps = ['android.s1', 'android.s2', 'android.s3'] as const
</script>

<div class="flex flex-col gap-4">
  <section class="relative overflow-hidden rounded-xl border border-emerald-700/20 bg-gradient-to-br from-emerald-700 via-teal-700 to-cyan-800 p-6 text-white shadow-sm">
    <div class="relative flex flex-wrap items-start justify-between gap-6">
      <div class="max-w-2xl">
        <p class="flex items-center gap-2 text-xs font-medium tracking-[0.2em] text-emerald-100/80 uppercase">
          <Smartphone class="size-4" aria-hidden="true" />
          Android
        </p>
        <h2 class="mt-2 text-2xl font-semibold">{t('menu.erp.android')}</h2>
        <p class="mt-2 text-sm text-emerald-50/85">{t('android.intro')}</p>
        <p class="mt-3 text-xs text-emerald-50/70">{t('android.requirements')}</p>
      </div>

      <div class="flex shrink-0 flex-col items-start gap-2 rounded-lg border border-white/15 bg-white/10 px-4 py-3 backdrop-blur-sm">
        {#if available}
          <a
            href={APK_PATH}
            download="dafeiyu-erp.apk"
            class="inline-flex items-center gap-2 rounded-md bg-white px-4 py-2 text-sm font-medium text-emerald-800 transition hover:bg-emerald-50"
          >
            <Download class="size-4" aria-hidden="true" />
            {t('android.download')}
          </a>
          {#if sizeLabel}
            <span class="text-xs text-emerald-50/80">{t('android.size', { size: sizeLabel })}</span>
          {/if}
        {:else if available === false}
          <Button variant="outline" size="sm" disabled class="border-white/25 bg-white/10 text-white">
            {t('android.download')}
          </Button>
          <span class="max-w-xs text-xs text-emerald-50/80">{t('android.notUploaded')}</span>
        {:else}
          <span class="text-xs text-emerald-50/80">{t('common.loading')}</span>
        {/if}
      </div>
    </div>
  </section>

  <div class="grid gap-4 lg:grid-cols-2">
    <section class="rounded-xl border bg-card">
      <div class="border-b px-4 py-3">
        <h3 class="flex items-center gap-2 text-sm font-medium">
          <CheckCircle2 class="size-4 text-emerald-600" aria-hidden="true" />
          {t('android.featureList')}
        </h3>
      </div>
      <ul class="flex flex-col gap-2 p-4 text-sm">
        {#each features as key (key)}
          <li class="flex gap-2">
            <span class="mt-1 size-1.5 shrink-0 rounded-full bg-emerald-500" aria-hidden="true"></span>
            <span>{t(key)}</span>
          </li>
        {/each}
      </ul>
    </section>

    <div class="flex flex-col gap-4">
      <section class="rounded-xl border bg-card">
        <div class="border-b px-4 py-3">
          <h3 class="text-sm font-medium">{t('android.limits')}</h3>
        </div>
        <ul class="flex flex-col gap-2 p-4 text-sm text-muted-foreground">
          {#each limits as key (key)}
            <li class="flex gap-2">
              <span class="mt-1 size-1.5 shrink-0 rounded-full bg-muted-foreground/40" aria-hidden="true"></span>
              <span>{t(key)}</span>
            </li>
          {/each}
        </ul>
      </section>

      <section class="rounded-xl border bg-card">
        <div class="border-b px-4 py-3">
          <h3 class="text-sm font-medium">{t('android.install')}</h3>
        </div>
        <ol class="flex flex-col gap-2 p-4 text-sm">
          {#each steps as key, i (key)}
            <li class="flex gap-3">
              <Badge variant="secondary" class="mt-0.5 h-5 w-5 shrink-0 justify-center p-0">{i + 1}</Badge>
              <span>{t(key)}</span>
            </li>
          {/each}
        </ol>
        <p class="flex items-start gap-2 border-t px-4 py-3 text-xs text-muted-foreground">
          <ShieldAlert class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
          {t('android.security')}
        </p>
      </section>
    </div>
  </div>
</div>
