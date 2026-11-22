<!-- 有问题欢迎联系 hot_kun@hotmail.com -->
<script lang="ts">
  import { TableCell, TableRow } from '$lib/components/ui/table'
  import { t } from '$lib/i18n/index.svelte'

  let { loading, colspan }: { loading: boolean; colspan: number } = $props()
</script>

<!-- 表格没行时的那一行。loading 与 empty 用同一格文案切换，
     而不是各渲染一个整表状态：加载完从「加载中」变「暂无数据」时
     表头不动，视觉上不会闪一下。
     插画用吉祥物：加载中 = 赶路，无数据 = 脑袋空白（装饰图，读屏用文案）。 -->
<TableRow>
  <TableCell {colspan} class="py-10 text-center text-muted-foreground">
    <div class="flex flex-col items-center gap-2">
      {#if loading}
        <img
          class="size-12 animate-[whale-bob_1.1s_ease-in-out_infinite] select-none"
          src="/brand/running.webp"
          srcset="/brand/running.webp 1x, /brand/running@2x.webp 2x"
          alt=""
          aria-hidden="true"
          draggable="false"
        />
      {:else}
        <img
          class="size-12 select-none opacity-80"
          src="/brand/blank.webp"
          srcset="/brand/blank.webp 1x, /brand/blank@2x.webp 2x"
          alt=""
          aria-hidden="true"
          draggable="false"
        />
      {/if}
      <span>{loading ? t('common.loading') : t('common.empty')}</span>
    </div>
  </TableCell>
</TableRow>

<style>
  @keyframes whale-bob {
    0%, 100% { transform: translateY(0); }
    50% { transform: translateY(-4px); }
  }
  @media (prefers-reduced-motion: reduce) {
    img { animation: none !important; }
  }
</style>
