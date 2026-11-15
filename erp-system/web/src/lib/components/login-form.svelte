<!-- 有问题欢迎联系 hot_kun@hotmail.com -->
<script lang="ts">
  import ArrowRight from '@lucide/svelte/icons/arrow-right'
  import Boxes from '@lucide/svelte/icons/boxes'
  import CircleAlert from '@lucide/svelte/icons/circle-alert'
  import ClipboardCheck from '@lucide/svelte/icons/clipboard-check'
  import Eye from '@lucide/svelte/icons/eye'
  import EyeOff from '@lucide/svelte/icons/eye-off'
  import Globe from '@lucide/svelte/icons/globe'
  import KeyRound from '@lucide/svelte/icons/key-round'
  import LockKeyhole from '@lucide/svelte/icons/lock-keyhole'
  import ShieldCheck from '@lucide/svelte/icons/shield-check'
  import ShoppingCart from '@lucide/svelte/icons/shopping-cart'
  import UserRound from '@lucide/svelte/icons/user-round'

  import { ApiError } from '$lib/api/client'
  import { Input } from '$lib/components/ui/input'
  import { getLocale, locales, setLocale, t, tKey, type MessageKey } from '$lib/i18n/index.svelte'
  import { login } from '$lib/stores/session.svelte'

  let username = $state('')
  let password = $state('')
  let showPassword = $state(false)
  let submitting = $state(false)
  let errorKey = $state<string | null>(null)

  // 语言切换按钮显示「切过去之后」的语言，而不是当前语言：
  // 按钮上写当前语言的话，用户会以为点一下是「切回来」。
  const nextLocale = $derived(getLocale() === 'zh-CN' ? 'en-US' : 'zh-CN')

  // 左侧三条能力说明。文案键而不是文案：切换语言时随 t() 一起变。
  const features: Array<{
    icon: typeof ShoppingCart
    title: MessageKey
    desc: MessageKey
    accent: string
    surface: string
  }> = [
    {
      icon: ShoppingCart,
      title: 'login.feature.purchase.title',
      desc: 'login.feature.purchase.desc',
      accent: 'text-emerald-300',
      surface: 'border-emerald-300/20 bg-emerald-400/[0.07]',
    },
    {
      icon: Boxes,
      title: 'login.feature.stock.title',
      desc: 'login.feature.stock.desc',
      accent: 'text-teal-300',
      surface: 'border-teal-300/20 bg-teal-400/[0.07]',
    },
    {
      icon: ClipboardCheck,
      title: 'login.feature.audit.title',
      desc: 'login.feature.audit.desc',
      accent: 'text-cyan-300',
      surface: 'border-cyan-300/20 bg-cyan-400/[0.07]',
    },
  ]

  async function submit(event: SubmitEvent) {
    event.preventDefault()
    submitting = true
    errorKey = null
    try {
      await login(username, password)
    } catch (err) {
      // 只存后端的 i18n 键，组件里不出现任何具体文案
      errorKey = err instanceof ApiError ? err.key : 'error.internal'
    } finally {
      submitting = false
    }
  }
</script>

