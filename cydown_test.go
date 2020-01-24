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
	task.Download("")

	finish := false

	// Show the rate
	go func() {
		var last int64
		for !finish {
			fmt.Printf("\r%v Bytes/%v Bytes - %v Bytes/s\t\t", task.Count(), task.size, task.Count()-last)
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

func TestLoadSave(t *testing.T) {
	log.Println("Start test")

	UseGlobalLocalProxy("1080")
	task := NewTask("https://dl.google.com/go/go1.13.6.windows-amd64.msi")
	EnableLog()

	log.Println("Downloading")
	task.Download("")

	log.Println("Waiting...")
	time.Sleep(time.Second * 10)
	log.Println("Stop!")

	task.Stop()
	task.Save()
	Load(task.FileName+".json", task)
	log.Println("Continue")
	task.Download("")

	Wait()
	log.Println("All done")
	time.Sleep(time.Second)
}
