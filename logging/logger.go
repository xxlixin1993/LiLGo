package logging

import (
	"errors"
	"fmt"
	"github.com/xxlixin1993/LiLGo/configure"
	"github.com/xxlixin1993/LiLGo/graceful"
	"github.com/xxlixin1993/LiLGo/utils"
	"path"
	"runtime"
	"sync"
)

// Log message level
const (
	LevelFatal = iota
	LevelError
	LevelWarning
	LevelNotice
	LevelInfo
	LevelTrace
	LevelDebug
)

// Log module name
const LogModuleName = "logModule"

// Log output message level abbreviation
var LevelName = [7]string{"F", "E", "W", "N", "I", "T", "D"}

// Log instance
var loggerInstance *LogBase

// Log output type
const (
	OutputFile   = "file"
	OutputStdout = "stdout"
)

// Log interface. Need to be implemented when you want to extend.
type ILog interface {
	// Initialize Logger
	Init() error

	// Output message to log
	OutputLogMsg(msg []byte) error

	Flush()
}

// Log core program
type LogBase struct {
	mu sync.Mutex
	sync.WaitGroup
	handle  ILog
	message chan []byte
	skip    int
	level   int
}

// Implement ExitInterface
func (l *LogBase) GetModuleName() string {
	return LogModuleName
}

// Implement ExitInterface
func (l *LogBase) Stop() error {
	close(loggerInstance.message)
	loggerInstance.Wait()
	return nil
}

// Initialize Log
func InitLog() error {
	outputType := configure.DefaultString("log.output", OutputStdout)
	level := configure.DefaultInt("log.level", LevelDebug)

	logger, err := createLogger(outputType, level)
	if err != nil {
		return err
	}

	logger.handle.Init()
	graceful.GetExitList().Pop(logger)

	go logger.Run()

	return err
}

// Create Logger instance
func createLogger(outputType string, level int) (*LogBase, error) {
	switch outputType {
	case OutputStdout:
		loggerInstance = &LogBase{
			handle:  NewStdoutLog(),
			message: make(chan []byte, 1000),
			skip:    3,
			level:   level,
		}
		return loggerInstance, nil
	case OutputFile:
		loggerInstance = &LogBase{
			handle:  NewFileLog(),
			message: make(chan []byte, 1000),
			skip:    3,
			level:   level,
		}
		return loggerInstance, nil
	default:
		return nil, errors.New(configure.UnknownTypeMsg)
	}
}

// Get Logger instance
func GetLogger() *LogBase {
	return loggerInstance
}

// Start log, receive information, wait information
func (l *LogBase) Run() {
	loggerInstance.Add(1)

	for {
		msg, ok := <-l.message
		if !ok {
			l.Done()
			l.handle.Flush()
			break
		}
		err := l.handle.OutputLogMsg(msg)
		if err != nil {
			fmt.Printf("Log: Output handle fail, err:%v\n", err.Error())
		}
	}
}

// Output message
func (l *LogBase) Output(nowLevel int, msg string) {
	now := utils.GetMicTimeFormat()

	l.mu.Lock()
	defer l.mu.Unlock()

	if nowLevel <= l.level {
		_, file, line, ok := runtime.Caller(l.skip)
		if !ok {
			file = "???"
			line = 0
		}
		_, filename := path.Split(file)
		msg = fmt.Sprintf("[%s] [%s %s:%d] %s\n", LevelName[nowLevel], now, filename, line, msg)
	}

	l.message <- []byte(msg)
}

func Debug(args ...interface{}) {
	msg := fmt.Sprint(args...)
	GetLogger().Output(LevelDebug, msg)
}

func DebugF(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	GetLogger().Output(LevelDebug, msg)
}

func Trace(args ...interface{}) {
	msg := fmt.Sprint(args...)
	GetLogger().Output(LevelTrace, msg)
}

func TraceF(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	GetLogger().Output(LevelTrace, msg)
}

func Info(args ...interface{}) {
	msg := fmt.Sprint(args...)
	GetLogger().Output(LevelInfo, msg)
}

func InfoF(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	GetLogger().Output(LevelInfo, msg)
}
func Notice(args ...interface{}) {
	msg := fmt.Sprint(args...)
	GetLogger().Output(LevelNotice, msg)
}

func NoticeF(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	GetLogger().Output(LevelNotice, msg)
}

func Warning(args ...interface{}) {
	msg := fmt.Sprint(args...)
	GetLogger().Output(LevelWarning, msg)
}

func WarningF(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	GetLogger().Output(LevelWarning, msg)
}

func Error(args ...interface{}) {
	msg := fmt.Sprint(args...)
	GetLogger().Output(LevelError, msg)
}

func ErrorF(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	GetLogger().Output(LevelError, msg)
}

func Fatal(args ...interface{}) {
	msg := fmt.Sprint(args...)
	GetLogger().Output(LevelFatal, msg)
}

func FatalF(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	GetLogger().Output(LevelFatal, msg)
}
