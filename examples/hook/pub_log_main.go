package main

import (
    "github.com/logrus"
    "github.com/logrus/examples/hook/pub"
    "github.com/logrus/examples/hook/sub"
    "time"
    "fmt"
)

var log *logrus.Logger = pub.GLog

func main() {
    sub.PubTestPrint()
    MainLogTest()
}

func MainLogTest() {
    log.Println("MAIN: Hello")
    for i := 0; i < 10; i++ {
        log.Debugf("MAIN: haha: %s - %d", "Welcome", i)
        log.Infof ("MAIN: haha: %s - %d", "Welcome", i)
        log.Warnf ("MAIN: haha: %s - %d", "Welcome", i)
        log.Errorf("MAIN: haha: %s - %d", "Welcome", i)
        fmt.Printf("Main: std-print- %d GLog:%v\n", i, pub.GLog)
        //log.Fatalf("MAIN: haha: %s - %d", "Welcome", i)
        time.Sleep(time.Second*2)
    }
}