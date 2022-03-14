package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
)

var Params struct {
	LogFile string
	Debug   bool
	Help    bool
	Version bool
	Mode    string
	Config  string
	GenKey  bool
}

var (
	Log       log.Logger
	WaitGroup sync.WaitGroup
)

func init() {
	flag.StringVar(&Params.LogFile, "logfile", "", "Set Log File")
	flag.BoolVar(&Params.Debug, "debug", false, "Set Debug Mode")
	flag.BoolVar(&Params.Help, "h", false, "Show Help")
	flag.BoolVar(&Params.Version, "v", false, "Show Version")
	flag.StringVar(&Params.Mode, "mode", "", "Set Run Mode")
	flag.StringVar(&Params.Config, "c", "./config.json", "Set Config File Path")
	flag.BoolVar(&Params.GenKey, "gen", false, "Gen ECC Key")
	flag.Parse()
}

func main() {
	if Params.Help {
		flag.Usage()
		return
	}
	if Params.Version {
		_, _ = fmt.Fprintln(os.Stdout, AppName, AppVersion, "Build From", AppAuthor)
		return
	}
	if Params.GenKey {
		pub, pri, err := GenEccKey()
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "gen fail:", err)
		} else {
			FilePub, err := os.Create("./eccpublic.pem")
			defer func(FilePub *os.File) {
				_ = FilePub.Close()
			}(FilePub)
			_, err = FilePub.Write(Base64Encode(pub))
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, "file save fail")
				return
			}
			FilePri, err := os.Create("./eccprivate.pem")
			defer func(FilePri *os.File) {
				_ = FilePri.Close()
			}(FilePri)
			_, err = FilePri.Write(Base64Encode(pri))
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, "file save fail")
				return
			}
			_, _ = fmt.Fprintln(os.Stdout, "OK")
		}
		return
	}
	Log = *SetLog()
	Logout(1, AppName, AppVersion, "(Build From", AppAuthor+")")
	CtrlC()
	switch Params.Mode {
	case "server", "client":
	default:
		Logout(-1, "invalid mode, only support server and client")
	}
	Logout(1, "Run Mode", Params.Mode)
	Logout(1, "Read Config", Params.Config)
	Config, err := ReadConfig(Params.Config, Params.Mode)
	if err != nil {
		Logout(-1, err)
	}
	switch Params.Mode {
	case "server":
		Logout(1, "Start Server", net.JoinHostPort(Config.(ServerConfigStruct).ListenAddr, strconv.Itoa(int(Config.(ServerConfigStruct).ListenPort))))
		ServerRun(Config.(ServerConfigStruct))
	case "client":
		Logout(1, "Start Client")
		ClientRun(Config.(ClientConfigStruct))
	}
}

func SetLog() *log.Logger {
	LogInside := log.Logger{}
	LogInside.SetFlags(log.Ldate | log.Ltime)
	LogInside.SetPrefix("")
	LogInside.SetOutput(os.Stdout)
	return &LogInside
}

func Logout(LogLevel int, Message ...interface{}) {
	LogLevelTag := "Unknown"
	switch LogLevel {
	case 1:
		LogLevelTag = "Info"
	case 2:
		LogLevelTag = "Warning"
	case 3:
		LogLevelTag = "Debug"
		if !Params.Debug {
			return
		}
	case -1:
		LogLevelTag = "FatalError"
	case -2:
		LogLevelTag = "SimpleError"
	}
	var MessageAll string
	for _, v := range Message {
		MessageAll += fmt.Sprint(v) + " "
	}
	MessageAll = MessageAll[:len(MessageAll)-1]
	Log.Println("["+LogLevelTag+"]", MessageAll)
	if LogLevel == -1 {
		Logout(1, "Good Bye!!")
		os.Exit(-1)
	}
}

func CtrlC() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		Logout(1, "Good Bye!!")
		os.Exit(0)
	}()
}
