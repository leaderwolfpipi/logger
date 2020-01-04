logger日志组件

## 使用案例

#### 优化性能的措施
> 1.使用批量写入100ms写一次
> 2.使用异步记录日志
> 3.减少fmt家族函数的使用
> 4.使用缓存记录日志

```code
package main

import (
	// "fmt"
	"github.com/pxlh007/logger"
)

func main() {
	var l *logger.Logger = logger.NewLogger()
	l.Info("200 | ok! | 1.052µs | POST /PING ")
	l.Info("200 | ok! | 1.053µs | GET  /PONG ")

	var s = []string{
		"200-g",
		"请求成功",
		"0.55ms",
		"GET-y",
		"/hello",
	}
	l.Info(s)

	var serror = []string{
		"404-r",
		"NOT FOUND!",
		"2.556ms",
		"POST-b",
		"/",
	}
	l.Error(serror)

	l.Debug("输出debug信息！")

	// l.Error("This is an error")
	// l.Warn("This is warning!")
	// l.Debug("This is debugging!")
	// l.Fatal("This is Fatal!")
	// fmt.Println(l)

	// 文件测试
	var lf *logger.RotateFileLogger = logger.NewRotateFileLogger("./")
	lf.Info("记录进文件测试...")
	lf.Error("记录错误信息...")

	var sf = []string{
		"200",
		"请求成功",
		"0.55ms",
		"GET",
		"/hello",
	}
	lf.Info(sf)

	var sferror = []string{
		"404",
		"NOT FOUND!",
		"2.556ms",
		"POST",
		"/",
	}
	lf.Error(sferror)

}

```

#### 基准测试结果

```
JonahdeMacBook-Pro:logger jonah$ go test -test.bench=".*" -run=none  -test.benchmem  -benchtime=1s
goos: darwin
goarch: amd64
pkg: whitewolfpipi/logger/logger
BenchmarkLogger-8        1000000              1157 ns/op             438 B/op          8 allocs/op
PASS
ok      whitewolfpipi/logger/logger     1.186s
```
 
