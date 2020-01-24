package cydown

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
)

type Task struct {
	URL      string
	FileName string

	size       int64
	stop       bool
	syncSign   chan bool
	threads    []Thread
	tempFiles  []*os.File
	waitThread sync.WaitGroup
}

type taskJSON struct {
	URL      string
	FileName string
	Size     int64
	Threads  []Thread
}

func NewTask(url string) *Task {
	traceLog.Println("NewTask", url)
	defer traceLog.Println("NewTask", url, "Done")

	task := &Task{
		URL:      url,
		FileName: GetFileNameFromURL(url),
		syncSign: make(chan bool),
	}

	//log.Println("Request file info...")
	transport := &http.Transport{Proxy: globalProxy}
	client := &http.Client{Transport: transport}
	res, err := client.Get(url)
	if err != nil {
		errorLog.Println(err)
	}
	defer res.Body.Close()

	task.size = res.ContentLength

	//log.Println("Init threads")
	// If the size of the file is less than 1MB, download it by only one thread
	if task.size < 1024*1024 {
		task.threads = make([]Thread, 1)
	} else {
		task.threads = make([]Thread, ThreadNum)
	}
	InitThreads(task.threads, task.size)
	//log.Println("Init done")

	return task
}

func (task Task) MarshalJSON() ([]byte, error) {
	return json.Marshal(taskJSON{
		task.URL,
		task.FileName,

		task.size,
		task.threads,
	})
}

func (task *Task) UnmarshalJSON(data []byte) error {
	var JSON taskJSON
	if err := json.Unmarshal(data, &JSON); err != nil {
		return err
	}

	*task = Task{
		URL:      JSON.URL,
		FileName: JSON.FileName,

		size:       JSON.Size,
		stop:       false,
		syncSign:   make(chan bool),
		threads:    JSON.Threads,
		tempFiles:  nil,
		waitThread: sync.WaitGroup{},
		//waitThread: sync.WaitGroup{},
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

	task.tempFiles = make([]*os.File, len(task.threads))
	err := os.Mkdir(task.DirName(), 0644)
	if err != nil && err != os.ErrExist {
		errorLog.Println(err)
	}

	for i := range task.tempFiles {
		task.tempFiles[i], err = os.OpenFile(fmt.Sprintf("./%s/tmp%v", task.DirName(), i),
			os.O_CREATE|os.O_RDWR, 0644)

		if err != nil {
			errorLog.Println(err)
		}

		task.waitThread.Add(1)
		go task.StartThread(i)
	}

	// The downloading ends after this line
	task.waitThread.Wait()
	if task.stop {
		for i := range task.tempFiles {
			task.tempFiles[i].Close()
		}
		task.syncSign <- true
		return
	}

	// Merge temp files to the origin file
	task.MergeTemp()

	// Delete the temp files and temp dir
	for i := range task.tempFiles {
		tempFileName := task.tempFiles[i].Name()
		task.tempFiles[i].Close()
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
func (task *Task) StartThread(i int) {
	defer task.waitThread.Done()
	traceLog.Println(task.URL, task.FileName, "StartThread", i)
	defer traceLog.Println(task.URL, task.FileName, "StartThread", i, "Done")

	var (
		thread = &task.threads[i]
		file   = task.tempFiles[i]
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

	var data = make([]byte, 32*1024)
	for thread.Recv < thread.Size() {
		// Download

		for {
			if task.stop {
				res.Body.Close()
				return
			}

			readCnt, err := res.Body.Read(data)
			if err != nil {
				errorLog.Println(readCnt, err)
				break
			} else if readCnt == 0 {
				break
			}

			//log.Println("Before:", readCnt, thread.Recv)
			WriteCnt, err := file.WriteAt(data[:readCnt], thread.Recv)
			thread.Recv += int64(WriteCnt)
			//log.Println("After:", thread.Recv)
			if err != nil {
				errorLog.Println(WriteCnt, err)
				break
			} else if WriteCnt == 0 {
				break
			}
		}
		res.Body.Close()

		// Continue if downloading has not finished yet
		if thread.Recv < thread.Size() {
			traceLog.Println("Continue", thread.Recv, thread.Size())
			req.Header.Set("Range",
				fmt.Sprintf("bytes=%v-%v", thread.Range[0]+thread.Recv, thread.Range[1]))
			if res, err = client.Do(req); err != nil {
				errorLog.Println(err)
			}
		}
	}
}

func (task *Task) Stop() {
	task.stop = true
	<-task.syncSign
}

// Load a task from a json file
func Load(jsonFile string, task *Task) *Task {
	bytes, err := ioutil.ReadFile(jsonFile)
	if err != nil{
		log.Println(err)
	}
	if task == nil {
		task = &Task{}
	}

	err = json.Unmarshal(bytes, task)
	log.Println(err)
	return task
}

// Save the task as a json file
func (task *Task) Save() {
	if !task.stop {
		task.Stop()
	}

	bytes, err := json.Marshal(task)
	fmt.Println(string(bytes))
	if err != nil{
		log.Println(err)
	}

	jsonFile, err := os.OpenFile(task.FileName+".json", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil{
		log.Println(err)
	}
	jsonFile.Write(bytes)
	jsonFile.Close()
}

// Merge the temp files
func (task *Task) MergeTemp() {
	traceLog.Println(task.URL, task.FileName, "MergeTemp")
	file, _ := os.OpenFile(task.FileName, os.O_CREATE|os.O_APPEND, 0644)
	defer traceLog.Println(task.URL, task.FileName, "MergeTemp", "Done")
	defer file.Close()

	for i := range task.tempFiles {
		task.tempFiles[i].Seek(0, os.SEEK_SET)
		io.Copy(file, task.tempFiles[i])
	}
}

func (task *Task) Close() {
	for i := range task.tempFiles {
		task.tempFiles[i].Close()
	}
}
