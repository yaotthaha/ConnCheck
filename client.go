package main

import (
	"context"
	"net"
	"strconv"
	"time"
)

func ClientRun(Config ClientConfigStruct) {
	ConnPool := make(chan *net.Conn)
	RetryChan := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		Retry := 0
		for {
			Conn, err := net.Dial("tcp", net.JoinHostPort(Config.DialAddr, strconv.Itoa(int(Config.DialPort))))
			if err != nil {
				Logout(2, err)
			} else {
				Logout(1, "Connected Server "+net.JoinHostPort(Config.DialAddr, strconv.Itoa(int(Config.DialPort))))
				ConnPool <- &Conn
				<-RetryChan
			}
			if Config.Retry < -1 {
				continue
			} else if Config.Retry == 0 {
				Logout(-1, "Server Connect Fail")
				break
			} else if Retry == Config.Retry {
				Logout(-1, "Server Connect Fail")
				break
			} else {
				Retry++
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
					_, err = (*Conn).Write(msg)
					if err != nil {
						_ = (*Conn).Close()
						Logout(2, "Send Msg Fail, Reconnect Server")
						RetryChan <- struct{}{}
						BreakTag = true
					}
				}
				if BreakTag {
					break
				} else {
					<-time.After(3 * time.Second)
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
