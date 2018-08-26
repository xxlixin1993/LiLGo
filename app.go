package main

import (
	"flag"
	"fmt"
	"github.com/xxlixin1993/LiLGo/configure"
	"github.com/xxlixin1993/LiLGo/graceful"
	"github.com/xxlixin1993/LiLGo/logging"
	"github.com/xxlixin1993/LiLGo/server"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

const (
	KVersion = "0.0.1"
)

func main() {
	initFrame()
	waitSignal()
}

// Initialize framework
func initFrame() {
	// Parsing configuration environment
	runMode := flag.String("m", "local", "Use -m <config mode>")
	configFile := flag.String("c", "./app.ini", "use -c <config file>")
	version := flag.Bool("v", false, "Use -v <current version>")
	flag.Parse()

	// Show version
	if *version {
		fmt.Println("Version", KVersion, runtime.GOOS+"/"+runtime.GOARCH)
		os.Exit(0)
	}

	// Initialize exitList
	graceful.InitExitList()

	// Initialize configure
	configErr := configure.InitConfig(*configFile, *runMode)
	if configErr != nil {
		fmt.Printf("Initialize Configure error : %s", configErr)
		os.Exit(configure.InitConfigError)
	}

	// Initialize log
	logErr := logging.InitLog()
	if logErr != nil {
		fmt.Printf("Initialize log error : %s", logErr)
		os.Exit(configure.InitLogError)
	}

	// TODO just test
	eh := server.NewEasyHandler()
	eh.GET("/", hello)
	go eh.StartHTTPServer()

	logging.Trace("Initialized frame")
}

func hello(context server.Context) error {
	res := make(map[string]interface{})
	res["message"] = "ok"
	return context.JSON(200, res)

}

// Wait signal
func waitSignal() {
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan)

	sig := <-sigChan

	logging.TraceF("signal: %d", sig)

	switch sig {
	case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
		logging.Trace("exit...")
		graceful.GetExitList().Stop()
	case syscall.SIGUSR1:
		logging.Trace("catch the signal SIGUSR1")
	default:
		logging.Trace("signal do not know")
	}
}
