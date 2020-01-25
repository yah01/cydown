package cydown

import (
	"net/http"
)

type Range = [2]int64
type Thread struct {
	Index int
	Range Range
	Recv  int64
}

// Get the size of thread
func (thread *Thread) Size() int64 {
	return thread.Range[1] - thread.Range[0] + 1
}

// Create a http client for thread.
func (thread *Thread) NewClient() *http.Client {
	traceLog.Println("NewClient")
	defer traceLog.Println("NewClient Done")

	var transport = &http.Transport{}
	if globalProxy != nil {
		transport.Proxy = globalProxy
	}

	return &http.Client{Transport: transport}
}
