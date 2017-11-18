package logrus

import (
    "bytes"
    "fmt"
    "runtime"
    "strings"
    "time"
    "regexp"
)

const (
    FileDelimiterRecord = "logrus/record.go"
    FileDelimiterLogger = "logrus/logger.go"
)

func Prepare(level Level, msg, pkgPath string) *LogRecord {
    depth, offset := 0, 0
    dest  := FileDelimiterLogger

    if pkgPath != "" {
        dest = pkgPath
    }
    for i, j := 0, 0; ; i++ {
        _, file, _, _ := runtime.Caller(i)
        if strings.Contains(file, FileDelimiterRecord) {
            j += 1
            if j == 2 {
                // offset: 文件record.go,第一次和第二次出现之间的文件间隔个数
                // (如果为0,说明logrus包外层没进一步包装);
                offset = i
            }
        }
        if strings.Contains(file, dest) {
            // i+1: 代表下一个为目标文件名及行号; -offset: 获取真正的文件嵌套深度;
            depth = i + 1 - offset
            break
        }
    }

    funcPath, packagePath := "_", "_"
    pc, file, line, _ := runtime.Caller(depth)

    me := runtime.FuncForPC(pc)
    if me != nil {
        funcPath = me.Name()
        packagePath = splitPackage(funcPath)
    }

    return &LogRecord{
        Level:       level,
        Timestamp:   time.Now(),
        SourceFile:  file,
        SourceLine:  line,
        Message:     msg,
        FuncPath:    funcPath,
        PackagePath: packagePath,
    }
}

// Split a full package.function into just the package component.
func splitPackage(pkg string) string {
    split := strings.Split(pkg, ".")
    return strings.Join(split[:len(split)-1], "")
}

// This packs up all the message data and metadata. This structure
// will be passed to the LogFormatter
type LogRecord struct {
    Level       Level
    Timestamp   time.Time
    SourceFile  string
    SourceLine  int
    Message     string
    FuncPath    string
    PackagePath string
}

// Format a log message before writing
type PrintFormatter interface {
    Format(rec *LogRecord) string
}

var prefixRegexp = regexp.MustCompile(`^[\-+]?[0-9]+`)

type PrintFormat struct {
    format        string
    formatCompile string
    formatDynamic []byte
}

// Format codes:
//   %T - Time: 17:24:05.333 HH:MM:SS.ms
//   %t - Time: 17:24:05 HH:MM:SS
//   %D - Date: 2011-12-25 yyyy-mm-dd
//   %d - Date: 2011/12/25
//   %L - Level (FNST, FINE, DEBG, TRAC, WARN, EROR, CRIT)
//   %S - Source: full runtime.Caller line
//   %s - Short Source: just file and line number
//   %x - Extra Short Source: just file without .go suffix
//   %M - Message
//   %% - Percent sign
// 	 %P - Caller Path: package path + calling function name
// 	 %p - Caller Path: package path
// the string number prefixes are allowed e.g.: %10s will pad the source field to 10 spaces
func NewPrintFormat(format string) *PrintFormat {
    pf := new(PrintFormat)
    pf.format = format
    pf.formatDynamic = make([]byte, 0, 9)            // there are only 9 format codes so this is probably enough
    pf.formatCompile = string(pf.compileForLevel(0)) // TODO figure out if I really want to cache each level
    return pf
}

