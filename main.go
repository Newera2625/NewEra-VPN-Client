package main

// 新启年VPN客户端 - 多平台（Windows / Linux / Android-Termux）
// 功能：SSO登录、节点列表（本地延迟测试）、一键连接（系统代理）、自动更新

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	Version       = "1.0.0"
	LocalAddr     = "127.0.0.1:12901"
	SiteBase      = "https://vpn.newera2625.top"
	SSOBase       = "https://sso.newera2625.top"
	ClientID      = "vpn-site"
	LocalCallback = "http://127.0.0.1:12901/callback"
)

var (
	UpdatePlatform = "windows" // windows / linux / linux-arm
	configDir      string
	tokenFile      string
	stateFile      string
	updating       sync.Mutex
	updateDone     = false
)

type UserInfo struct {
	UID      int    `json:"uid"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Email    string `json:"email"`
}

type NodeInfo struct {
	ID          string `json:"id"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
	Protocol    string `json:"protocol"`
	Latency     int    `json:"latency"`
	IP          string `json:"-"`
	Port        int    `json:"-"`
}

type ConnState struct {
	Connected bool   `json:"connected"`
	NodeID    string `json:"node_id"`
	IP        string `json:"ip"`
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`
	Country   string `json:"country"`
	StartedAt int64  `json:"started_at"`
}

func platformTag() string {
	if runtime.GOOS == "windows" {
		return "windows"
	}
	if runtime.GOOS == "linux" && runtime.GOARCH == "arm64" {
		return "linux-arm"
	}
	return "linux"
}

func init() {
	UpdatePlatform = platformTag()
	base, err := os.UserHomeDir()
	if err != nil {
		base = "."
	}
	configDir = filepath.Join(base, ".newera-vpn")
	os.MkdirAll(configDir, 0o755)
	tokenFile = filepath.Join(configDir, "token.txt")
	stateFile = filepath.Join(configDir, "state.json")
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 12 * time.Second}
}

func readToken() string {
	b, err := os.ReadFile(tokenFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func writeToken(tok string) error {
	return os.WriteFile(tokenFile, []byte(tok), 0o600)
}

func readState() *ConnState {
	s := &ConnState{}
	b, err := os.ReadFile(stateFile)
	if err != nil {
		return s
	}
	json.Unmarshal(b, s)
	return s
}

func saveState(s *ConnState) error {
	b, _ := json.Marshal(s)
	return os.WriteFile(stateFile, b, 0o600)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	go func() {
		defer func() { recover() }()
		cmd.Run()
	}()
}

// ---------------- SSO ----------------

func getUser(token string) (*UserInfo, error) {
	if token == "" {
		return nil, fmt.Errorf("no token")
	}
	req, _ := http.NewRequest("GET", SSOBase+"/api/status", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var d struct {
		Code int `json:"code"`
		Data struct {
			User *UserInfo `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &d); err != nil || d.Code != 200 || d.Data.User == nil {
		return nil, fmt.Errorf("sso status failed")
	}
	return d.Data.User, nil
}

func apiStatus(w http.ResponseWriter, r *http.Request) {
	tok := readToken()
	u, err := getUser(tok)
	if tok == "" || err != nil {
		writeJSON(w, map[string]interface{}{
			"logged_in": false,
			"connected": readState().Connected,
			"version":   Version,
		})
		return
	}
	av := u.Avatar
	if av != "" && !strings.HasPrefix(av, "http") {
		av = SSOBase + (func() string {
			if strings.HasPrefix(av, "/") {
				return av
			}
			return "/" + av
		}())
	}
	writeJSON(w, map[string]interface{}{
		"logged_in": true,
		"user":      u,
		"avatar":    av,
		"connected": readState().Connected,
		"version":   Version,
	})
}

func apiLogin(w http.ResponseWriter, r *http.Request) {
	authURL := fmt.Sprintf("%s/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code&state=client",
		SSOBase, ClientID, LocalCallback)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "缺少授权码", http.StatusBadRequest)
		return
	}
	tokURL := fmt.Sprintf("%s/oauth/token?grant_type=authorization_code&code=%s&client_id=%s&redirect_uri=%s",
		SSOBase, code, ClientID, LocalCallback)
	resp, err := httpClient().Get(tokURL)
	if err != nil {
		http.Error(w, "换取Token失败: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var td struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	json.Unmarshal(body, &td)
	if td.AccessToken == "" {
		http.Error(w, "登录失败: "+td.Error, http.StatusUnauthorized)
		return
	}
	writeToken(td.AccessToken)
	writeHTML(w, `<html><head><meta charset="utf-8"><title>登录成功</title></head><body style="display:flex;align-items:center;justify-content:center;height:100vh;font-family:sans-serif;background:#f5f7fa"><div style="text-align:center"><div style="font-size:56px;color:#18a058">✓</div><h2>登录成功</h2><p style="color:#888">新启年VPN客户端已登录，可以关闭本页面</p></div></body></html>`)
}

func apiLogout(w http.ResponseWriter, r *http.Request) {
	os.Remove(tokenFile)
	writeJSON(w, map[string]interface{}{"code": 200})
}

// ---------------- 主入口 ----------------

func main() {
	log.SetFlags(log.LstdFlags)
	fmt.Printf("新启年VPN客户端 v%s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)
	fmt.Printf("控制台地址: http://%s\n", LocalAddr)

	// 启动本地控制服务
	go startServer()
	time.Sleep(400 * time.Millisecond)

	// 自动打开浏览器控制页
	openBrowser("http://" + LocalAddr)

	// 启动时后台检查更新（不打扰）
	go func() {
		time.Sleep(8 * time.Second)
		info, err := checkUpdate()
		if err == nil && info != nil && info.Update {
			fmt.Printf("发现新版本 %s，可到控制页点击更新\n", info.Version)
		}
	}()

	// 退出时若系统代理是我们设置的则清理
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Println("退出中，清理系统代理...")
	clearSystemProxy()
}
