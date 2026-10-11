package main

// 本地控制服务 + 内嵌前端页面

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
)

//go:embed web/index.html web/style.css web/app.js
var webFS embed.FS

func newExecCmd(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func runCmd(name string, args ...string) {
	cmd := exec.Command(name, args...)
	_ = cmd.Run()
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func writeHTML(w http.ResponseWriter, html string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

func startServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			b, _ := webFS.ReadFile("web/index.html")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(b)
		case "/style.css":
			b, _ := webFS.ReadFile("web/style.css")
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
			w.Write(b)
		case "/app.js":
			b, _ := webFS.ReadFile("web/app.js")
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			w.Write(b)
		case "/api/status":
			apiStatus(w, r)
		case "/api/login":
			apiLogin(w, r)
		case "/callback":
			callbackHandler(w, r)
		case "/api/logout":
			apiLogout(w, r)
		case "/api/nodes":
			apiNodes(w, r)
		case "/api/connect":
			apiConnect(w, r)
		case "/api/disconnect":
			apiDisconnect(w, r)
		case "/api/check-update":
			apiCheckUpdate(w, r)
		case "/api/do-update":
			apiDoUpdate(w, r)
		case "/favicon.ico":
			b, err := webFS.ReadFile("web/favicon.ico")
			if err == nil {
				w.Header().Set("Content-Type", "image/x-icon")
				w.Write(b)
			} else {
				w.WriteHeader(404)
			}
		default:
			http.NotFound(w, r)
		}
	})
	srv := &http.Server{Addr: LocalAddr, Handler: mux}
	if err := srv.ListenAndServe(); err != nil {
		// 端口被占用（可能已在运行）则直接打开浏览器
		fmt.Println("本地服务启动失败(可能已在运行):", err)
		openBrowser("http://" + LocalAddr)
	}
}
