package main

// 自动更新：检查站点 /api/client/update -> GitHub Release 下载 -> 替换自身

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type UpdateInfo struct {
	Update  bool   `json:"update"`
	Version string `json:"version"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
		Size int64  `json:"size"`
	} `json:"assets"`
}

func checkUpdate() (*UpdateInfo, error) {
	url := fmt.Sprintf("%s/api/client/update?platform=%s&current=%s", SiteBase, UpdatePlatform, Version)
	resp, err := httpClient().Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var d struct {
		Code int        `json:"code"`
		Data UpdateInfo `json:"data"`
	}
	if err := json.Unmarshal(body, &d); err != nil || d.Code != 200 {
		return nil, fmt.Errorf("更新服务异常")
	}
	return &d.Data, nil
}

func apiCheckUpdate(w http.ResponseWriter, r *http.Request) {
	info, err := checkUpdate()
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 500, "message": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"code": 200, "data": info})
}

func apiDoUpdate(w http.ResponseWriter, r *http.Request) {
	if updateDone {
		writeJSON(w, map[string]interface{}{"code": 200, "message": "更新已开始，客户端即将重启"})
		return
	}
	info, err := checkUpdate()
	if err != nil || info == nil || !info.Update {
		writeJSON(w, map[string]interface{}{"code": 200, "message": "当前已是最新版本"})
		return
	}
	updating.Lock()
	defer updating.Unlock()
	if updateDone {
		writeJSON(w, map[string]interface{}{"code": 200, "message": "更新已开始"})
		return
	}
	// 选择匹配当前平台的资产
	var asset *struct {
		Name string `json:"name"`
		URL  string `json:"url"`
		Size int64  `json:"size"`
	}
	for i := range info.Assets {
		n := strings.ToLower(info.Assets[i].Name)
		if strings.Contains(n, UpdatePlatform) || strings.Contains(n, platformTag()) {
			asset = &info.Assets[i]
			break
		}
	}
	if asset == nil {
		writeJSON(w, map[string]interface{}{"code": 200, "message": "没有匹配当前平台的更新文件"})
		return
	}
	writeJSON(w, map[string]interface{}{"code": 200, "message": "开始下载更新 " + info.Version})
	go applyUpdate(asset.URL, asset.Name, info.Version)
}

// applyUpdate 下载 -> 替换 -> 重启
func applyUpdate(url, fname, version string) {
	updateDone = true
	tmp := filepath.Join(os.TempDir(), "newera-vpn-"+version+"-"+fname)
	// 下载
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := httpClient().Do(req)
	if err != nil {
		updateDone = false
		return
	}
	defer resp.Body.Close()
	f, err := os.Create(tmp)
	if err != nil {
		updateDone = false
		return
	}
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil || fileSize(tmp) == 0 {
		os.Remove(tmp)
		updateDone = false
		return
	}

	exe, err := os.Executable()
	if err != nil {
		updateDone = false
		return
	}
	exeReal, _ := filepath.EvalSymlinks(exe)
	dir := filepath.Dir(exeReal)
	if runtime.GOOS == "windows" {
		// 运行中的exe不能自覆盖：用 .cmd 延迟替换
		old := exeReal + ".old"
		os.Rename(exeReal, old)
		script := "@echo off\r\n" +
			"timeout /t 1 /nobreak >nul\r\n" +
			"del /f /q \"" + old + "\" >nul 2>&1\r\n" +
			"move /y \"" + tmp + "\" \"" + exeReal + "\" >nul 2>&1\r\n" +
			"start \"\" \"" + exeReal + "\"\r\n"
		bat := filepath.Join(dir, "_update.cmd")
		os.WriteFile(bat, []byte(script), 0o755)
		exec.Command("cmd", "/c", bat).Start()
	} else {
		os.Rename(tmp, exeReal)
		os.Chmod(exeReal, 0o755)
		// 重启（detach）
		cmd := exec.Command(exeReal)
		cmd.Start()
	}
	time.Sleep(500 * time.Millisecond)
	os.Exit(0)
}

func fileSize(p string) int64 {
	st, err := os.Stat(p)
	if err != nil {
		return 0
	}
	return st.Size()
}
