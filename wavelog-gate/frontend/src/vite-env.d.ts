/// <reference types="svelte" />
/// <reference types="vite/client" />

declare global {
  interface Window {
    go: {
      main: {
        App: {
          GetConfig(): Promise<ConfigDTO>
          SaveConfig(cfg: ConfigDTO): Promise<void>
          SyncFromWavelog(): Promise<void>
          SyncAll(): Promise<void>
          GetStationProfiles(): Promise<StationProfileDTO[]>
          BrowseLogFile(): Promise<string>
        }
      }
    }
    runtime: {
      EventsOn(event: string, callback: (...args: any[]) => void): void
      EventsOff(event: string): void
    }
  }
}

export interface ConfigDTO {
  wavelog_url: string
  api_key: string
  station_id: string
  wsjtx_log_path: string
}

export interface StationProfileDTO {
  station_id: string
  station_profile_name: string
}

export {}
