// log_formatter.go:
//   Added by richard.liu@nio.com <谷子慧> 2017/07/05.
//   Print to log like down:
//   2017/07/05 10:56:30 [hook_standard.go:52] [WARN] The group's number increased tremendously!
//   2017/07/05 10:56:30 [hook_standard.go:57] [FATAL] The ice breaks!
package logrus

import (
    "bytes"
    "fmt"
    "runtime"
)

const LOG_TIME_FORMAT = "2006-01-02 15:04:05" // Never modify this special time format.

type LogFormatter struct {
    // PrintFormat
    PrintFormat string

    // Set to true to bypass checking for a TTY before outputting colors.
    ForceColors bool
}

// Format as:
// For file: [2017-07-03 15:09:55.666 main/main.go:31] [WARN] Message Info.
// For Stdio: with colors.
// LogFormatter只针对打印到终端起作用,写入日志部分由 logrus.record.go中相关方法执行;
func (f *LogFormatter) Format(entry *Entry) ([]byte, error) {

    var keys []string = make([]string, 0, len(entry.Data))
    for k := range entry.Data {
        keys = append(keys, k)
    }

    bfr := &bytes.Buffer{}

    PrefixFieldClashes(entry.Data)

    if f.PrintFormat == "" {
        f.PrintFormat = "[%T %s] [%L] %M"
    }

    isColorTerminal := isTerminal && (runtime.GOOS != "windows")
    isColored := (f.ForceColors && isColorTerminal)

    var levelColor int
    switch entry.Level {
    case DebugLevel:
        levelColor = black
    case InfoLevel:
        levelColor = green
    case WarnLevel:
        levelColor = yellow
    case ErrorLevel, FatalLevel, PanicLevel:
        levelColor = red
    default:
        levelColor = nocolor
    }
    record := Prepare(entry.Level, entry.Message, entry.Logger.PkgPath)
    des := NewPrintFormat(f.PrintFormat).Format(record)
    //fmt.Printf("entry:%v record:%v des:%s printFormat:<%s>\n", entry, record, des, f.PrintFormat)
    if isColored {
        fmt.Fprintf(bfr, "\x1b[%dm%s\x1b[0m", levelColor, des)
        for _, k := range keys {
            v := entry.Data[k]
            fmt.Fprintf(bfr, " \x1b[%dm%s\x1b[0m=%v", levelColor, k, v)
        }
    } else {
        bfr.WriteString(des)
        for _, k := range keys {
            v := entry.Data[k]
            fmt.Fprintf(bfr, "%v=%v", k, v)
        }
    }
    if len(keys) > 0 {
        bfr.WriteString("\n")
    }

    return bfr.Bytes(), nil
}