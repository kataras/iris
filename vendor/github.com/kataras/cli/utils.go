package cli

import (
	"time"
)

// ShowIndicator shows a silly terminal indicator for a process, close of the finish channel is done here.
func ShowIndicator(newLine bool) chan bool {
	finish := make(chan bool)
	go func() {
		if newLine {
			Output.Write([]byte("\n"))
		}
		Output.Write([]byte("|"))
		Output.Write([]byte("_"))
		Output.Write([]byte("|"))

		for {
			select {
			case v := <-finish:
				{
					if v {
						Output.Write([]byte("\010\010\010")) //remove the loading chars
						close(finish)
						return
					}
				}
			default:
				Output.Write([]byte("\010\010-"))
				time.Sleep(time.Second / 2)
				Output.Write([]byte("\010\\"))
				time.Sleep(time.Second / 2)
				Output.Write([]byte("\010|"))
				time.Sleep(time.Second / 2)
				Output.Write([]byte("\010/"))
				time.Sleep(time.Second / 2)
				Output.Write([]byte("\010-"))
				time.Sleep(time.Second / 2)
				Output.Write([]byte("|"))
			}
		}

	}()

	return finish
}
