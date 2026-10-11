# 新启年VPN客户端

新启年VPN官方客户端，支持 Windows / Linux / 安卓(Termux)。

- 官网: https://www.newera2625.top
- VPN服务: https://vpn.newera2625.top
- SSO统一登录，节点自动测速，一键连接，自动更新

## 下载
请前往 Releases 页面下载对应平台的最新版本。

## 构建
```
go build -o vpn-client-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o vpn-client-windows-amd64.exe .
GOOS=linux GOARCH=arm64 go build -o vpn-client-linux-arm64 .
```
