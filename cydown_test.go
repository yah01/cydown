package cydown

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestDownload(t *testing.T) {
	log.Println("Start test")
	UseGlobalLocalProxy("1080")
	task := NewTask("https://dl.google.com/go/go1.13.6.windows-amd64.msi")
	EnableLog()
	log.Println("Downloading")
	TaskCounter.Add(1)
	go task.Download("")
	finish := false
	go func() {
		var last int64
		for !finish {
			fmt.Printf("\r%v Bytes/%v Bytes - %v Bytes/s\t\t", task.Count(), task.Size, task.Count()-last)
			last = task.Count()
			time.Sleep(time.Second)
		}
	}()
	Wait()
	finish = true
	fmt.Println()
	log.Println("All done")
	time.Sleep(time.Second)
}
