package cydown

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
)

type ProxyFn = func(*http.Request) (*url.URL, error)

var (
	ThreadNum   int
	waitAllTask sync.WaitGroup
	globalProxy ProxyFn
)

func init() {
	ThreadNum = 10
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

func InitThreads(threads []DownloadThread, url string, size int64) {
	partSize := size / int64(len(threads))
	for i := range threads {
		// Set Range of downloading thread
		threads[i] = DownloadThread{
			Range: Range{int64(i) * partSize, int64((i+1))*partSize - 1},
		}
		if i == len(threads)-1 {
			threads[i].Range = Range{int64(i) * partSize, size - 1}
		}

		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", threads[i].Range[0], threads[i].Range[1]))
		client := threads[i].NewClient()
		threads[i].res, _ = client.Do(req)
	}
}

// Download from url and save as fileName,
// save as the name from url when fileName equals to empty string
func Download(url string, fileName string) {
	task := NewTask(url)
	task.Download(fileName)
}

func Wait() {
	waitAllTask.Wait()
}

type Range = [2]int64
type DownloadThread struct {
	Range Range
	Recv  int64
	res   *http.Response
	proxy ProxyFn
}

func (thread *DownloadThread) Size() int64 {
	return thread.Range[1] - thread.Range[0] + 1
}

func (thread *DownloadThread) NewClient() *http.Client {
	var transport *http.Transport
	if thread.proxy != nil {
		transport = &http.Transport{
			Proxy: thread.proxy,
		}
		return &http.Client{Transport: transport}
	} else if globalProxy != nil {
		transport = &http.Transport{
			Proxy: globalProxy,
		}
		return &http.Client{Transport: transport}
	} else {
		return &http.Client{}
	}
}

type DownloadTask struct {
	URL        string
	Size       int64
	FileName   string
	threads    []DownloadThread
	waitThread sync.WaitGroup
}

func NewTask(url string) *DownloadTask {
	task := &DownloadTask{
		URL:      url,
		FileName: GetFileNameFromURL(url),
	}

	transport := &http.Transport{Proxy: globalProxy}
	client := &http.Client{Transport: transport}
	res, _ := client.Get(url)
	defer res.Body.Close()
	task.Size = res.ContentLength

	// If the size of the file is less than 1MB, download it by only one thread
	if task.Size < 1024*1024 {
		task.threads = make([]DownloadThread, 1)
	} else {
		task.threads = make([]DownloadThread, ThreadNum)
	}

	InitThreads(task.threads, task.URL, task.Size)
	var totalSize int64
	for i := range task.threads {
		totalSize += task.threads[i].Size()
		log.Println(totalSize)
	}
	return task
}

func (task *DownloadTask) Download(fileName string) {
	waitAllTask.Add(1)
	defer waitAllTask.Done()

	if fileName != "" {
		task.FileName = fileName
	}
	tempFiles := make([]*os.File, len(task.threads))
	err := os.Mkdir("TMP"+task.FileName, 0644)
	if err != nil {
		log.Println(err)
	}
	for i := range tempFiles {
		tempFiles[i], err = os.OpenFile(fmt.Sprintf("./TMP%s/tmp%v", task.FileName, i), os.O_CREATE|os.O_APPEND, 0644)

		if err != nil {
			log.Println(err)
		}

		task.waitThread.Add(1)

		startThread := func(task *DownloadTask, i int) {
			defer task.waitThread.Done()
			thread := &task.threads[i]
			file := tempFiles[i]
			req, _ := http.NewRequest("GET", task.URL, nil)
			client := thread.NewClient()

			// Set header range
			req.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", thread.Range[0], thread.Range[1]))

			// Data buffer
			//var data = make([]byte, 4*1024)

			for thread.Recv < thread.Size() {
				// Download
				file.Seek(0, os.SEEK_END)
				n, err := io.Copy(file, thread.res.Body)
				if err != nil {
					log.Println(n, err)
				}
				thread.Recv += n
				thread.res.Body.Close()

				fileInfo, _ := os.Stat(file.Name())
				if thread.Recv != fileInfo.Size() {
					log.Println("Error!")
				}

				// Continue if downloading has not finished yet
				if thread.Recv < thread.Size() {
					req.Header.Set("Range",
						fmt.Sprintf("bytes=%v-%v", thread.Range[0]+thread.Recv, thread.Range[1]))
					thread.res, _ = client.Do(req)
				} else if thread.Recv != thread.Size() {
					log.Println("recv,size:",
						thread.Recv, thread.Size())
				}
			}
		}

		go startThread(task, i)
	}

	// The downloading ends after this line
	task.waitThread.Wait()
	for i := range task.threads {
		fileInfo, _ := os.Stat(tempFiles[i].Name())
		log.Println("recv,fileSize", task.threads[i].Recv, fileInfo.Size())
	}
	task.MergeTemp(tempFiles)
}

func (task *DownloadTask) StartThread(i int) {
	defer task.waitThread.Done()
	thread := &task.threads[i]
	file := tempFiles[i]
	req, _ := http.NewRequest("GET", task.URL, nil)
	client := thread.NewClient()

	// Set header range
	req.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", thread.Range[0], thread.Range[1]))

	// Data buffer
	//var data = make([]byte, 4*1024)

	for thread.Recv < thread.Size() {
		// Download
		file.Seek(0, os.SEEK_END)
		n, err := io.Copy(file, thread.res.Body)
		if err != nil {
			log.Println(n, err)
		}
		thread.Recv += n
		thread.res.Body.Close()

		fileInfo, _ := os.Stat(file.Name())
		if thread.Recv != fileInfo.Size() {
			log.Println("Error!")
		}

		// Continue if downloading has not finished yet
		if thread.Recv < thread.Size() {
			req.Header.Set("Range",
				fmt.Sprintf("bytes=%v-%v", thread.Range[0]+thread.Recv, thread.Range[1]))
			thread.res, _ = client.Do(req)
		} else if thread.Recv != thread.Size() {
			log.Println("recv,size:",
				thread.Recv, thread.Size())
		}
	}
}

func (task *DownloadTask) MergeTemp(files []*os.File) {
	file, _ := os.OpenFile(task.FileName, os.O_CREATE|os.O_APPEND, 0644)
	defer file.Close()

	for i := range files {
		files[i].Seek(0, os.SEEK_SET)
		io.Copy(file, files[i])
		files[i].Close()
	}
}
