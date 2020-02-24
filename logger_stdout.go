package logger

import (
	// "bytes"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

const (
	// 日志类型常量
	DEBUG    = LogType(0)
	INFO     = LogType(1)
	NOTICE   = LogType(2)
	WARN     = LogType(3)
	ERROR    = LogType(4)
	CRITICAL = LogType(5)
	FATAL    = LogType(6)

	// 同步状态常量
	statusInit  syncStatus = iota // 初始状态
	statusDoing                   // 同步中
	statusDone                    // 同步已经完成
)

/*
   Logger 日志

   l := NewLogger()
   l.Info("hello")
   l.Warn(1)

   输入格式可以通过 SetLoggerFormat 设置。默认输出格式定义在 见Logger的DefaultLogFormatFunc
   可以通过 SetLogLevel 设置输出等级。
*/
type (
	// 日志对象定义
	Logger struct {
		sync.RWMutex
		mu            sync.Mutex
		out           io.Writer
		logFormatFunc FormatFunc
		logLevel      LogType
		status        syncStatus  // 日志状态
		queue         chan string // 通过实现消息队列
		queueSize     int         // 队列通道大小
		// 缓存控制块
		cache struct {
			use      bool          // 是否使用缓存
			data     []string      // 缓存数据
			mutex    sync.Mutex    // 写cache时的互斥锁
			cacheCap int           // 缓存容量默认64
			duration time.Duration // 同步数据到文件的周期，默认为100毫秒
		}
	}

	// log同步的状态
	syncStatus int

	// 日志类型
	LogType int
)

var (
	// 定义数据段颜色
	dataColor map[string]string = map[string]string{
		"r": "41;97", // 深红
		"g": "42;97", // 绿色
		"y": "43;97", // 土黄
		"b": "44;97", // 深蓝
	}

	// 定义日志样式
	logTypeStrings = func() []string {
		// log 类型对应的 名称字符串，用于输出，所以统一了长度，故 DEBUG 为 "DEBUG..." 和 "CRITICAL"等长
		types := []string{"DEBUG", "INFO", "NOTICE", "WARN", "ERROR", "CRITICAL", "FATAL"}
		maxTypeLen := 0
		for _, t := range types {
			if len(t) > maxTypeLen {
				maxTypeLen = len(t)
			}
		}
		for index, t := range types {
			typeLen := len(t)
			if typeLen < maxTypeLen {
				types[index] += strings.Repeat(" ", maxTypeLen-typeLen)
			}
		}
		return types
	}()

	// 定义样色品类
	// 41;97 底色深红, 加亮白色;
	// 42;97 底色绿字, 加亮白色;
	// 43;97 底色黄色, 加亮白色;
	// 45;97 底色紫色, 加亮白色;
	logTypesColors = []string{"45;97", "42;97", "43;97", "43;97", "41;97", "41;97", "41;97"}

	// 声明接口实现者
	_ ILogger = &Logger{}
)

/*
 * 创建Logger对象
 */
func NewLogger() *Logger {
	// 实例化日志对象并初始化参数
	logger := &Logger{}

	// 设置日志的默认参数
	logger.out = os.Stdout      // 设置输出
	logger.cache.use = true     // 缓存开关
	logger.cache.duration = 100 // 缓存同步周期
	logger.cache.cacheCap = 128 // 缓存容量
	logger.queueSize = 100000   // 默认队列大小1000000
	logger.logLevel = DEBUG     // 设置默认级别
	logger.cache.data = make([]string, 0, logger.cache.cacheCap)
	logger.logFormatFunc = logger.DefaultLogFormatFunc

	return logger
}

// 获取日志类型串
func GetLogTypeString(t LogType) string {
	return logTypeStrings[t]
}

// 启动日志记录器
func (l *Logger) Start() {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 关闭缓存
	if !l.cache.use {
		// 初始化通道
		l.queue = make(chan string, l.queueSize)

		// 异步写
		go func() {
			// 启动监听通道goroutine
			for {
				select {
				case msg, ok := <-l.queue:
					// 逐个写入终端
					if ok {
						_, err := io.WriteString(l.out, msg)
						if err != nil {
							// 重试
							_, err := io.WriteString(l.out, msg)
							if err != nil {
								panic(err)
							}
						}
					}
				}
			}
		}()

		return
	}

	// 使用缓存
	timer := time.NewTicker(time.Millisecond * l.cache.duration)

	go func() {
		// 实现异步写日志
		for {
			select {
			case <-timer.C:
				//now := nowFunc()
				l.RLock()
				if l.status != statusDoing {
					// 单开goroutine将当前缓存中的日志刷出
					go l.flush()
				}
				l.RUnlock()
			}
		}
	}()

}

// 设置cache开关
func (l *Logger) SetCacheSwitch(use bool) {
	l.cache.use = use
}

// 设置cache周期
func (l *Logger) SetCacheDuration(duration time.Duration) {
	l.cache.duration = duration
}

// 设置队列容量
func (l *Logger) SetQueueSize(size int) {
	l.queueSize = size
}

// 设置cache容量
func (l *Logger) SetCacheCap(cap int) {
	l.cache.cacheCap = cap
}

// 设置日志级别
func (l *Logger) SetLogLevel(logType LogType) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logLevel = logType
}

