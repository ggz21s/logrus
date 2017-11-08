package file

import (
    "fmt"
	"github.com/logrus"
)

// level: see, github.com/logrus/logrus.go const's XxxLevel
func NewHook(jsonConfig, printFormat string) *FileHook {

    w := NewFileWriter()

    if err := w.Init(jsonConfig); err != nil {
        fmt.Printf("hooks: FileWriter.Init(%s) error:%v\n", jsonConfig, err)
        panic(err)
    }

	return &FileHook{
		W: w,
		PrintFormat: printFormat,
	}
}

type FileHook struct {
	PrintFormat string
	W LoggerInterface
}

func (hook *FileHook) Fire(entry *logrus.Entry) (err error) {

    // 使用 logrus.record.go 中的相关API,
    // 跟log_formatter.go<A>没啥关系,A只针对打印到终端有用;
    record  := logrus.Prepare(entry.Level, entry.Message, entry.Logger.PkgPath)
    message := logrus.NewPrintFormat(hook.PrintFormat).Format(record)

    return hook.W.WriteMsg(message, int(entry.Level))
}

func (hook *FileHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}