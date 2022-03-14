package main

import "time"

const (
	AppName    = "ConnCheck"
	AppVersion = "v0.0.1"
	AppAuthor  = "Yaott"
)

var (
	SendTime          = 3 * time.Second
	ServerTerminal    = "/bin/bash"
	ServerTerminalArg = "-c"
	ReConnectTry      = 6
)

type ServerConfigStruct struct {
	ListenAddr            string `json:"addr"`
	ListenPort            uint16 `json:"port"`
	Terminal              string `json:"terminal"`
	TerminalArg           string `json:"terminal_arg"`
	ReConnectTry          uint   `json:"reconnect_try"`
	ClientSettingInServer []struct {
		Name                string `json:"name"`               // 客户端名
		PrivateKey          string `json:"private_key"`        // ECC 私钥
		BrokenLineTimeout   uint64 `json:"retry"`              // 断线重连超时时间
		BrokenLineRunScript string `json:"offline_run_script"` // 断线执行脚本
		OnlineRunScript     string `json:"online_run_script"`  // 上线执行脚本
	} `json:"clients"`
}

type ClientConfigStruct struct {
	DialAddr  string `json:"addr"`       // 服务器地址
	DialPort  uint16 `json:"port"`       // 服务器端口
	PublicKey string `json:"public_key"` // ECC 公钥
	Retry     int    `json:"retry"`      // 重试
}
