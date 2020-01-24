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
cydown could get the file name from URL, if you wanna use the name from url:
```go
cydown.Download(url,"")
```
Download() is non-blocking, so you must call:
```go
cydown.Wait()
```
at the end of your program, to make sure the all downloads end before the program quiting.

You could set proxy before downloading:
```go
cydown.SetGlobalProxy(cydown.ProxyFn{
    return url.Parse("http://127.0.0.1:1080")
})
```
or just:
```go
cydown.UseGlobalLocalProxy("1080")
```

## Stop, Save, Load, and Restart
Stop a task after it started downloading:
```go
task.Stop()
```

and save it:
```go
task.Save()
```
This will create a JSON file named *fileName.json*, and you could load the JSON file:
```go
Load(JSONfileName,&task)
```
or:
```go
task := Load(JSONfileName,nil)
```
the type of the latter is **Task*.

Restart a task is simple, just call Download() again:
```go
task.Download("")
```
arg fileName is empty string, so that keep the name which you used before.
