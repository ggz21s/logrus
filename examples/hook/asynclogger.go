

package main

import (
    "fmt"
    "io/ioutil"
    "github.com/logrus"
    "github.com/logrus/hooks/file"
    "sync"
    "time"
)

var GLog *logrus.Logger

type LogConfig struct {
    FileName    string  `json:"filename"`
    MaxLines    int     `json:"maxlines"`
    MaxSize     int     `json:"maxsize"`
    Daliy       bool    `json:"daliy"`
    MaxDays     int     `json:"maxdays"`
    Rotate      bool    `json:"rotate"`
    Level       int     `json:"level"`
    PrintFormat string  `json:"PrintFormat"`
    PrintToTTY  bool    `json:"PrintToTTY"`
    AsyncBuffer bool    `json:"AsyncBuffer"`
    BufferSize  int     `json:"BufferSize"`
}

func init() {
    // 创建一个不打印到TTY的空日志对象);
    GLog = logrus.NewLogger(
        nil,
        nil,
        logrus.DebugLevel,
        //os.Stdout,
        //&logrus.LogFormatter{
        //    PrintFormat: "[%T %s] [%L] %M",
        //    ForceColors: true,
        //},
        //logrus.InfoLevel,
    )
}

func main() {
    json, err := ioutil.ReadFile("asynclogger.json")
    if err != nil {
        fmt.Printf("Err: %s\n", err)
        return
    }
    // PrintFormat只会在GLog.Hooks.Add()时被修改;
    // 另外,同步模式的日志记录底层调用的是log模块,所以会重复打印 日期和时间;所以,对于选择非异步刷新模式记录日志时,可以不选 %d %T
    GLog.Hooks.Add(file.NewHook(string(json), "[%d %T %s] [%L] %M"))
    //GLog.Hooks.Add(file.NewHook(string(json), "%s [%L] %M")) // for 同步日志;

    golen := 1000
    var wg sync.WaitGroup
    wg.Add(golen)
    start := time.Now()
    for i := 0; i < golen; i++ {
        go func(i int) {
            for j := 0; j < 500; j++ {
                GLog.Debugf("Go<%d> Debug ===========================================%d===========================================", i, j)
                GLog.Infof ("Go<%d> Infof ===========================================%d===========================================", i, j)
                GLog.Errorf("Go<%d> Error ===========================================%d===========================================", i, j)
            }
            wg.Done()
        }(i)
    }
    wg.Wait()
    fmt.Printf("TotalCost:%v\n", time.Now().Sub(start))
    time.Sleep(5*time.Second)
}
