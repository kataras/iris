package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

// NewDev returns a new Logger which prints the critical messages to the console.
func NewDev() Logger {
	lastLog := time.Now()
	loggerOuput := log.New(os.Stdout, "", log.LstdFlags)
	distanceDuration := 850 * time.Millisecond
	return func(errorMessage string) {
		nowLog := time.Now()
		if nowLog.Before(lastLog.Add(distanceDuration)) {
			fmt.Println(errorMessage)
			lastLog = nowLog
			return
		}

		loggerOuput.Println("\u2192\n" + errorMessage)
		lastLog = nowLog
	}
}
