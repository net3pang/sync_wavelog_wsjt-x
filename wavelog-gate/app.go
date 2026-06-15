package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"wavelog-gate/internal/adif"
	"wavelog-gate/internal/config"
	"wavelog-gate/internal/wavelog"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx    context.Context
	cfg    *config.Config
	client *wavelog.Client
}

type ConfigDTO struct {
	WavelogURL string `json:"wavelog_url"`
	APIKey     string `json:"api_key"`
	StationID  string `json:"station_id"`
	LogPath    string `json:"wsjtx_log_path"`
}

func NewApp() *App {
	cfg := config.Load()
	return &App{
		cfg: cfg,
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Shutdown(_ context.Context) {
}

// ---- Exported to frontend ----

func (a *App) GetConfig() ConfigDTO {
	return ConfigDTO{
		WavelogURL: a.cfg.WavelogURL,
		APIKey:     a.cfg.APIKey,
		StationID:  a.cfg.StationID,
		LogPath:    a.cfg.LogPath,
	}
}

func (a *App) BrowseLogFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 wsjtx_log.adi",
		Filters: []runtime.FileFilter{
			{DisplayName: "ADIF", Pattern: "*.adi"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
}

func (a *App) SaveConfig(dto ConfigDTO) error {
	a.cfg.WavelogURL = dto.WavelogURL
	a.cfg.APIKey = dto.APIKey
	a.cfg.StationID = dto.StationID
	a.cfg.LogPath = dto.LogPath
	if err := config.Save(a.cfg); err != nil {
		return err
	}
	a.log("配置已保存")
	a.refreshClient()
	return nil
}

func (a *App) GetStationProfiles() ([]wavelog.StationProfile, error) {
	a.refreshClient()
	if a.client == nil {
		return nil, fmt.Errorf("请先配置 Wavelog URL 和 API Key")
	}
	return a.client.GetStationProfiles()
}

func (a *App) SyncFromWavelog() error {
	a.refreshClient()
	if a.client == nil || a.cfg.WavelogURL == "" || a.cfg.APIKey == "" {
		return fmt.Errorf("请先配置 Wavelog URL 和 API Key")
	}
	logPath := a.cfg.LogPath
	if logPath == "" {
		return fmt.Errorf("请先设置本地日志路径")
	}

	a.log("正在拉取远程日志...")
	contacts, err := a.client.GetContacts()
	if err != nil {
		return fmt.Errorf("拉取失败: %w", err)
	}
	if len(contacts) == 0 {
		a.log("远程日志为空")
		return nil
	}
	a.log(fmt.Sprintf("拉取到 %d 条记录", len(contacts)))

	localKeys := make(map[[3]string]struct{})
	needHead := false
	if data, err := os.ReadFile(logPath); err == nil {
		localKeys = adif.ParseLocalKeys(string(data))
	} else {
		needHead = true
	}

	var newADIFs []string
	for _, c := range contacts {
		call := toStr(c["CALL"])
		qd := toStr(c["QSO_DATE"])
		to := toStr(c["TIME_ON"])
		if call == "" || qd == "" {
			continue
		}
		key := [3]string{call, qd, to}
		if _, exists := localKeys[key]; exists {
			continue
		}
		newADIFs = append(newADIFs, adif.MakeRecord(map[string]string{
			"CALL":        call,
			"QSO_DATE":    qd,
			"TIME_ON":     to,
			"BAND":        toStr(c["BAND"]),
			"MODE":        toStr(c["MODE"]),
			"RST_SENT":    toStr(c["RST_SENT"]),
			"RST_RCVD":    toStr(c["RST_RCVD"]),
			"GRIDSQUARE": toStr(c["GRIDSQUARE"]),
		}))
	}

	if len(newADIFs) == 0 {
		a.log("本地已是最新")
		return nil
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()
	if needHead {
		if _, err := f.WriteString(adif.Header()); err != nil {
			return err
		}
	}
	for _, rec := range newADIFs {
		if _, err := f.WriteString(rec + "\n"); err != nil {
			return err
		}
	}
	a.log(fmt.Sprintf("新增 %d 条到 %s", len(newADIFs), filepath.Base(logPath)))
	return nil
}

func (a *App) SyncAll() error {
	a.refreshClient()
	if a.client == nil || a.cfg.WavelogURL == "" || a.cfg.APIKey == "" {
		return fmt.Errorf("请先配置 Wavelog URL 和 API Key")
	}
	logPath := a.cfg.LogPath
	if logPath == "" {
		return fmt.Errorf("请先设置本地日志路径")
	}

	// 1. Fetch remote contacts
	a.log("正在拉取远程日志...")
	contacts, err := a.client.GetContacts()
	if err != nil {
		return fmt.Errorf("拉取失败: %w", err)
	}
	a.log(fmt.Sprintf("远程共 %d 条记录", len(contacts)))

	// Build remote keys set
	remoteKeys := make(map[[3]string]struct{})
	for _, c := range contacts {
		call := toStr(c["CALL"])
		qd := toStr(c["QSO_DATE"])
		to := toStr(c["TIME_ON"])
		if call != "" && qd != "" {
			remoteKeys[[3]string{call, qd, to}] = struct{}{}
		}
	}

	// 2. Read local file
	var localData string
	needHead := false
	if data, err := os.ReadFile(logPath); err == nil {
		localData = string(data)
	} else {
		needHead = true
	}

	localRecords := adif.ParseLocalRecords(localData)
	localKeys := make(map[[3]string]struct{})
	for k := range localRecords {
		localKeys[k] = struct{}{}
	}
	a.log(fmt.Sprintf("本地共 %d 条记录", len(localRecords)))

	// 3. Upload local records not in remote
	var uploaded int
	for key, adifStr := range localRecords {
		if _, exists := remoteKeys[key]; exists {
			continue
		}
		a.log(fmt.Sprintf("上传: %s %s %s", key[0], key[1], key[2]))
		if err := a.client.ForwardQSO(adifStr); err != nil {
			a.log(fmt.Sprintf("上传失败 %s: %v", key[0], err))
			continue
		}
		uploaded++
	}
	if uploaded > 0 {
		a.log(fmt.Sprintf("上传完成: %d 条", uploaded))
	} else {
		a.log("本地无新增记录需要上传")
	}

	// 4. Download remote contacts not in local
	var downloaded []string
	for _, c := range contacts {
		call := toStr(c["CALL"])
		qd := toStr(c["QSO_DATE"])
		to := toStr(c["TIME_ON"])
		if call == "" || qd == "" {
			continue
		}
		key := [3]string{call, qd, to}
		if _, exists := localKeys[key]; exists {
			continue
		}
		downloaded = append(downloaded, adif.MakeRecord(map[string]string{
			"CALL":        call,
			"QSO_DATE":    qd,
			"TIME_ON":     to,
			"BAND":        toStr(c["BAND"]),
			"MODE":        toStr(c["MODE"]),
			"RST_SENT":    toStr(c["RST_SENT"]),
			"RST_RCVD":    toStr(c["RST_RCVD"]),
			"GRIDSQUARE": toStr(c["GRIDSQUARE"]),
		}))
	}

	if len(downloaded) > 0 {
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("打开文件失败: %w", err)
		}
		defer f.Close()
		if needHead {
			if _, err := f.WriteString(adif.Header()); err != nil {
				return err
			}
		}
		for _, rec := range downloaded {
			if _, err := f.WriteString(rec + "\n"); err != nil {
				return err
			}
		}
		a.log(fmt.Sprintf("下载写入 %d 条到 %s", len(downloaded), filepath.Base(logPath)))
	} else {
		a.log("远程无新增记录需要下载")
	}

	return nil
}

// ---- internal ----

func (a *App) log(msg string) {
	runtime.EventsEmit(a.ctx, "log", msg)
}

func (a *App) refreshClient() {
	if a.cfg.WavelogURL != "" && a.cfg.APIKey != "" {
		a.client = wavelog.New(a.cfg.WavelogURL, a.cfg.APIKey, a.cfg.StationID)
	}
}

func toStr(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
