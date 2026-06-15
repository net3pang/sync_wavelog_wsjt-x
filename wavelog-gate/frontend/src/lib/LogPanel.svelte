<script lang="ts">
  import type { LogEntry } from './stores'

  export let entries: LogEntry[] = []
  export let onClear: () => void = () => {}

  let logEl: HTMLDivElement

  $: if (logEl) {
    logEl.scrollTop = logEl.scrollHeight
  }
</script>

<div style="display:flex;flex-direction:column;flex:1;min-height:0;">
  <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:2px;flex-shrink:0;">
    <h3 style="font-size:14px;color:#a6adc8;text-transform:uppercase;letter-spacing:0.5px;">运行日志</h3>
    <button class="btn btn-secondary btn-small" on:click={onClear}>清空</button>
  </div>
  <div bind:this={logEl}
    style="flex:1;min-height:0;overflow-y:auto;background:#11111b;border:1px solid #313244;border-radius:8px;padding:8px;margin-bottom:8px;font-family:'Cascadia Code','Fira Code','JetBrains Mono',Consolas,monospace;font-size:12px;line-height:1.5;word-break:break-all;color:#a6adc8;">
    {#each entries as entry}
      <span class={'log-entry ' + entry.type}>{entry.text}</span><br />
    {:else}
      <span style="color:#585b70;">暂无日志</span>
    {/each}
  </div>
</div>
