package cydown

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
)

type ProxyFn = func(*http.Request) (*url.URL, error)

var (
	ThreadNum   int
	TaskCounter sync.WaitGroup
	globalProxy ProxyFn

	traceLog *log.Logger
	errorLog *log.Logger
)

// Disable log for default.
func init() {
	ThreadNum = 10

	DisableLog()
}

func EnableLog() {
	traceLogFile, _ := os.OpenFile("cydownTrace.log", os.O_CREATE|os.O_APPEND, 0644)
	errorLogFile, _ := os.OpenFile("cydownError.log", os.O_CREATE|os.O_APPEND, 0644)
	traceLog = log.New(traceLogFile, "TRACE: ", log.LstdFlags)
	errorLog = log.New(errorLogFile, "Error: ", log.LstdFlags|log.Lshortfile)
}

func DisableLog() {
	traceLog = log.New(ioutil.Discard, "", 0)
	errorLog = log.New(ioutil.Discard, "Error: ", 0)
}

//func AddTask() {
//	TaskCounter.Add(1)
//}

func Wait() {
	TaskCounter.Wait()
}

func SetThreadNum(num int) {
	ThreadNum = num
}

func SetGlobalProxy(proxyFn func(r *http.Request) (*url.URL, error)) {
	globalProxy = proxyFn
}

func UseGlobalLocalProxy(port string) {
	SetGlobalProxy(func(_ *http.Request) (*url.URL, error) {
		return url.Parse("http://127.0.0.1:" + port)
	})
}

func GetFileNameFromURL(url string) string {
	pos := len(url) - 1
	for url[pos] != '/' {
		pos--
	}
	return url[pos+1:]
}

// Download from url and save as fileName,
// save as the name from url when fileName equals to empty string.
// The function create a task and then call task.Download(),
// non-blocking as well.
func Download(url string, fileName string) {
	task := NewTask(url)
	task.Download(fileName)
}
