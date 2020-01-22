package cydown

import (
	"net/http"
)

type Range = [2]int64
type Thread struct {
	Index int
	Range Range
	Recv  int64
	proxy ProxyFn
}

func InitThreads(threads []Thread, size int64) {
	traceLog.Println("InitThreads", size)
	defer traceLog.Println("InitThreads", size, "Done")
	partSize := size / int64(len(threads))
	for i := range threads {
		// Set Range of downloading thread
		threads[i] = Thread{
			Range: Range{int64(i) * partSize, int64((i+1))*partSize - 1},
		}
		if i == len(threads)-1 {
			threads[i].Range = Range{int64(i) * partSize, size - 1}
		}
	}
}

func (thread *Thread) Size() int64 {
	return thread.Range[1] - thread.Range[0] + 1
}

func (thread *Thread) NewClient() *http.Client {
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
