package cydown

import (
	"fmt"
	"testing"
	"time"
)

func TestDownload(t *testing.T) {
	task := NewTask("https://dl.google.com/go/go1.13.6.windows-amd64.msi")
	go task.Download("")
	finish := false
	go func() {
		var last int64
		for !finish {
			fmt.Printf("%v Bytes/%v Bytes - %v Bytes/s\t\r", task.Count(), task.Size, task.Count()-last)
			last = task.Count()
			time.Sleep(time.Second)
		}
	}()
	Wait()
	finish = true
	time.Sleep(time.Second)
}
