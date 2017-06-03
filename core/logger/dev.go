// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logger

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

func NewDevLogger(omitTimeFor ...string) io.Writer {
	mu := &sync.Mutex{} // for now and last log
	lastLog := time.Now()
	distanceDuration := 850 * time.Millisecond

	return writerFunc(func(p []byte) (int, error) {
		logMessage := string(p)
		for _, s := range omitTimeFor {
			if strings.Contains(logMessage, s) {
				n, err := fmt.Print(logMessage)
				lastLog = time.Now()
				return n, err
			}
		}

		mu.Lock()
		defer mu.Unlock() // "slow" but we don't care here.
		nowLog := time.Now()
		if nowLog.Before(lastLog.Add(distanceDuration)) {
			// don't use the log.Logger to print this message
			// if the last one was printed before some seconds.
			n, err := fmt.Println(logMessage) // fmt because we don't want the time, dev is dev so console.
			lastLog = nowLog
			return n, err
		}

		// begin with new line in order to have the time once at the top
		// and the child logs below it.
		n, err := fmt.Printf("%s \u2192\n%s\n", nowLog.Format("01/02/2006 03:04:05"), logMessage)
		lastLog = nowLog
		return n, err
	})
}
