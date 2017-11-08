package sub

import (
    "github.com/logrus"
    "github.com/logrus/examples/hook/pub"
)

var log *logrus.Logger = pub.GLog

func PubTestPrint() {
    log.Debugf("Sub: Hello...%s", "World")
    log.Warnf ("Sub: Hello...%s", "World")
    log.Infof ("Sub: Hello...%s", "World")
    //log.Fatalf("Sub: Hello...%s", "World")
}