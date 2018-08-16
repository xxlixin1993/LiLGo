package logging

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/xxlixin1993/LiLGo/configure"
	"github.com/xxlixin1993/LiLGo/utils"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

var (
	logName string
)

const (
	// 每隔多久刷新一次日志 单位秒
	flushInterval = 10

	// 默认日志路径
	defaultLogDir = "/tmp"

	// 缓冲上限
	maxSize = 256 * 1024

	// 缓冲区大小
	bufferSize = 1024 * 1024 * 1800
)

type LogFile struct {
	mu sync.Mutex
	*bufio.Writer
	logFile       *os.File
	LogDir        string
	MaxSize       uint64
	BufferSize    int
	FlushInterval uint64
	nBytes        uint64
}

func NewFileLog() ILog {
	logFile := &LogFile{
		LogDir:        configure.DefaultString("log.dir", defaultLogDir),
		FlushInterval: uint64(configure.DefaultInt("log.flush_interval", flushInterval)),
		MaxSize:       uint64(configure.DefaultInt("log.max_size", maxSize)),
		BufferSize:    configure.DefaultInt("log.buffer_size", bufferSize),
	}

	go logFile.flushDaemon()

	return logFile
}

// Initialize
func (f *LogFile) Init() error {
	return f.BeginLog(time.Now())
}

// Output message to log file
func (f *LogFile) OutputLogMsg(msg []byte) error {
	var err error

	f.mu.Lock()
	if f.nBytes+uint64(len(msg)) >= f.MaxSize {
		// When the buffer upper limit is reached, create a new file to write to avoid missing
		// Consider changing the configuration log.max_size when this happens
		if err = f.BeginLog(time.Now()); err != nil {
			return err
		}
	}
	n, err := f.Writer.Write(msg)
	f.nBytes += uint64(n)
	f.mu.Unlock()

	return err
}

func (f *LogFile) Flush() {
	f.lockAndFlushAll()
}

// Create a log file
func (f *LogFile) create(t time.Time) (osFile *os.File, filename string, err error) {
	fName := filepath.Join(f.LogDir, f.getName(t))
	fileHandle, err := os.Create(fName)
	if err != nil {
		return nil, "", fmt.Errorf("log: cannot create log: %v", err)
	}

	return fileHandle, fName, nil
}

// Begin log goroutine
func (f *LogFile) BeginLog(now time.Time) error {
	if f.logFile != nil {
		f.Flush()
		f.logFile.Close()
	}

	var err error

	f.logFile, _, err = f.create(now)
	f.nBytes = 0
	if err != nil {
		return err
	}

	f.Writer = bufio.NewWriterSize(f.logFile, f.BufferSize)

	// Write header.
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Log file created at: %s\n", utils.GetMicTimeFormat())
	fmt.Fprintf(&buf, "Build with %s for %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&buf, "Log line format: [FEWNITD]mmdd hh:mm:ss.uuuuuu threadid file:line] msg\n")
	n, err := f.logFile.Write(buf.Bytes())
	f.nBytes += uint64(n)
	return err
}

// Generate log file name
func (f *LogFile) getName(t time.Time) string {
	appName := configure.DefaultString("app.log_name", "game.log")
	logName = fmt.Sprintf("%s.%04d%02d%02d-%02d%02d%02d.%d",
		appName,
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second(),
		configure.Pid)

	return logName
}

func (f *LogFile) lockAndFlushAll() {
	f.mu.Lock()
	f.flushAll()
	f.mu.Unlock()
}

func (f *LogFile) flushAll() {
	if f.logFile != nil {
		f.Writer.Flush()
		f.logFile.Sync()
	}
}

// Timed write the data of the buffer to the file
func (f *LogFile) flushDaemon() {
	for range time.NewTicker(time.Duration(f.FlushInterval) * time.Second).C {
		f.lockAndFlushAll()
	}
}

// Returns Log file name
func GetLogName() string {
	return logName
}
