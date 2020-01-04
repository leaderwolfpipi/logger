package logger

// 单元测试和基准测试文件

import (
	// "log"
	"testing"

	"github.com/whitewolfpipi/logger"
)

// go test -test.bench=".*" -run=none  -test.benchmem  -benchtime=3s
func BenchmarkLogger(b *testing.B) {
	// 复位计时器
	b.ResetTimer()

	// 初始化变量
	// var l *logger.Logger = logger.NewLogger()
	var l *logger.RotateFileLogger = logger.NewRotateFileLogger("./")
	l.SetCacheSwitch(true)
	l.SetQueueSize(200000)
	l.SetCacheDuration(100)
	l.SetCacheCap(64)
	l.Start()
	//	var s = []string{
	//		"200",
	//		"请求成功",
	//		"0.55ms",
	//		"GET",
	//		"/hello",
	//	}

	s := "200 | ok! | 1.053µs | GET  /PONG "

	// 循环执行测试代码
	for i := 0; i < b.N; i++ {
		// 这里书写测试代码
		// log.Println(ss)
		l.Info(s)
		// lf.Info(s)
	}

}
