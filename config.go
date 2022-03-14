package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"os"
)

func ReadConfig(FileName string, Mode string) (interface{}, error) {
	File, err := os.Open(FileName)
	if err != nil {
		return nil, errors.New("read config file fail: " + err.Error())
	}
	DataRaw, err := ioutil.ReadAll(File)
	if err != nil {
		return nil, errors.New("read config file fail: " + err.Error())
	}
	switch Mode {
	case "server":
		var ServerConfig ServerConfigStruct
		err = json.Unmarshal(DataRaw, &ServerConfig)
		if err != nil {
			return nil, errors.New("read config file fail: " + err.Error())
		}
		if ServerConfig.ListenAddr == "" {
			return nil, errors.New("invalid addr")
		}
		if ServerConfig.ListenPort == 0 {
			return nil, errors.New("invalid port")
		}
		if ServerConfig.Terminal != "" {
			ServerTerminal = ServerConfig.Terminal
		}
		if ServerConfig.Terminal != "" {
			ServerTerminalArg = ServerConfig.TerminalArg
		}
		if len(ServerConfig.ClientSettingInServer) <= 0 {
			return nil, errors.New("clients settings is nil")
		}
		for k, v := range ServerConfig.ClientSettingInServer {
			if v.PrivateKey == "" {
				return nil, errors.New("client[" + v.Name + "] invalid private key")
			} else if Pri, err := Base64Decode([]byte(v.PrivateKey)); err != nil {
				return nil, errors.New("client[" + v.Name + "] invalid private key")
			} else {
				ServerConfig.ClientSettingInServer[k].PrivateKey = string(Pri)
			}
		}
		return ServerConfig, nil
	case "client":
		var ClientConfig ClientConfigStruct
		err = json.Unmarshal(DataRaw, &ClientConfig)
		if err != nil {
			return nil, errors.New("read config file fail: " + err.Error())
		}
		if net.ParseIP(ClientConfig.DialAddr) == nil {
			return nil, errors.New("invalid addr")
		}
		if ClientConfig.DialPort == 0 {
			return nil, errors.New("invalid port")
		}
		if ClientConfig.PublicKey == "" {
			return nil, errors.New("invalid public key")
		} else if Pub, err := Base64Decode([]byte(ClientConfig.PublicKey)); err != nil {
			return nil, errors.New("invalid public key")
		} else {
			ClientConfig.PublicKey = string(Pub)
		}
		return ClientConfig, nil
	default:
		return nil, errors.New("invalid mode, only support server and client")
	}
}