<main class="relative flex min-h-svh overflow-hidden bg-[#061014]">
  <h1 class="sr-only">{t('app.name')}</h1>

  <!-- 背景：网格 + 低饱和光晕。纯 CSS，不引入图片资源。 -->
  <div class="pointer-events-none absolute inset-0" aria-hidden="true">
    <div class="absolute -left-32 top-[-20%] size-[560px] rounded-full bg-emerald-500/20 blur-[140px]"></div>
    <div class="absolute -bottom-40 right-[-8%] size-[480px] rounded-full bg-teal-500/10 blur-[120px]"></div>
    <div class="login-grid absolute inset-0"></div>
    <div class="absolute -right-44 top-[16%] size-[560px] rounded-full border border-white/[0.06]"></div>
    <div class="absolute -right-24 top-[32%] size-[340px] rounded-full border border-white/[0.04]"></div>
  </div>

  <!-- 左侧：品牌与业务叙事。窄屏隐藏，避免把表单挤到首屏之外。 -->
  <section class="relative z-10 hidden w-[55%] flex-col justify-between px-12 py-12 lg:flex xl:px-16 xl:py-14">
    <div class="login-rise flex items-center gap-3">
      <span
        class="flex size-11 items-center justify-center rounded-2xl bg-gradient-to-br from-emerald-400 to-teal-600 shadow-lg shadow-emerald-500/25"
      >
        <Boxes class="size-5 text-white" aria-hidden="true" />
      </span>
      <div>
        <p class="text-sm font-semibold tracking-wide text-white">{t('app.name')}</p>
        <p class="mt-0.5 text-[10px] font-medium tracking-[0.24em] text-white/40">
          {t('login.eyebrow')}
        </p>
      </div>
    </div>

    <div class="login-rise login-delay-1 max-w-xl">
      <div
        class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/[0.04] px-3.5 py-1.5 text-[11px] font-medium tracking-wide text-white/60 backdrop-blur"
      >
        <span class="relative flex size-1.5" aria-hidden="true">
          <span class="absolute inline-flex size-full rounded-full bg-emerald-400/60 motion-safe:animate-ping"></span>
          <span class="relative inline-flex size-1.5 rounded-full bg-emerald-400"></span>
        </span>
        {t('login.heroBadge')}
      </div>

      <h2 class="mt-7 text-[2.35rem] font-semibold leading-[1.16] tracking-[-0.035em] text-white xl:text-[3rem]">
        {t('login.heroTitle')}
        <span class="block bg-linear-to-r from-emerald-300 via-teal-200 to-cyan-300 bg-clip-text text-transparent">
          {t('login.heroTitleAccent')}
        </span>
      </h2>

      <p class="mt-5 max-w-[26rem] text-[15px] leading-7 text-white/55">
        <span class="block">{t('login.heroSubtitle')}</span>
        <span class="mt-1 block">{t('login.heroSubtitleSecond')}</span>
      </p>

      <ul class="mt-10 space-y-5">
        {#each features as item (item.title)}
          {@const Icon = item.icon}
          <li class="flex gap-4">
            <div class="flex size-10 shrink-0 items-center justify-center rounded-xl border {item.surface} {item.accent}">
              <Icon class="size-[18px]" aria-hidden="true" />
            </div>
            <div>
              <p class="text-sm font-medium text-white/85">{t(item.title)}</p>
              <p class="mt-1 text-[13px] leading-5 text-white/50">{t(item.desc)}</p>
            </div>
          </li>
        {/each}
      </ul>
    </div>

    <div class="login-rise login-delay-2 flex items-center gap-4">
      <div class="h-px w-10 bg-white/15"></div>
      <p class="text-[11px] font-medium tracking-[0.16em] text-white/35">{t('login.heroFoot')}</p>
    </div>
  </section>

  <!-- 右侧：登录表单。浅色面板与左侧深色形成对比，保证表单可读性。 -->
  <section
    class="relative flex min-h-svh min-w-0 flex-1 items-center justify-center bg-[#f5f7f6] px-5 py-10 sm:px-8 lg:rounded-l-[2.5rem] lg:px-10 xl:px-16"
  >
    <div class="login-panel-glow pointer-events-none absolute inset-0 lg:rounded-l-[2.5rem]" aria-hidden="true"></div>

    <div class="absolute right-5 top-5 z-20 sm:right-7 sm:top-6">
      <button
        type="button"
        onclick={() => setLocale(nextLocale)}
        aria-label={t('login.switchLocale')}
        class="group inline-flex items-center gap-1.5 rounded-full border border-slate-200/80 bg-white/80 px-3 py-1.5 text-[11px] font-medium text-slate-500 shadow-sm backdrop-blur transition-colors hover:border-slate-300 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/30"
      >
        <Globe class="size-3.5 text-slate-400 transition-colors group-hover:text-emerald-600" aria-hidden="true" />
        {locales[nextLocale]}
      </button>
    </div>

    <div class="relative z-10 w-full max-w-[400px]">
      <!-- 移动端：左侧叙事隐藏后，用一行品牌信息保留辨识度。 -->
      <div class="mb-8 flex items-center gap-3 lg:hidden">
        <span
          class="flex size-10 items-center justify-center rounded-xl bg-gradient-to-br from-emerald-400 to-teal-600 shadow-md shadow-emerald-500/20"
        >
          <Boxes class="size-[18px] text-white" aria-hidden="true" />
        </span>
        <div class="min-w-0">
          <p class="truncate text-sm font-semibold text-slate-900">{t('app.name')}</p>
          <p class="mt-0.5 truncate text-[11px] text-slate-500">{t('app.description')}</p>
        </div>
      </div>

      <div
        class="login-rise login-delay-1 relative overflow-hidden rounded-[1.75rem] border border-slate-200/70 bg-white p-6 shadow-[0_30px_80px_-35px_rgba(6,16,20,0.35)] sm:p-8"
      >
        <div class="absolute inset-x-8 top-0 h-px bg-linear-to-r from-transparent via-emerald-500/40 to-transparent"></div>

        <p class="text-[10px] font-semibold tracking-[0.22em] text-emerald-600">{t('login.eyebrow')}</p>
        <h2 class="mt-2 text-[1.7rem] font-semibold tracking-[-0.03em] text-slate-900">
          {t('login.welcome')}
        </h2>
        <p class="mt-1.5 text-[13px] text-slate-500">{t('login.welcomeHint')}</p>

        <form class="mt-6 space-y-4" onsubmit={submit}>
          <div class="space-y-1.5">
            <label for="login-username" class="text-[13px] font-medium text-slate-700">
              {t('login.username')}
            </label>
            <div class="relative">
              <UserRound
                class="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-slate-400"
                aria-hidden="true"
              />
              <!-- svelte-ignore a11y_autofocus -->
              <Input
                id="login-username"
                class="h-11 rounded-xl border-slate-200 bg-slate-50/70 pl-10 pr-3 text-sm text-slate-900 shadow-none transition-colors placeholder:text-slate-400 focus-visible:border-emerald-400 focus-visible:bg-white focus-visible:ring-4 focus-visible:ring-emerald-500/10"
                bind:value={username}
                autocomplete="username"
                autofocus
                required
              />
            </div>
          </div>

          <div class="space-y-1.5">
            <label for="login-password" class="text-[13px] font-medium text-slate-700">
              {t('login.password')}
            </label>
            <div class="relative">
              <LockKeyhole
                class="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-slate-400"
                aria-hidden="true"
              />
              <Input
                id="login-password"
                type={showPassword ? 'text' : 'password'}
                class="h-11 rounded-xl border-slate-200 bg-slate-50/70 pl-10 pr-11 text-sm text-slate-900 shadow-none transition-colors placeholder:text-slate-400 focus-visible:border-emerald-400 focus-visible:bg-white focus-visible:ring-4 focus-visible:ring-emerald-500/10"
                bind:value={password}
                autocomplete="current-password"
                required
              />
              <button
                type="button"
                aria-label={showPassword ? t('login.passwordHide') : t('login.passwordShow')}
                aria-pressed={showPassword}
                aria-controls="login-password"
                onclick={() => (showPassword = !showPassword)}
                class="absolute right-1.5 top-1/2 -translate-y-1/2 rounded-lg p-2 text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/30"
              >
                {#if showPassword}
                  <EyeOff class="size-4" aria-hidden="true" />
                {:else}
                  <Eye class="size-4" aria-hidden="true" />
                {/if}
              </button>
            </div>
          </div>

          <!-- 用 aria-live 而不是弹窗：登录失败是预期内的结果，不该打断用户，
               但读屏软件要能念出来。空态放一句辅助说明，避免错误出现时布局跳动。 -->
          <div id="login-feedback" class="min-h-[44px]" aria-live="polite">
            {#if errorKey}
              <div class="flex items-start gap-2 rounded-xl border border-red-200/80 bg-red-50 px-3 py-2.5 text-[13px] leading-5 text-red-700">
                <CircleAlert class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
                <span>{tKey(errorKey)}</span>
              </div>
            {:else}
              <p class="flex items-center gap-1.5 px-1 pt-1 text-[11px] text-slate-500">
                <ShieldCheck class="size-3.5" aria-hidden="true" />
                {t('login.helper')}
              </p>
            {/if}
          </div>

          <button
            type="submit"
            disabled={submitting}
            class="group flex h-11 w-full items-center justify-center gap-2 rounded-xl bg-gradient-to-r from-emerald-600 to-teal-600 px-4 text-sm font-medium text-white shadow-lg shadow-emerald-600/20 transition-all hover:from-emerald-500 hover:to-teal-500 hover:shadow-emerald-600/30 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-emerald-500/25 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {#if submitting}
              <span
                class="size-4 animate-spin rounded-full border-2 border-white/30 border-t-white"
                aria-hidden="true"
              ></span>
              {t('login.submitting')}
            {:else}
              {t('login.submit')}
              <ArrowRight
                class="size-4 transition-transform group-hover:translate-x-0.5"
                aria-hidden="true"
              />
            {/if}
          </button>
        </form>

        <div class="mt-6 flex gap-2.5 rounded-xl bg-slate-50 px-3.5 py-3 text-[11px] leading-5 text-slate-500 ring-1 ring-slate-100">
          <KeyRound class="mt-0.5 size-3.5 shrink-0 text-slate-400" aria-hidden="true" />
          <p>{t('login.hint')}</p>
        </div>
      </div>

      <p class="mt-8 text-center text-[11px] tracking-wide text-slate-500">{t('login.footer')}</p>
    </div>
  </section>
</main>

<style>
  /* 网格用 CSS 画，避免为一张背景图引入额外资源；透明度压得很低，只做质感。 */
  .login-grid {
    background-image:
      linear-gradient(to right, rgba(255, 255, 255, 0.045) 1px, transparent 1px),
      linear-gradient(to bottom, rgba(255, 255, 255, 0.045) 1px, transparent 1px);
    background-size: 72px 72px;
    -webkit-mask-image: radial-gradient(ellipse 75% 65% at 42% 42%, #000 20%, transparent 78%);
    mask-image: radial-gradient(ellipse 75% 65% at 42% 42%, #000 20%, transparent 78%);
  }

  .login-panel-glow {
    background: radial-gradient(60% 50% at 100% 0%, rgba(16, 185, 129, 0.08), transparent 70%);
  }

  .login-rise {
    animation: login-rise 0.65s cubic-bezier(0.2, 0.65, 0.25, 1) both;
  }

  .login-delay-1 {
    animation-delay: 0.08s;
  }

  .login-delay-2 {
    animation-delay: 0.16s;
  }

  @keyframes login-rise {
    from {
      opacity: 0;
      transform: translateY(14px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .login-rise {
      animation: none;
    }
  }
</style>
