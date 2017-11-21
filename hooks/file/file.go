// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package file

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"os/exec"
)

type LoggerInterface interface {
	Init(config string) error
	WriteMsg(msg string, level int) error
	Destroy()
	Flush()
}

// FileLogWriter implements LoggerInterface.
// It writes messages by lines limit, file size limit, or time frequency.
type FileLogWriter struct {
	*log.Logger
	mw *MuxWriter
	// The opened file
	Filename string `json:"filename"`

	Maxlines int `json:"maxlines"`
	maxlines_curlines int

	// Rotate at size
	Maxsize  int `json:"maxsize"`
	maxsize_cursize int

	// Rotate daily
	Daily    bool  `json:"daily"`
	Maxdays  int64 `json:"maxdays"`
	daily_opendate int

	Rotate bool `json:"rotate"`

	startLock sync.Mutex
	Level int `json:"level"`

	// 18/09/2017新增:
	//   日志缓存模块,批量刷新到磁盘;
	//   Add by 164776775@qq.com WeChat:164776775 谷子慧
	AsyncBuffer     bool `json:"asyncbuffer"`// 默认: false.
	BufferSize       int `json:"buffersize"` // 缓冲区大小,默认: 8KB
	buffer     [2][]byte  // 日志在写入文件之前的缓冲区;
	bufchan chan *[]byte  // 当缓冲区写满时,将缓冲地址发往管道,供刷新协程刷入文件;
	bufptr       *[]byte  // 2个缓冲区,指针来回切换,指向空缓冲区;(日志信息通过该指针写入缓冲);
}

// an *os.File writer with locker.
type MuxWriter struct {
	sync.Mutex
	fd *os.File
}

// write to os.File.
func (l *MuxWriter) Write(b []byte) (int, error) {
	l.Lock()
	defer l.Unlock()
	return l.fd.Write(b)
}

// set os.File in writer.
func (l *MuxWriter) SetFd(fd *os.File) {
	if l.fd != nil {
		l.fd.Close()
	}
	l.fd = fd
}

// create a FileLogWriter returning as LoggerInterface.
func NewFileWriter() LoggerInterface {
	w := &FileLogWriter{
		Filename: "",
		Maxlines: 1000000,
		Maxsize:  1 << 28, //256 MB
		Daily:    true,
		Maxdays:  7,
		Rotate:   true,
		Level:    4, // info level.
		AsyncBuffer: false,
		BufferSize:  8 * 1024,
	}
	// use MuxWriter instead direct use os.File for lock write when rotate
	w.mw = new(MuxWriter)

	return w
}

func (w *FileLogWriter) Init(json_config string) error {
	err := json.Unmarshal([]byte(json_config), w)
	if err != nil {
		return err
	}
	if len(w.Filename) == 0 {
		return errors.New("json_config must have filename")
	}
	// filepath.Split() return dir & filename,
	// if w.Filename doesn't contain path, then dir is null-string("").
	dir, _ := filepath.Split(w.Filename)
	if len(dir) != 0 {
		if err = exec.Command("mkdir", "-p", dir).Run(); err != nil {
			return fmt.Errorf("hooks/file: `mkdir -p %s` error:%v", dir, err)
		}
	}
	if err = w.startLogger(); err != nil {
		return err
	}
	if !w.AsyncBuffer {
		// 同步版本:
		// 保留源码,不改动,默认:这里设置了日期和时间在日志中的展示,所以,配置PrintFormat时,可不用重复添加`%d %t`参考:record.go);
		// set MuxWriter as Logger's io.Writer
		w.Logger = log.New(w.mw, "", 0)//log.Ldate|log.Ltime)
	} else {
		// 异步缓存版本: 就不用log模块了,直接从内存刷入磁盘;
		if w.BufferSize < 8*1024 {
			w.BufferSize = 8*1024
		}
		// 经测试发现:
		// 当容量大小一定时,通过append执行拷贝数据,比copy几乎高一倍;
		// 另一好处是: append可以使用len()函数,但copy的len值是一定的,无法更改;
		w.buffer[0] = make([]byte, 0, w.BufferSize)
		w.buffer[1] = make([]byte, 0, w.BufferSize)
		w.bufptr  = &w.buffer[0]
		w.bufchan = make(chan *[]byte)

		// 启动缓冲模块
		go w.startBuffer()
	}

	return nil
}

