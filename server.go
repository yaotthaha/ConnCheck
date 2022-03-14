package main

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ClientRunParamsStruct struct {
	Status         bool
	Retry          int
	ReconnectedTag bool
}

type ClientSetting struct { // 运行池子项
	Setting struct {
		Name                string
		PrivateKey          []byte
		BrokenLineRetry     uint64
		BrokenLineRunScript string
		OnlineRunScript     string
	}
	RunParams ClientRunParamsStruct
	TimePool  struct {
		Mu   *sync.Mutex
		Time *time.Time
	}
}

var (
	ClientRunChan     chan ClientSetting
	ClientRunningPool map[string]*ClientSetting
	DataCachePool     struct {
		Mu   sync.Mutex
		Pool map[string]struct{}
	} // 防重放数据池
)

func ServerRun(Config ServerConfigStruct) {
	WaitGroup.Add(1)
	defer WaitGroup.Done()
	ClientRunChan = make(chan ClientSetting, len(Config.ClientSettingInServer)) // 初始化运行通道
	ClientRunningPool = make(map[string]*ClientSetting)                         // 初始化运行池
	for _, v := range Config.ClientSettingInServer {                            // 客户端配置丢进运行通道
		var TempSetting ClientSetting
		TempSetting.Setting.Name = v.Name
		TempSetting.RunParams.Status = false // 运行状态 false offline true online
		TempSetting.RunParams.Retry = 0      // 重试次数
		TempSetting.RunParams.ReconnectedTag = false
		TempSetting.Setting.PrivateKey = []byte(v.PrivateKey)
		TempSetting.Setting.BrokenLineRetry = v.BrokenLineRetry
		TempSetting.Setting.BrokenLineRunScript = v.BrokenLineRunScript
		TempSetting.Setting.OnlineRunScript = v.OnlineRunScript
		TempSetting.TimePool.Mu = &sync.Mutex{}
		TempSetting.TimePool.Time = nil
		ClientRunChan <- TempSetting
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ListenConfig := net.ListenConfig{}
	Listener, err := ListenConfig.Listen(ctx, "tcp", net.JoinHostPort(Config.ListenAddr, strconv.Itoa(int(Config.ListenPort))))
	if err != nil {
		Logout(-1, "server listen fail:", err.Error())
	}
	DataCachePool.Pool = make(map[string]struct{}, 0) // 初始化缓存池
	go CacheDel()                                     // 运行缓存删除函数
	go PoolRun()                                      // 运行检测函数
	for {
		Conn, err := Listener.Accept()
		if err != nil {
			continue
		}
		go func(conn net.Conn) {
			c, err := ConnValidCheck(conn)
			if err != nil {
				_ = conn.Close()
			} else {
				Logout(1, "New Client ["+ClientRunningPool[c].Setting.Name+"]")
				go Handler(conn, c)
			}
		}(Conn)
	}
}

func PoolRun() {
	//
	//检测是否在线
	//--- 是
	//  检测Conn时间
	//  --- 超时
	//    重试过期
	//    --- 是
	//      -- 运行脚本
	//      -- SetRetry => 0
	//      -- SetStatus => false
	//      -- SetConnTime => nil
	//    --- 否
	//      -- SetRetry++
	//  --- 未超时
	//      -- SetRetry => 0
	//--- 否
	//  检测Conn时间
	//  --- 有
	//    -- SetStatus => true
	//    -- 运行脚本
	//    -- JUMP
	//  --- 没有
	//    -- JUMP
	//
	var wg sync.WaitGroup
	Run := func(c *ClientSetting) {
		defer wg.Done()
		for {
			<-time.After(1 * time.Second)
			if c.RunParams.Status {
				c.TimePool.Mu.Lock()
				if c.TimePool.Time.Add(SendTime + 1*time.Second).Before(*TimeNow()) {
					if c.RunParams.Retry == int(c.Setting.BrokenLineRetry) {
						Logout(1, "["+c.Setting.Name+"] Offline")
						c.TimePool.Time = nil
						c.TimePool.Mu.Unlock()
						c.RunParams.Retry = 0
						c.RunParams.Status = false
						c.RunParams.ReconnectedTag = true
						if c.Setting.BrokenLineRunScript != "" {
							Logout(1, "["+c.Setting.Name+"] Run OfflineScript:", c.Setting.BrokenLineRunScript)
							CommandRun(c.Setting.BrokenLineRunScript)
						}
					} else {
						c.RunParams.Retry++
						c.RunParams.ReconnectedTag = true
						Logout(3, "["+c.Setting.Name+"] Disconnect, Retry", c.RunParams.Retry, "...")
						c.TimePool.Mu.Unlock()
					}
				} else {
					if c.RunParams.ReconnectedTag {
						Logout(3, "Reconnected")
						c.RunParams.ReconnectedTag = false
					}
					c.RunParams.Retry = 0
					c.TimePool.Mu.Unlock()
				}
			} else {
				c.TimePool.Mu.Lock()
				GetTime := c.TimePool.Time
				c.TimePool.Mu.Unlock()
				if GetTime != nil {
					Logout(1, "["+c.Setting.Name+"] Online")
					c.RunParams.Status = true
					if c.Setting.OnlineRunScript != "" {
						Logout(1, "["+c.Setting.Name+"] Run OnlineScript:", c.Setting.OnlineRunScript)
						CommandRun(c.Setting.OnlineRunScript)
					}
				}
			}
		}
	}
	for {
		BreakTag := false
		select {
		case ClientData := <-ClientRunChan:
			ClientRunningPool[ClientData.Setting.Name] = &ClientData
			wg.Add(1)
			go Run(ClientRunningPool[ClientData.Setting.Name])
		default:
			BreakTag = true
			break
		}
		if BreakTag {
			break
		}
	}
	wg.Wait()
}

func CacheDel() {
	for {
		<-time.After(2 * time.Second)
		DataCachePool.Mu.Lock()
		for k := range DataCachePool.Pool {
			ToInt64, _ := strconv.ParseInt(k, 10, 64)
			ToTime := time.Unix(ToInt64, 0)
			if ToTime.Add(1 * time.Second).Before(time.Now()) {
				delete(DataCachePool.Pool, k)
			}
		}
		DataCachePool.Mu.Unlock()
	}
}

func RawDataEccDecrypt(DataRaw, Key []byte) bool {
	data, err := EccDecrypt(DataRaw, Key)
	if err != nil {
		return false
	} else {
		dataToString := string(data)
		ToInt64, err := strconv.ParseInt(dataToString, 10, 64)
		if err != nil {
			return false
		}
		ToTime := time.Unix(ToInt64, 0)
		if ToTime.Add(10 * time.Second).Before(time.Now()) {
			return false
		}
		Tag := false
		DataCachePool.Mu.Lock()
		for k := range DataCachePool.Pool {
			if k == dataToString {
				Tag = true
			}
		}
		DataCachePool.Mu.Unlock()
		if !Tag {
			DataCachePool.Mu.Lock()
			DataCachePool.Pool[dataToString] = struct{}{}
			DataCachePool.Mu.Unlock()
			return true
		} else {
			return false
		}
	}
} // 验证数据合法性

func ConnValidCheck(Conn net.Conn) (string, error) {
	buf := make([]byte, 4096)
	Len, err := Conn.Read(buf)
	if err != nil {
		return "", err
	}
	DataRv := buf[:Len]
	var wg sync.WaitGroup
	RvChan := make(chan string, len(ClientRunningPool))
	for k, v := range ClientRunningPool {
		wg.Add(1)
		go func(c *ClientSetting, key string) {
			defer wg.Done()
			ok := RawDataEccDecrypt(DataRv, c.Setting.PrivateKey)
			if ok {
				RvChan <- key
			}
		}(v, k)
	}
	wg.Wait()
	if len(RvChan) <= 0 {
		return "", errors.New("invalid data")
	}
	return <-RvChan, nil
} // 验证连接合法性

func Handler(Conn net.Conn, ClientID string) {
	defer func(Conn net.Conn) {
		_ = Conn.Close()
	}(Conn)
	ClientRunningPool[ClientID].TimePool.Mu.Lock()
	ClientRunningPool[ClientID].TimePool.Time = TimeNow()
	ClientRunningPool[ClientID].TimePool.Mu.Unlock()
	RvChan := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func(CTX context.Context) {
		for {
			BreakTag := false
			select {
			case <-CTX.Done():
				BreakTag = true
				break
			default:
				buf := make([]byte, 4096)
				Len, err := Conn.Read(buf)
				if err != nil {
					if strings.Contains(err.Error(), "EOF") {
						RvChan <- err
						BreakTag = true
						break
					}
					continue
				}
				ok := RawDataEccDecrypt(buf[:Len], ClientRunningPool[ClientID].Setting.PrivateKey)
				if !ok {
					RvChan <- errors.New("invalid data")
					BreakTag = true
					break
				} else {
					RvChan <- nil
				}
			}
			if BreakTag {
				break
			}
		}
	}(ctx)
	RetryTime := 0
	for {
		BreakTag := false
		select {
		case r := <-RvChan:
			if r != nil {
				BreakTag = true
				break
			}
			RetryTime = 0
			ClientRunningPool[ClientID].TimePool.Mu.Lock()
			ClientRunningPool[ClientID].TimePool.Time = TimeNow()
			ClientRunningPool[ClientID].TimePool.Mu.Unlock()
		case <-time.After(SendTime):
			RetryTime++
			if RetryTime == 3 {
				cancel()
				BreakTag = true
				break
			}
		}
		if BreakTag {
			break
		}
	}
}
