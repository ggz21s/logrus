package main

import (
    "github.com/logrus"
    "github.com/logrus/hooks/file"
    //"os"
    "os"
)

var log *logrus.Logger

func init() {
    log = logrus.NewLogger(
        os.Stderr,
        //nil,
        new(logrus.LogFormatter),
        //&logrus.LogFormatter{
        //    PrintFormat: "[%T %s] [%L] %M",
        //    ForceColors: true,
        //},
        logrus.DebugLevel,
    )

    // 如果给Log添加hooks(比如文件类型的Hook),那么,信息不但会打印到终端,
    // 还会写入到文件(如果是其他hook,则按Hook的实现执行操作;
    config_json := `{
    	"filename": "logs/sample.log",
    	"maxlines": 50,
    	"maxsize" : 10000000,
    	"daily"   : true,
    	"maxdays" : 15,
    	"rotate"  : true,
    	"level"   : 5
     }`
    log.Hooks.Add(file.NewHook(config_json, "[%s] [%L] %M"))
}

func main() {

    test()

    log.Debug("Hello %s", "World, I'm richard.liu@nio.com")

    log.Errorf("test-Error:%s", "How do i do?")

    log.WithFields(logrus.Fields{
        "animal": "walrus",
        "size":   10,
    }).Info("A group of walrus emerges from the ocean")

    log.WithFields(logrus.Fields{
        "omg":    true,
        "number": 122,
    }).Warn("The group's number increased tremendously!")

    log.WithFields(logrus.Fields{
        "omg":    true,
        "number": 100,
    }).Fatal("The ice breaks!")
}

func test() {
    log.Errorf("test: error: %s.", "you are 1900.")
}