// this precompiles a sprintf format string for later use
// it looks nasty but it should only be run once at config time
func (pf *PrintFormat) compileForLevel(level int) []byte {
    parts := bytes.Split([]byte(pf.format), []byte{'%'})
    // check for a number formatter
    var sprintfFmt []byte
    for i, part := range parts {
        if i == 0 {
            sprintfFmt = append(sprintfFmt, part...)
            continue
        }
        fmt_str := part
        var num []byte
        if num = prefixRegexp.Find(part); num != nil {
            fmt_str = part[len(num):]
        }

        //fmt.Printf("%d A:<%s> N:<%s> P:<%s>\n", i, string(fmt_str), string(num), string(part))
        switch fmt_str[0] {
        case 'T':
            if num != nil {
                sprintfFmt = append(sprintfFmt, '%')
                sprintfFmt = append(sprintfFmt, num...)
                sprintfFmt = append(sprintfFmt, 's')
                pf.formatDynamic = append(pf.formatDynamic, 'e')
            }
            sprintfFmt = append(sprintfFmt, []byte("%02d:%02d:%02d.%03d")...)
            sprintfFmt = append(sprintfFmt, fmt_str[1:]...)
            pf.formatDynamic = append(pf.formatDynamic, 'T')
        case 't':
            if num != nil {
                sprintfFmt = append(sprintfFmt, '%')
                sprintfFmt = append(sprintfFmt, num...)
                sprintfFmt = append(sprintfFmt, 's')
                pf.formatDynamic = append(pf.formatDynamic, 'e')
            }
            sprintfFmt = append(sprintfFmt, []byte("%02d:%02d:%02d")...)
            sprintfFmt = append(sprintfFmt, fmt_str[1:]...)
            pf.formatDynamic = append(pf.formatDynamic, 't')
        case 'D':
            if num != nil {
                sprintfFmt = append(sprintfFmt, '%')
                sprintfFmt = append(sprintfFmt, num...)
                sprintfFmt = append(sprintfFmt, 's')
                pf.formatDynamic = append(pf.formatDynamic, 'e')
            }
            sprintfFmt = append(sprintfFmt, []byte("%d-%02d-%02d")...)
            sprintfFmt = append(sprintfFmt, fmt_str[1:]...)
            pf.formatDynamic = append(pf.formatDynamic, 'D')
        case 'd':
            if num != nil {
                sprintfFmt = append(sprintfFmt, '%')
                sprintfFmt = append(sprintfFmt, num...)
                sprintfFmt = append(sprintfFmt, 's')
                pf.formatDynamic = append(pf.formatDynamic, 'e')
            }
            sprintfFmt = append(sprintfFmt, []byte("%d/%02d/%02d")...)
            sprintfFmt = append(sprintfFmt, fmt_str[1:]...)
            pf.formatDynamic = append(pf.formatDynamic, 'd')
        case 'L':
            sprintfFmt = append(sprintfFmt, '%')
            if num != nil {
                sprintfFmt = append(sprintfFmt, num...)
            }
            sprintfFmt = append(sprintfFmt, 's')
            sprintfFmt = append(sprintfFmt, fmt_str[1:]...)
            pf.formatDynamic = append(pf.formatDynamic, 'L')
        case 'S':
            sprintfFmt = append(sprintfFmt, '%')
            if num != nil {
                sprintfFmt = append(sprintfFmt, num...)
            }
            sprintfFmt = append(sprintfFmt, 's')
            sprintfFmt = append(sprintfFmt, fmt_str[1:]...)
            pf.formatDynamic = append(pf.formatDynamic, 'S')
        case 's':
            sprintfFmt = append(sprintfFmt, '%')
            if num != nil {
                sprintfFmt = append(sprintfFmt, num...)
            }
            sprintfFmt = append(sprintfFmt, 's')
            sprintfFmt = append(sprintfFmt, fmt_str[1:]...)
            pf.formatDynamic = append(pf.formatDynamic, 's')
        case 'x':
            sprintfFmt = append(sprintfFmt, '%')
            if num != nil {
                sprintfFmt = append(sprintfFmt, num...)
            }
            sprintfFmt = append(sprintfFmt, 's')
            sprintfFmt = append(sprintfFmt, fmt_str[1:]...)
            pf.formatDynamic = append(pf.formatDynamic, 'x')
        case 'M':
            sprintfFmt = append(sprintfFmt, '%')
            if num != nil {
                sprintfFmt = append(sprintfFmt, num...)
            }
            sprintfFmt = append(sprintfFmt, 's')
            sprintfFmt = append(sprintfFmt, fmt_str[1:]...)
            pf.formatDynamic = append(pf.formatDynamic, 'M')
        case '%':
            sprintfFmt = append(sprintfFmt, '%')
            sprintfFmt = append(sprintfFmt, fmt_str...)
        case 'P':
            sprintfFmt = append(sprintfFmt, '%')
            if num != nil {
                sprintfFmt = append(sprintfFmt, num...)
            }
            sprintfFmt = append(sprintfFmt, 's')
            sprintfFmt = append(sprintfFmt, fmt_str[1:]...)
            pf.formatDynamic = append(pf.formatDynamic, 'P')
        case 'p':
            sprintfFmt = append(sprintfFmt, '%')
            if num != nil {
                sprintfFmt = append(sprintfFmt, num...)
            }
            sprintfFmt = append(sprintfFmt, 's')
            sprintfFmt = append(sprintfFmt, fmt_str[1:]...)
            pf.formatDynamic = append(pf.formatDynamic, 'p')
        default:
            sprintfFmt = append(sprintfFmt, fmt_str...)
        } // end switch

    } // end for
    sprintfFmt = append(sprintfFmt, '\n')
    //fmt.Printf("compileForLevel: %s", string(sprintfFmt))
    return sprintfFmt
}

// LogFormatter interface
func (pf *PrintFormat) Format(rec *LogRecord) string {
    data := pf.getDynamic(rec)
    //fmt.Printf("%v", data)
    return fmt.Sprintf(pf.formatCompile, data...)
}

func (pf *PrintFormat) getDynamic(rec *LogRecord) []interface{} {
    tm := rec.Timestamp
    ret := make([]interface{}, 0, 10)
    for _, dyn := range pf.formatDynamic {
        switch dyn {
        case 'e':
            ret = append(ret, "")
        case 'T':
            ret = append(ret, parseTimeMs(tm)...)
        case 't':
            ret = append(ret, parseTime(tm)...)
        case 'D', 'd':
            ret = append(ret, parseDate(tm)...)
        case 'L':
            ret = append(ret, rec.Level.String())
        case 'S':
            ret = append(ret, parseSourceLong(rec.SourceFile, rec.SourceLine))
        case 's':
            ret = append(ret, parseSourceShort(rec.SourceFile, rec.SourceLine))
        case 'x':
            ret = append(ret, parseSourceXShort(rec.SourceFile))
        case 'M':
            ret = append(ret, rec.Message)
        case 'P':
            ret = append(ret, rec.FuncPath)
        case 'p':
            ret = append(ret, rec.PackagePath)
        }
    }
    return ret
}

func parseSourceLong(file string, line int) string {
    return fmt.Sprintf("%s:%d", file, line)
}

func parseSourceShort(file string, line int) string {
    just_file := file[strings.LastIndex(file, "/")+1:]
    return fmt.Sprintf("%s:%d", just_file, line)
}

func parseSourceXShort(file string) string {
    return file[strings.LastIndex(file, "/")+1 : (len(file) - 3)]
}

func parseDate(t time.Time) []interface{} {
    return []interface{}{t.Year(), t.Month(), t.Day()}
}

func parseTime(t time.Time) []interface{} {
    return []interface{}{t.Hour(), t.Minute(), t.Second()}
}

func parseTimeMs(t time.Time) []interface{} {
    return []interface{}{t.Hour(), t.Minute(), t.Second(), t.Nanosecond() / 1e6}
}
