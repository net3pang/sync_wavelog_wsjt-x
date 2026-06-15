<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import ConfigPanel from './lib/ConfigPanel.svelte'
  import LogPanel from './lib/LogPanel.svelte'
  import type { ConfigDTO, StationProfileDTO } from './vite-env'

  let cfg: ConfigDTO = {
    wavelog_url: '', api_key: '', station_id: '',
    wsjtx_log_path: '',
  }
  let profiles: StationProfileDTO[] = []
  let loadingProfiles = false
  let entries: { text: string; type: string }[] = []
  let syncing = false
  let saving = false

  onMount(async () => {
    try {
      cfg = await window.go.main.App.GetConfig()
    } catch (e) {
      addLog('加载配置失败: ' + e, 'error')
    }
    window.runtime.EventsOn('log', (msg: string) => addLog(msg, 'info'))
  })

  onDestroy(() => {
    window.runtime.EventsOff('log')
  })

  const VERSION = 'v1.0.0'

  function addLog(msg: string, type = 'info') {
    const ts = new Date().toLocaleTimeString('zh-CN', { hour12: false })
    entries = [...entries, { text: `[${ts}] ${msg}`, type }].slice(-100)
  }

  function clearLog() { entries = [] }

  async function refreshProfiles() {
    if (loadingProfiles) return
    loadingProfiles = true
    try {
      profiles = await window.go.main.App.GetStationProfiles()
      addLog(`加载到 ${profiles.length} 个台站`, 'success')
    } catch (e: any) {
      addLog('加载台站失败: ' + (e?.message || e), 'error')
    } finally { loadingProfiles = false }
  }

  async function saveConfig() {
    saving = true
    try {
      await window.go.main.App.SaveConfig(cfg)
      addLog('配置已保存', 'success')
    } catch (e: any) {
      addLog('保存失败: ' + (e?.message || e), 'error')
    } finally { saving = false }
  }

  async function syncDown() {
    syncing = true
    try {
      await window.go.main.App.SyncFromWavelog()
      addLog('拉取完成', 'success')
    } catch (e: any) {
      addLog('拉取失败: ' + (e?.message || e), 'error')
    } finally { syncing = false }
  }

  async function syncAll() {
    syncing = true
    try {
      await window.go.main.App.SyncAll()
      addLog('双向同步完成', 'success')
    } catch (e: any) {
      addLog('同步失败: ' + (e?.message || e), 'error')
    } finally { syncing = false }
  }
</script>

<div style="display:flex;flex-direction:column;height:100vh;overflow:hidden;padding:4px 8px;gap:2px;">
  <ConfigPanel {cfg} {profiles} {loadingProfiles} onRefreshProfiles={refreshProfiles} />

  <div style="display:flex;gap:8px;align-items:center;flex-shrink:0;">
    <button class="btn btn-secondary" on:click={syncDown} disabled={syncing}>
      {syncing ? '同步中...' : '拉取远程日志'}
    </button>
    <button class="btn btn-primary" on:click={syncAll} disabled={syncing}>
      {syncing ? '同步中...' : '开始双向同步'}
    </button>
    <button class="btn btn-secondary" on:click={saveConfig} disabled={saving}>
      {saving ? '保存中...' : '保存配置'}
    </button>
  </div>

  <div style="flex:1;min-height:0;display:flex;flex-direction:column;margin-bottom:28px;">
    <LogPanel {entries} onClear={clearLog} />
  </div>
</div>
<div style="position:fixed;bottom:4px;left:0;right:0;text-align:center;font-size:12px;color:#585b70;">
  © 2026 BH5HIE · GPL-3.0 · {VERSION}
</div>