func (w *FileLogWriter) startBuffer() {
	for {
		select {
		// 1.频繁整块刷新日志入库;
		case ptr := <-w.bufchan: // channel中保存的是已写满的缓冲区的地址;
			{
				if err := w.writeLogger(ptr); err != nil {
					fmt.Printf("Fatal: (new buffer) w.writeLogger(ptr:%p) Err:%s\n", ptr, err)
                    // TODO: retry for 3 times?
				}
			}
		// 2.当超时5秒后,如果仍没有新的日志写满buffer,就将buffer中的日志刷新到磁盘;
		//   如果在这5秒间隔内,程序挂掉了,那没办法,缓冲区中的日志将会丢失;
		case <-time.After(time.Second * 5):
			{
				if len(*w.bufptr) > 0 {
					ptr := w.bufptr

					w.startLock.Lock()
					if w.bufptr == &w.buffer[0] {
						w.bufptr = &w.buffer[1]
					} else {
						w.bufptr = &w.buffer[0]
					}
					w.startLock.Unlock()

					if err := w.writeLogger(ptr); err != nil {
						fmt.Printf("Fatal: after(5s) w.writeLogger(ptr:%p) Err:%s\n", ptr, err)
						// TODO: retry for 3 times?
					}
				}
			}
		}
	}
}

// 刷新到日志文件;
func (w *FileLogWriter) writeLogger(ptr *[]byte) error {
	//start := time.Now()

	bfrlen := len(*ptr)
	if _, err := w.mw.fd.Write(*ptr); err != nil {
		return fmt.Errorf("fd.Write(bufptr:%p buflen:%d) Err:%s", *ptr, bfrlen, err)
	}
	//fmt.Printf("recv:%p bfr:%p len(buffer):%d Cost:%v Now:%v\n", ptr, *ptr, bfrlen, time.Now().Sub(start), time.Now().UnixNano())

	// 先将所有的日志数据刷新到文本,然后再判断并打开新的日志文件;
	//start = time.Now()
	w.docheck(bfrlen)
	//fmt.Printf("END: %p docheck: bfr:%p len(buffer):%d Cost:%v Now:%v\n\n", ptr, *ptr, bfrlen, time.Now().Sub(start), time.Now().UnixNano())

	//start = time.Now()
	// 清空缓存空间;
	*ptr = (*ptr)[0:0] // 情况SLICE就这么简单;
	//fmt.Printf("bfr:%p len(buffer):%d Cost:%v\n", *ptr, bfrlen, time.Now().Sub(start))

	return nil
}

// start file logger. create log file and set to locker-inside file writer.
func (w *FileLogWriter) startLogger() error {
	fd, err := w.createLogFile()
	if err != nil {
		return err
	}
	w.mw.SetFd(fd)
	err = w.initFd()
	if err != nil {
		return err
	}
	return nil
}

func (w *FileLogWriter) docheck(size int) {
	if !w.AsyncBuffer {
        w.startLock.Lock()
        defer w.startLock.Unlock()
	}
	if w.Rotate && ((w.Maxlines > 0 && w.maxlines_curlines >= w.Maxlines) ||
		(w.Maxsize > 0 && w.maxsize_cursize >= w.Maxsize) ||
		(w.Daily && time.Now().Day() != w.daily_opendate)) {
		if err := w.DoRotate(); err != nil {
			fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.Filename, err)
			return
		}
	}
    if !w.AsyncBuffer {
		w.maxlines_curlines++
	}
	w.maxsize_cursize += size
}