// 获取日志级别
func (l *Logger) GetLogLevel() LogType {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.logLevel
}

// 设置格式化log输出函数
// 函数返回 format 和 对应格式 []interface{}
func (l *Logger) SetLoggerFormat(formatFunc FormatFunc) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logFormatFunc = formatFunc
}

// 输出信息
func (l *Logger) Debug(i interface{}) {
	l.log(DEBUG, i)
}

func (l *Logger) Info(i interface{}) {
	l.log(INFO, i)
}

func (l *Logger) Notice(i interface{}) {
	l.log(NOTICE, i)
}

func (l *Logger) Warn(i interface{}) {
	l.log(WARN, i)
}

func (l *Logger) Error(i interface{}) {
	l.log(ERROR, i)
}

func (l *Logger) Critical(i interface{}) {
	l.log(CRITICAL, i)
}

func (l *Logger) Fatal(i interface{}) {
	l.log(FATAL, i)
}

func (l *Logger) DefaultLogFormatFunc(logType LogType, i interface{}) (string, []interface{}, bool) {
	// 异常捕获
	defer func() {
		e := recover()
		if e != nil {
			panic(debug.Stack())
		}
	}()

	// 计算日期format
	layout := "2006/01/02 - 15:04:05.0000"
	formatTime := time.Now().Format(layout)
	if len(formatTime) != len(layout) {
		// 可能出现结尾是0被省略如：2006/01/02 - 15:04:05.9 补足成 2006/01/02 - 15:04:05.9000
		if len(formatTime) == 21 {
			formatTime += "."
		}
		formatTime += ".000"[4-(len(layout)-len(formatTime)) : 4]
	}

	// 计算数据format
	// format := ""
	values := []interface{}{}
	var b strings.Builder // 提升性能
	b.Grow(32)
	if iSli, ok := i.([]string); ok {
		// 切片
		l := len(iSli)
		b.WriteString("[\033[")
		b.WriteString(logTypesColors[logType])
		b.WriteString("m%s\033[0m] %s | ")
		// format = "[\033[" + logTypesColors[logType] + "m%s\033[0m] %s | "
		values = make([]interface{}, l+2)
		values[0] = logTypeStrings[logType]
		values[1] = formatTime
		for j := 0; j < l; j++ {
			ls := len(iSli[j])
			tj := ""
			if ls >= 2 {
				// 截取最后两个字符
				// 颜色标志：-g 绿色; -r 红色; -b 蓝色
				color := iSli[j][ls-2:]
				if color[0] == '-' && (color[1] == 'g' || color[1] == 'b' || color[1] == 'r' || color[1] == 'y') {
					tj = iSli[j][0 : ls-2] // 去除颜色标志
					b.WriteString("\033[")
					b.WriteString(dataColor[string(color[1])])
					b.WriteString("m%s\033[0m | ")
					// format += "\033[" + dataColor[string(color[1])] + "m%s\033[0m | "
				} else {
					tj = iSli[j]
					b.WriteString("%s | ")
					// format += "%s | "
				}
			} else {
				tj = iSli[j]
				b.WriteString("%s | ")
				// format += "%s | "
			}
			// 计算输出值
			values[j+2] = tj
		}
		b.WriteString("\n")
		// format += "\n"
	} else if iStr, ok := i.(string); ok {
		// 文本
		b.WriteString("[\033[")
		b.WriteString(logTypesColors[logType])
		b.WriteString("m%s\033[0m] %s | %s | \n")
		// format = "[\033[" + logTypesColors[logType] + "m%s\033[0m] %s | %s | \n"
		// 计算输出值
		values = make([]interface{}, 3)
		values[0] = logTypeStrings[logType]
		values[1] = formatTime
		values[2] = iStr
	}

	// 返回格式/值
	return b.String(), values, true
}

func (l *Logger) log(logType LogType, i interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logLevel > logType {
		return
	}

	format, data, isLog := l.logFormatFunc(logType, i)
	if !isLog {
		return
	}

	var err error
	if l.cache.use {
		// 使用缓存
		l.cache.data = append(l.cache.data, fmt.Sprintf(string(format), data...))
	} else {
		// 追加进队列
		l.queue <- fmt.Sprintf(string(format), data...)
		// _, err = fmt.Fprintf(l.out, string(format), data...)
	}
	if err != nil {
		panic(err)
	}
}

// 将当前缓存中的日志刷出
func (l *Logger) flush() error {
	l.status = statusDoing
	defer func() {
		l.status = statusDone
	}()

	// 获取缓存数据
	l.cache.mutex.Lock()
	cache := l.cache.data
	l.cache.data = l.cache.data[0:0] // 极大的节省空间分配减轻垃圾回收压力
	// l.cache.data = make([]string, 0, l.cache.cacheCap)
	l.cache.mutex.Unlock()

	if len(cache) == 0 {
		return nil
	}

	_, err := io.WriteString(l.out, strings.Join(cache, ""))
	if err != nil {
		// 重试
		_, err := io.WriteString(l.out, strings.Join(cache, ""))
		if err != nil {
			panic(err)
		}
	}

	return nil
}

// 兼容gorm日志实现Print
func (l *Logger) Print(v ...interface{}) {
	// @Todo...
	panic("method not implement")
}
