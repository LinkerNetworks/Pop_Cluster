package common

import (
//	"errors"
//	"github.com/Sirupsen/logrus"
	"github.com/magiconair/properties"
//	"github.com/samuel/go-zookeeper/zk"
//	"math/rand"
//	"strings"
//	"time"
)

var UTIL *Util
var (
	clusterMgmtPath      string = "/cluster"
	clusterMgmtEndpoints []string
)

type Util struct {
	Props    *properties.Properties
}



