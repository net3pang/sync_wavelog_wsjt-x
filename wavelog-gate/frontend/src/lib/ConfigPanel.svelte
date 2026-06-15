<script lang="ts">
  import type { ConfigDTO, StationProfileDTO } from '../vite-env'

  export let cfg: ConfigDTO
  export let profiles: StationProfileDTO[] = []
  export let loadingProfiles = false
  export let onRefreshProfiles: () => void = () => {}

  const browse = async () => {
    try {
      const result = await window.go.main.App.BrowseLogFile()
      if (result) {
        cfg.wsjtx_log_path = result
      }
    } catch (e) {
      console.error('browse failed', e)
    }
  }
</script>

<div class="config-group">
  <h3>配置参数</h3>

  <div class="field-row">
    <label for="url">Wavelog URL</label>
    <input id="url" type="text" bind:value={cfg.wavelog_url} placeholder="https://log.example.com/index.php" />
  </div>

  <div class="field-row">
    <label for="key">API Key</label>
    <input id="key" type="password" bind:value={cfg.api_key} placeholder="输入 API Key" />
  </div>

  <div class="field-row">
    <label for="sid">台站</label>
    <select id="sid" bind:value={cfg.station_id} disabled={loadingProfiles}>
      <option value="">-- 请选择台站 --</option>
      {#each profiles as p}
        <option value={p.station_id}>{p.station_profile_name}</option>
      {/each}
    </select>
    <button class="btn btn-secondary btn-small" on:click={onRefreshProfiles} disabled={loadingProfiles}>
      {loadingProfiles ? '加载中...' : '↻ 刷新'}
    </button>
  </div>

  <div class="field-row">
    <label for="path">本地日志</label>
    <input id="path" type="text" bind:value={cfg.wsjtx_log_path} placeholder="wsjtx_log.adi 路径" />
    <button class="btn btn-secondary btn-small" on:click={browse}>浏览</button>
  </div>

</div>
