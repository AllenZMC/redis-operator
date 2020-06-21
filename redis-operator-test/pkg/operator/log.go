package operator

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"runtime"
	"strings"
)

var Logger *logrus.Logger

func NewLogger() {
	var log = logrus.New()
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)

	filenameHook := NewHook()
	filenameHook.Field = "line"
	log.AddHook(filenameHook)

	log.Out = os.Stdout

	// You could set this to any `io.Writer` such as a file
	file, err := os.OpenFile("/tmp/redis.log", os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		log.Out = file
	} else {
		log.Info("Failed to log to file, using default stderr")
	}

	Logger = log
}

func ToggleDebug(toggleDebugMode bool) {
	//if toggleDebugMode {
	Logger.SetLevel(logrus.DebugLevel)
	//}
}

type Hook struct {
	Field     string
	Skip      int
	levels    []logrus.Level
	Formatter func(file, function string, line int) string
}

func (hook *Hook) Levels() []logrus.Level {
	return hook.levels
}

func (hook *Hook) Fire(entry *logrus.Entry) error {
	entry.Data[hook.Field] = hook.Formatter(findCaller(hook.Skip))
	return nil
}

func NewHook(levels ...logrus.Level) *Hook {
	hook := Hook{
		Field:  "source",
		Skip:   5,
		levels: levels,
		Formatter: func(file, function string, line int) string {
			return fmt.Sprintf("%s:%d", file, line)
		},
	}
	if len(hook.levels) == 0 {
		hook.levels = logrus.AllLevels
	}

	return &hook
}

func findCaller(skip int) (string, string, int) {
	var (
		pc       uintptr
		file     string
		function string
		line     int
	)
	for i := 0; i < 10; i++ {
		pc, file, line = getCaller(skip + i)
		if !strings.HasPrefix(file, "logrus") {
			break
		}
	}
	if pc != 0 {
		frames := runtime.CallersFrames([]uintptr{pc})
		frame, _ := frames.Next()
		function = frame.Function
	}

	return file, function, line
}

func getCaller(skip int) (uintptr, string, int) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return 0, "", 0
	}

	n := 0
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			n += 1
			if n >= 2 {
				file = file[i+1:]
				break
			}
		}
	}

	return pc, file, line
}
