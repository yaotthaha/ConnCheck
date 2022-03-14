package main

import (
	"context"
	"net"
	"strconv"
	"sync"
	"time"
)

func ClientRun(Config ClientConfigStruct) {
	ConnPool := make(chan *net.Conn)
	RetryChan := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var Retry struct {
		Mu    sync.Mutex
		Value int
	}
	Retry.Value = 0
	go func() {
		for {
			Conn, err := net.Dial("tcp", net.JoinHostPort(Config.DialAddr, strconv.Itoa(int(Config.DialPort))))
			if err == nil {
				ConnPool <- &Conn
				<-RetryChan
				_ = Conn.Close()
			}
			if Config.Retry < 0 {
				<-time.After(2 * time.Second)
				continue
			} else if Config.Retry == 0 {
				Logout(-1, "Server Connect Fail")
				break
			} else {
				Retry.Mu.Lock()
				if Retry.Value == Config.Retry {
					Retry.Mu.Unlock()
					Logout(-1, "Server Connect Fail")
					break
				} else {
					Retry.Value++
					Retry.Mu.Unlock()
				}
			}
		}
		cancel()
	}()
	BreakTagMax := false
	for {
		select {
		case Conn := <-ConnPool:
			BreakTag := false
			for {
				Data := strconv.FormatInt(TimeNow().Unix(), 10)
				msg, err := EccEncrypt([]byte(Data), []byte(Config.PublicKey))
				if err != nil {
					_ = (*Conn).Close()
					Logout(-1, "ECC Encrypt Fail")
				}
				_, err = (*Conn).Write(msg)
				if err != nil {
					Logout(2, "Send Msg Fail, Reconnect Server")
					RetryChan <- struct{}{}
					BreakTag = true
				}
				Retry.Mu.Lock()
				Retry.Value = 0
				Retry.Mu.Unlock()
				if BreakTag {
					break
				} else {
					<-time.After(SendTime)
				}
			}
		case <-ctx.Done():
			BreakTagMax = true
			break
		}
		if BreakTagMax {
			break
		}
	}
}
