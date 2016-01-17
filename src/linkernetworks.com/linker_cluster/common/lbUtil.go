package common

import "github.com/magiconair/properties"

var (
	UTIL *Util
)

type Util struct {
	Props    *properties.Properties
	LbClient *LbClient
}

type LbClient struct {
	Host string
}

func (p *LbClient) GetUserMgmtEndpoint() (endpoint string, err error) {
	userMgmtPort := UTIL.Props.MustGetString("lb.usermgmt.port")
	endpoint = p.Host + ":" + userMgmtPort
	return
}
