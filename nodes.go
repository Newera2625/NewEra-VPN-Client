package main

// 节点拉取 + 系统代理连接

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sort"
	"time"
)

func fetchNodes() ([]NodeInfo, error) {
	resp, err := httpClient().Get(SiteBase + "/api/nodes")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var d struct {
		Code int `json:"code"`
		Data struct {
			Nodes []NodeInfo `json:"nodes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &d); err != nil || d.Code != 200 {
		return nil, fmt.Errorf("节点服务异常")
	}
	return d.Data.Nodes, nil
}

func apiNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := fetchNodes()
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 500, "message": err.Error()})
		return
	}
	sort.Slice(nodes, func(a, b int) bool { return nodes[a].Latency < nodes[b].Latency })
	writeJSON(w, map[string]interface{}{"code": 200, "data": map[string]interface{}{
		"nodes": nodes,
		"now":   time.Now().Unix(),
	}})
}

// ---------------- 连接（系统代理） ----------------

func apiConnect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.ID == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "message": "缺少节点ID"})
		return
	}
	tok := readToken()
	if tok == "" {
		writeJSON(w, map[string]interface{}{"code": 401, "message": "请先登录"})
		return
	}
	// 向站点请求该节点代理地址（校验登录 + 节点可见）
	req2, _ := http.NewRequest("GET", SiteBase+"/api/nodes/"+req.ID+"/proxy", nil)
	req2.AddCookie(&http.Cookie{Name: "auth_token", Value: tok})
	resp, err := httpClient().Do(req2)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 502, "message": "获取代理信息失败"})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var d struct {
		Code int `json:"code"`
		Data struct {
			IP       string `json:"ip"`
			Port     int    `json:"port"`
			Protocol string `json:"protocol"`
			Country  string `json:"country"`
		} `json:"data"`
	}
	json.Unmarshal(body, &d)
	if d.Code != 200 || d.Data.IP == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "message": "节点不可用或未登录"})
		return
	}
	if err := setSystemProxy(d.Data.IP, d.Data.Port); err != nil {
		writeJSON(w, map[string]interface{}{"code": 500, "message": "设置系统代理失败: " + err.Error()})
		return
	}
	state := &ConnState{
		Connected: true,
		NodeID:    req.ID,
		IP:        d.Data.IP,
		Port:      d.Data.Port,
		Protocol:  d.Data.Protocol,
		Country:   d.Data.Country,
		StartedAt: time.Now().Unix(),
	}
	saveState(state)
	writeJSON(w, map[string]interface{}{"code": 200, "data": state})
}

func apiDisconnect(w http.ResponseWriter, r *http.Request) {
	clearSystemProxy()
	saveState(&ConnState{})
	writeJSON(w, map[string]interface{}{"code": 200})
}

// ---------------- 系统代理实现 ----------------

func setSystemProxy(ip string, port int) error {
	if runtime.GOOS == "windows" {
		return winSetProxy(ip, port)
	}
	return setLinuxProxy(ip, port)
}

func clearSystemProxy() {
	if runtime.GOOS == "windows" {
		winClearProxy()
	} else {
		clearLinuxProxy()
	}
}

func setLinuxProxy(ip string, port int) error {
	runCmd("gsettings", "set", "org.gnome.system.proxy", "mode", "manual")
	runCmd("gsettings", "set", "org.gnome.system.proxy.http", "host", ip)
	runCmd("gsettings", "set", "org.gnome.system.proxy.http", "port", fmt.Sprintf("%d", port))
	runCmd("gsettings", "set", "org.gnome.system.proxy.https", "host", ip)
	runCmd("gsettings", "set", "org.gnome.system.proxy.https", "port", fmt.Sprintf("%d", port))
	runCmd("gsettings", "set", "org.gnome.system.proxy.socks", "host", ip)
	runCmd("gsettings", "set", "org.gnome.system.proxy.socks", "port", fmt.Sprintf("%d", port))
	return nil
}

func clearLinuxProxy() {
	runCmd("gsettings", "set", "org.gnome.system.proxy", "mode", "none")
}
