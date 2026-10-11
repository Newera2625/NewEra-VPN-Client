package main

// Windows 系统代理设置（注册表 + WinINET 刷新）
// Linux 由 nodes.go 的 gsettings 实现

import (
	"fmt"
	"os/exec"

	"golang.org/x/sys/windows/registry"
)

// winSetProxy 写 HKCU Internet Settings 并刷新
func winSetProxy(ip string, port int) error {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return fmt.Errorf("打开注册表失败: %v", err)
	}
	defer k.Close()
	addr := fmt.Sprintf("%s:%d", ip, port)
	if err := k.SetDWordValue("ProxyEnable", 1); err != nil {
		return err
	}
	if err := k.SetStringValue("ProxyServer", addr); err != nil {
		return err
	}
	if err := k.SetStringValue("ProxyOverride", "<local>"); err != nil {
		return err
	}
	winRefreshProxy()
	return nil
}

func winClearProxy() {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return
	}
	defer k.Close()
	_ = k.SetDWordValue("ProxyEnable", 0)
	winRefreshProxy()
}

// winRefreshProxy 通知系统代理设置已变更
func winRefreshProxy() {
	if err := exec.Command("rundll32", "url.dll,InternetSetOptionW", "0", "39", "0").Run(); err != nil {
		_ = err
	}
}