// 入口: 所有最上层的log.Debug/Info/Error/Fatal...都会执行到这里;
func (w *FileLogWriter) WriteMsg(msg string, level int) error {
	if level > w.Level {
		return nil
	}
	if !w.AsyncBuffer {
		w.docheck(len(msg))
		w.Logger.Print(msg)
	} else {
        w.startLock.Lock()
		if len(*w.bufptr) + len([]byte(msg)) > w.BufferSize {
			// 1.先将待刷新缓冲区的首地址发送给管道;
			w.bufchan <- w.bufptr
			//fmt.Printf("\n\nSTART: Send %p to channel. Now:%v\n", w.bufptr, time.Now().UnixNano())

			// 2.切换到另外一个缓冲区;
			if w.bufptr == &w.buffer[0] {
				w.bufptr = &w.buffer[1]
			} else {
				w.bufptr = &w.buffer[0]
			}
		}
		*w.bufptr = append(*w.bufptr, msg...)
		w.startLock.Unlock()
	}
	return nil
}

func (w *FileLogWriter) createLogFile() (*os.File, error) {
	fd, err := os.OpenFile(w.Filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	return fd, err
}

func (w *FileLogWriter) initFd() error {
	fd := w.mw.fd
	finfo, err := fd.Stat()
	if err != nil {
		return fmt.Errorf("hooks/file: fd.Stat() err: %s\n", err)
	}
	w.maxsize_cursize = int(finfo.Size())
	w.daily_opendate = time.Now().Day()
	if finfo.Size() > 0 {
		content, err := ioutil.ReadFile(w.Filename)
		if err != nil {
			return err
		}
		w.maxlines_curlines = len(strings.Split(string(content), "\n"))
	} else {
		w.maxlines_curlines = 0
	}
	return nil
}

// DoRotate means it need to write file in new file.
// new file name like xx.log.2013-01-01.2
func (w *FileLogWriter) DoRotate() error {
	_, err := os.Lstat(w.Filename)
	if err == nil { // file exists
		// Find the next available number
		num := 1
		fname := ""
		for ; err == nil && num <= 999; num++ {
			fname = w.Filename + fmt.Sprintf(".%s.%03d", time.Now().Format("2006-01-02"), num)
			_, err = os.Lstat(fname)
		}
		// return error if the last file checked still existed
		if err == nil {
			return fmt.Errorf("Rotate: Cannot find free log number to rename %s\n", w.Filename)
		}

		// block Logger's io.Writer
		w.mw.Lock()
		defer w.mw.Unlock()

		fd := w.mw.fd
		fd.Close()

		// close fd before rename
		// Rename the file to its newfound home
		err = os.Rename(w.Filename, fname)
		if err != nil {
			return fmt.Errorf("Rotate: %s\n", err)
		}

		// re-start logger
		err = w.startLogger()
		if err != nil {
			return fmt.Errorf("Rotate StartLogger: %s\n", err)
		}

		go w.deleteOldLog()
	}

	return nil
}

func (w *FileLogWriter) deleteOldLog() {
	dir := filepath.Dir(w.Filename)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) (returnErr error) {
		defer func() {
			if r := recover(); r != nil {
				returnErr = fmt.Errorf("hooks/file: Unable to delete old log '%s', error: %+v", path, r)
				fmt.Println(returnErr)
			}
		}()

		if !info.IsDir() && info.ModTime().Unix() < (time.Now().Unix()-60*60*24*w.Maxdays) {
			if strings.HasPrefix(filepath.Base(path), filepath.Base(w.Filename)) {
				os.Remove(path)
			}
		}
		return
	})
}

// destroy file logger, close file writer.
func (w *FileLogWriter) Destroy() {
	w.mw.fd.Close()
}

// flush file logger.
// there are no buffering messages in file logger in memory.
// flush file means sync file from disk.
func (w *FileLogWriter) Flush() {
	w.mw.fd.Sync()
}
