package cydown

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

type Task struct {
	URL        string
	Size       int64
	FileName   string
	threads    []Thread
	waitThread sync.WaitGroup
}

type taskJSON struct {
	URL      string
	Size     int64
	FileName string
	Threads  []Thread
}

func NewTask(url string) *Task {
	traceLog.Println("NewTask", url)
	defer traceLog.Println("NewTask", url, "Done")

	task := &Task{
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
		task.threads = make([]Thread, 1)
	} else {
		task.threads = make([]Thread, ThreadNum)
	}
	InitThreads(task.threads, task.Size)
	//log.Println("Init done")

	return task
}

func (task *Task) MarshalJSON() ([]byte, error) {
	return json.Marshal(taskJSON{
		task.URL,
		task.Size,
		task.FileName,
		task.threads,
	})
}

func (task *Task) UnmarshalJSON(b []byte) error {
	var JSON taskJSON
	if err := json.Unmarshal(b, &JSON); err != nil {
		return err
	}

	task = &Task{
		URL:        JSON.URL,
		Size:       JSON.Size,
		FileName:   JSON.FileName,
		threads:    JSON.Threads,
		waitThread: sync.WaitGroup{},
	}
	return nil
}

func (task *Task) Count() int64 {
	var count int64

	for i := range task.threads {
		count += task.threads[i].Recv
	}

	return count
}

func (task *Task) DirName() string {
	return "TMP" + task.FileName
}

// Start downloading.
// Download() is non-blocking
func (task *Task) Download(fileName string) {
	TaskCounter.Add(1)
	go task.download(fileName)
}

// Real implement of download
func (task *Task) download(fileName string) {
	defer TaskCounter.Done()

	traceLog.Println(task.URL, task.FileName, "Download")
	defer traceLog.Println(task.URL, task.FileName, "Download", "Done")

	if fileName != "" {
		task.FileName = fileName
	}

	tempFiles := make([]*os.File, len(task.threads))
	err := os.Mkdir(task.DirName(), 0644)
	if err != nil && err != os.ErrExist {
		errorLog.Println(err)
	}

	for i := range tempFiles {
		tempFiles[i], err = os.OpenFile(fmt.Sprintf("./%s/tmp%v", task.DirName(), i),
			os.O_CREATE|os.O_APPEND, 0644)

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

// Start a thread to download
func (task *Task) StartThread(tempFiles []*os.File, i int) {
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
func (task *Task) MergeTemp(tempFiles []*os.File) {
	traceLog.Println(task.URL, task.FileName, "MergeTemp")
	file, _ := os.OpenFile(task.FileName, os.O_CREATE|os.O_APPEND, 0644)
	defer traceLog.Println(task.URL, task.FileName, "MergeTemp", "Done")
	defer file.Close()

	for i := range tempFiles {
		tempFiles[i].Seek(0, os.SEEK_SET)
		io.Copy(file, tempFiles[i])
	}
}
