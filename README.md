# cydown

## Install and Import
Install cydown:
```shell script
go get -u github.com/yah01/cydown
```
And import cydown into your program:
```shell script
import "github.com/yah01/cydown"
```

## Quick Start
Download a file at *url*, and save it as *fileName*:
```go
cydown.Download(url,filename)
```
cydown could get the file name from URL, if you wanna use the name from url, just set *filename* = ""

u could set proxy before downloading, for example:
```go
cydown.SetGlobalProxy(cydown.ProxyFn{
    return url.Parse("http://127.0.0.1:1080")
})
```

