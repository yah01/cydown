package cydown

import (
	"log"
	"testing"
	"time"
)

func TestDownload(t *testing.T) {
	//UseGlobalLocalProxy("1080")
	task := NewTask("https://dl.google.com/go/go1.13.6.windows-amd64.msi")
	log.Println("Downloading")
	TaskCounter.Add(1)
	go task.Download("")
	//finish := false
	//go func() {
	//	var last int64
	//	for !finish {
	//		fmt.Printf("%v Bytes/%v Bytes - %v Bytes/s\t\r", task.Count(), task.Size, task.Count()-last)
	//		last = task.Count()
	//		time.Sleep(time.Second)
	//	}
	//}()
	Wait()
	log.Println("All done")
	//finish = true
	time.Sleep(time.Second)
}
