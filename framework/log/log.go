package log

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// level
const (
	IllegalLevel = -1
	DebugLevel   = iota
	ReleaseLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// level prefix
const (
	printDebugLevel   = "[debug]"
	printReleaseLevel = "[release]"
	printWarnLevel    = "[warn]"
	printErrorLevel   = "[error]"
	printFatalLevel   = "[fatal]"
)

type Logger struct {
	level        int
	pathname     string
	fileName     string
	baseFile     *os.File
	time         time.Time
	ch           chan string
	chNum        int
	exitCh       chan int
	rollSize     uint32 // unit: MB
	fullPathName string
	enableStdOut bool
	newFileTime  time.Time
	errToSkipMap map[string]struct{}
}

func redirectError(fmat string, args ...interface{}) {
	msg := fmt.Sprintf(fmat, args)
	if _, e := fmt.Fprintf(os.Stderr, msg); e != nil {
		fmt.Printf("error occured when redirecting err[%s]: %s\n", msg, e.Error())
	}
}

func getLogLevelInteger(strLevel string) int {
	switch strings.ToLower(strLevel) {
	case "debug":
		return DebugLevel
	case "release":
		return ReleaseLevel
	case "warn":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "fatal":
		return FatalLevel
	default:
		return IllegalLevel
	}
}

func New(strLevel string, pathname string, fileName string, chNum int, rollSize uint32) (*Logger, error) {
	level := getLogLevelInteger(strLevel)
	if level == -1 {
		return nil, errors.New("unknown level: " + strLevel)
	}

	logger := new(Logger)
	logger.level = level
	logger.pathname = pathname
	logger.fileName = fileName
	logger.time = time.Now()
	logger.ch = make(chan string, chNum)
	logger.chNum = chNum
	logger.exitCh = make(chan int)
	logger.rollSize = rollSize

	go logger.doLogTask()

	return logger, nil
}

// It's dangerous to call the method on logging
func (logger *Logger) CloseFile() {
	if logger.baseFile != nil {
		if er := logger.baseFile.Close(); er != nil {
			redirectError("close logger base file failed:%s", er.Error())
		}
		logger.baseFile = nil
	}
}

func (logger *Logger) SetLoglevel(strLevel string) {
	level := getLogLevelInteger(strLevel)

	if level != -1 {
		logger.level = level
	}
}

func (logger *Logger) GetLoglevel() int {
	return logger.level
}

func (logger *Logger) EnableStdOut(enable bool) {
	logger.enableStdOut = enable
}

func callStack() string {
	var (
		stackInfo  string
		f          runtime.Frame
		more       bool
		stackCount int
		index      int
		pc         = make([]uintptr, 64) // program caller
		frames     = runtime.CallersFrames(pc)
	)
	stackCount = runtime.Callers(3, pc) // Filter first 3 traces, which show invocations in log.go
	pc = pc[:stackCount]
	for {
		f, more = frames.Next()

		if f.Line == 0 {
			continue
		}
		// 不打印Logger的调用信息
		if index = strings.Index(f.File, "log."); index != -1 {
			continue
		}

		if index = strings.Index(f.File, "src"); index != -1 {
			f.File = f.File[index+len("src/"):]
		}
		stackInfo = fmt.Sprintf("%s%s\n\t%s:%d\n", stackInfo, f.Function, f.File, f.Line)
		if !more {
			break
		}
	}

	return stackInfo
}

func (logger *Logger) doPrintf(level int, printLevel string, format string, a ...interface{}) {
	if level < logger.level {
		return
	}

	sNow := fmt.Sprintf("%s ", time.Now().Format("2006-01-02 15:04:05.999999999"))

	_, file, line, _ := runtime.Caller(3)
	ss := strings.Split(file, "/")

	fileInfo := ss[len(ss)-1] + ":" + strconv.Itoa(line)

	format = sNow + printLevel + "[" + fileInfo + "] " + format
	format = fmt.Sprintf(format, a...)

	format += "\n"
	if level >= ErrorLevel {
		format += callStack()
	}

	fmt.Println(format)

	select {
	case logger.ch <- format:
	default:
		redirectError("fatal error: logger.chNum is full\n")
	}
	if level == FatalLevel {
		logger.ch <- ""
	}
}

func (logger *Logger) GetChanNum() int {
	return len(logger.ch)
}

func (logger *Logger) Debug(format string, a ...interface{}) {
	logger.doPrintf(DebugLevel, printDebugLevel, format, a...)
}

func (logger *Logger) Release(format string, a ...interface{}) {
	logger.doPrintf(ReleaseLevel, printReleaseLevel, format, a...)
}

func (logger *Logger) Warn(format string, a ...interface{}) {
	logger.doPrintf(WarnLevel, printWarnLevel, format, a...)
}

func (logger *Logger) Error(format string, a ...interface{}) {
	logger.doPrintf(ErrorLevel, printErrorLevel, format, a...)
}

func (logger *Logger) Fatal(format string, a ...interface{}) {
	logger.doPrintf(FatalLevel, printFatalLevel, format, a...)
}

var gLogger, _ = New("release", "", "", 100000, 50)

func Export(logger *Logger) {
	if logger != nil {
		if gLogger != nil {
			gLogger.CloseFile()
		}
		gLogger = logger
	}
}

func GetChanNum() int {
	return gLogger.GetChanNum()
}

func SetLogLevel(strLevel string) {
	if gLogger != nil {
		gLogger.SetLoglevel(strLevel)
	}
}

func GetLogLevel() int {
	if gLogger != nil {
		return gLogger.GetLoglevel()
	}

	return -1
}

func EnableStdOut(enable bool) {
	if gLogger != nil {
		gLogger.EnableStdOut(enable)
	}
}

func Debug(format string, a ...interface{}) {
	if DebugLevel < GetLogLevel() {
		return
	}
	gLogger.Debug(format, a...)
}

func Release(format string, a ...interface{}) {
	gLogger.Release(format, a...)
}

func Warn(format string, a ...interface{}) {
	gLogger.Warn(format, a...)
}

func Error(format string, a ...interface{}) {
	gLogger.Error(format, a...)
}

func Fatal(format string, a ...interface{}) {
	gLogger.Fatal(format, a...)
}

func Close() {
	close(gLogger.ch)
	<-gLogger.exitCh
}

func (logger *Logger) checkRoll() bool {
	if logger.fullPathName == "" {
		return false
	}

	if logger.rollSize == 0 {
		return false
	}

	if logger.baseFile == nil {
		return false
	}

	fi, er := logger.baseFile.Stat()
	if er == nil {
		if fi.Size() >= int64(logger.rollSize*1024*1024) {
			return true
		}
	}

	return false
}

func (logger *Logger) OpenNewFile() {
	now := time.Now()
	newFileName := path.Join(logger.pathname, logger.fileName)

	newFile, err := os.OpenFile(newFileName+".log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766)
	if err != nil {
		errMsg := fmt.Sprintf("fatal error: OpenFile %s failed\n", newFileName)
		if _, er := fmt.Fprintf(os.Stderr, errMsg); er != nil {
			fmt.Printf("redirect error failed, msg:%s, er:%s", errMsg, er.Error())
		}
		return
	}

	logger.fullPathName = newFileName
	logger.newFileTime = now

	logger.baseFile = newFile
}

func (logger *Logger) RollFile() {
	rollFileName := fmt.Sprintf("%s_%d%02d%02d%02d%02d%02d.log",
		logger.fileName,
		logger.newFileTime.Year(),
		logger.newFileTime.Month(),
		logger.newFileTime.Day(),
		logger.newFileTime.Hour(),
		logger.newFileTime.Minute(),
		logger.newFileTime.Second(),
	)
	rollFileName = path.Join(logger.pathname, rollFileName)

	rollFile, err := os.OpenFile(rollFileName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		redirectError("open roll file failed:%s", err.Error())
		return
	}

	_, err = logger.baseFile.Seek(0, 0)
	if err != nil {
		redirectError("seek base file failed:%s", err.Error())
	}
	_, err = io.Copy(rollFile, logger.baseFile)
	if err != nil {
		redirectError("copy base file failed:%s", err.Error())
	}

	err = rollFile.Close()
	if err != nil {
		redirectError("close rolled file failed:%s", err.Error())
	}
	logger.CloseFile()

	truncateFile, er := os.OpenFile(logger.fullPathName+".log", os.O_WRONLY|os.O_CREATE, 0644)
	if er == nil {
		err = truncateFile.Truncate(0)
		if err != nil {
			redirectError("Truncate failed:%s", err.Error())
		}
		err = truncateFile.Close()
		if err != nil {
			redirectError("close truncateFile failed:%s", err.Error())
		}
	}
}

func (logger *Logger) CheckFile() error {
	now := time.Now()
	if logger.baseFile == nil || now.Day() != logger.time.Day() {
		if logger.baseFile != nil {
			logger.RollFile()
		}

		if logger.pathname != "" {
			logger.OpenNewFile()
		} else {

		}
	}

	if logger.checkRoll() {
		logger.RollFile()
		logger.OpenNewFile()
	}

	logger.time = now

	return nil
}

func (logger *Logger) doLogTask() {
	var err error
	for {
		content, ok := <-logger.ch
		if !ok {
			break
		}
		if err = logger.CheckFile(); err != nil {
			continue
		}
		if content == "" {
			os.Exit(1)
		}

		if logger.baseFile != nil {
			_, err = logger.baseFile.WriteString(content)
			if err != nil {
				redirectError("write content failed:%s", err.Error())
				logger.CloseFile()
				logger.OpenNewFile()
			}
		} else {
			fmt.Println(content)
		}

		if logger.enableStdOut {
			fmt.Print(content)
		}
	}

	if logger.baseFile != nil {
		_, err = logger.baseFile.WriteString("logger exit\n")
		if err != nil {
			redirectError("write \"exit\" failed:%s", err.Error())
		}
	}

	if logger.enableStdOut {
		fmt.Println("logger exit")
	}

	logger.CloseFile()
	logger.exitCh <- 1
}
