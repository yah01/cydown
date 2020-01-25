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
	GlobalThreadNum int
	TaskCounter     sync.WaitGroup

	globalProxy     ProxyFn
	traceLog *log.Logger
	errorLog *log.Logger
)

// Disable log for default.
func init() {
	GlobalThreadNum = 10
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

// Wait for all tasks finishing.
func Wait() {
	TaskCounter.Wait()
}

// Set the proxy for all tasks which without proxy function.
func SetGlobalProxy(proxyFn func(r *http.Request) (*url.URL, error)) {
	globalProxy = proxyFn
}

// Same as SetGlobalProxy() but only need the port.
func UseGlobalLocalProxy(port string) {
	SetGlobalProxy(func(_ *http.Request) (*url.URL, error) {
		return url.Parse("http://127.0.0.1:" + port)
	})
}

// Get the file name from URL
func GetFileNameFromURL(url string) string {
	pos := len(url) - 1
	for url[pos] != '/' {
		pos--
	}
	return url[pos+1:]
}

// Download from url and save as FileName,
// save as the name from url when FileName equals to empty string.
// The function create a task and then call task.Download(),
// non-blocking as well.
func Download(url string, fileName string) {
	task := NewTask(url)
	task.Download(fileName)
}
