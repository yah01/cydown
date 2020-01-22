package cydown

import (
	"fmt"
	"io"
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
// save as the name from url when fileName equals to empty string
func Download(url string, fileName string) {
	task := NewTask(url)
	task.Download(fileName)
}

type Range = [2]int64
type DownloadThread struct {
	Index int
	Range Range
	Recv  int64
	proxy ProxyFn
}

func InitThreads(threads []DownloadThread, size int64) {
	traceLog.Println("InitThreads", size)
	defer traceLog.Println("InitThreads", size, "Done")
	partSize := size / int64(len(threads))
	for i := range threads {
		// Set Range of downloading thread
		threads[i] = DownloadThread{
			Range: Range{int64(i) * partSize, int64((i+1))*partSize - 1},
		}
		if i == len(threads)-1 {
			threads[i].Range = Range{int64(i) * partSize, size - 1}
		}
	}
}

func (thread *DownloadThread) Size() int64 {
	return thread.Range[1] - thread.Range[0] + 1
}

func (thread *DownloadThread) NewClient() *http.Client {
	traceLog.Println("NewClient")
	defer traceLog.Println("NewClient Done")

	var transport = &http.Transport{}
	if thread.proxy != nil {
		transport.Proxy = thread.proxy
	} else if globalProxy != nil {
		transport.Proxy = globalProxy
	}

	return &http.Client{Transport: transport}
}

type DownloadTask struct {
	URL        string
	Size       int64
	FileName   string
	threads    []DownloadThread
	waitThread sync.WaitGroup
}

func NewTask(url string) *DownloadTask {
	traceLog.Println("NewTask", url)
	defer traceLog.Println("NewTask", url, "Done")

	task := &DownloadTask{
		URL:      url,
		FileName: GetFileNameFromURL(url),
	}

	//log.Println("Request file info...")
	transport := &http.Transport{Proxy: globalProxy}
	client := &http.Client{Transport: transport}
	res, err := client.Get(url)
	if err != nil {
		errorLog.Println(err)
	}
	defer res.Body.Close()

	task.Size = res.ContentLength

	//log.Println("Init threads")
	// If the size of the file is less than 1MB, download it by only one thread
	if task.Size < 1024*1024 {
		task.threads = make([]DownloadThread, 1)
	} else {
		task.threads = make([]DownloadThread, ThreadNum)
	}
	InitThreads(task.threads, task.Size)
	//log.Println("Init done")

	return task
}

func (task *DownloadTask) Count() int64 {
	var count int64

	for i := range task.threads {
		count += task.threads[i].Recv
	}

	return count
}

// Start downloading.
// Download() is non-blocking
func (task *DownloadTask) Download(fileName string) {
	TaskCounter.Add(1)
	go task.download(fileName)
}

// Real implement of download
func (task *DownloadTask) download(fileName string) {
	defer TaskCounter.Done()

	traceLog.Println(task.URL, task.FileName, "Download")
	defer traceLog.Println(task.URL, task.FileName, "Download", "Done")

	if fileName != "" {
		task.FileName = fileName
	}

	tempFiles := make([]*os.File, len(task.threads))
	err := os.Mkdir("TMP"+task.FileName, 0644)
	if err != nil {
		errorLog.Println(err)
	}

	for i := range tempFiles {
		tempFiles[i], err = os.OpenFile(fmt.Sprintf("./TMP%s/tmp%v", task.FileName, i), os.O_CREATE|os.O_APPEND, 0644)

		if err != nil {
			errorLog.Println(err)
		}

		task.waitThread.Add(1)
		go task.StartThread(tempFiles, i)
	}

	// The downloading ends after this line
	task.waitThread.Wait()

	// Merge temp files to the origin file
	task.MergeTemp(tempFiles)

	// Delete the temp files and temp dir
	for i := range tempFiles {
		tempFileName := tempFiles[i].Name()
		tempFiles[i].Close()
		err = os.Remove(tempFileName)
		if err != nil {
			errorLog.Println(err)
		}
	}
	err = os.Remove("TMP" + task.FileName)
	if err != nil {
		errorLog.Println(err)
	}
}

// Turn on a thread to download
func (task *DownloadTask) StartThread(tempFiles []*os.File, i int) {
	traceLog.Println(task.URL, task.FileName, "StartThread", i)
	defer traceLog.Println(task.URL, task.FileName, "StartThread", i, "Done")

	var (
		thread = &task.threads[i]
		file   = tempFiles[i]
		req, _ = http.NewRequest("GET", task.URL, nil)
		client = thread.NewClient()
		res    *http.Response
		err    error
	)
	req.Header.Set("Range",
		fmt.Sprintf("bytes=%v-%v", thread.Range[0]+thread.Recv, thread.Range[1]))
	if res, err = client.Do(req); err != nil {
		errorLog.Println(err)
	}

	for thread.Recv < thread.Size() {
		// Download
		n, err := io.Copy(file, res.Body)
		if err != nil {
			errorLog.Println(n, err)
		}
		thread.Recv += n
		res.Body.Close()

		// Continue if downloading has not finished yet
		if thread.Recv < thread.Size() {
			req.Header.Set("Range",
				fmt.Sprintf("bytes=%v-%v", thread.Range[0]+thread.Recv, thread.Range[1]))
			if res, err = client.Do(req); err != nil {
				errorLog.Println(err)
			}
		}
	}
	task.waitThread.Done()
}

// Merge the temp files
func (task *DownloadTask) MergeTemp(tempFiles []*os.File) {
	traceLog.Println(task.URL, task.FileName, "MergeTemp")
	defer traceLog.Println(task.URL, task.FileName, "MergeTemp", "Done")
	file, _ := os.OpenFile(task.FileName, os.O_CREATE|os.O_APPEND, 0644)
	defer file.Close()

	for i := range tempFiles {
		tempFiles[i].Seek(0, os.SEEK_SET)
		io.Copy(file, tempFiles[i])
	}
}
