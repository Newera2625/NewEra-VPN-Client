package main

// Linux 平台 stub（系统代理走 gsettings，见 nodes.go）

func winSetProxy(ip string, port int) error {
	return setLinuxProxy(ip, port)
}

func winClearProxy() {
	clearLinuxProxy()
}
