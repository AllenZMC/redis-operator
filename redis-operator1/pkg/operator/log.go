package operator

import (
	"github.com/sirupsen/logrus"
	"os"
)

var Logger *logrus.Logger

func NewLogger() {
	var log = logrus.New()
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
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
