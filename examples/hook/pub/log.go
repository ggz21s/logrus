package pub

import (
    "github.com/logrus"
    "github.com/logrus/hooks/file"
    //"os"
    "os"
)

var GLog *logrus.Logger

func init() {
    GLog = logrus.NewLogger(
        os.Stdout,
        //nil,
        //nil,
        &logrus.LogFormatter{
            PrintFormat: "[%T %s] [%L] %M",
            ForceColors: true,
        },
        logrus.InfoLevel,
    )

    // 如果给Log添加hooks(比如文件类型的Hook),那么,信息不但会打印到终端,
    // 还会写入到文件(如果是其他hook,则按Hook的实现执行操作;
    config_json := `{
    	"filename": "/tmp/logrus.log",
    	"maxlines": 200,
    	"maxsize" : 10000000,
    	"daily"   : true,
    	"maxdays" : 15,
    	"rotate"  : true,
    	"level"   : 5
     }`
    GLog.Hooks.Add(file.NewHook(config_json, "[%s] [%L] %M"))
    GLog.Out = nil
    GLog.Formatter = nil
}
