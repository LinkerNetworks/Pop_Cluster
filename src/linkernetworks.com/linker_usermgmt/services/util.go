package services

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"regexp"
	"strings"
	"time"

	"encoding/json"
	"github.com/Sirupsen/logrus"
	"linkernetworks.com/linker_common_lib/httpclient"
	"linkernetworks.com/linker_common_lib/persistence/entity"
	"linkernetworks.com/linker_common_lib/rest/response"
	"linkernetworks.com/linker_usermgmt/common"
)

var (
	COMMON_ERROR_INVALIDATE   = "E12002"
	COMMON_ERROR_UNAUTHORIZED = "E12004"
	COMMON_ERROR_UNKNOWN      = "E12001"
	COMMON_ERROR_INTERNAL     = "E12003"
)

type UserParam struct {
	UserName string
	Email    string
	Password string
	Company  string
}

/*func IsFirstNodeInZK() bool {
	hostname, err := os.Hostname()
	if err != nil {
		logrus.Warnln("get host name error!", err)
		return false
	}

	path, err := common.UTIL.ZkClient.GetFirstUserMgmtPath()
	if err != nil {
		logrus.Warnln("get usermgmt node from zookeeper error!", err)
		return false
	}

	return strings.HasPrefix(path, hostname)

}*/

func HashString(password string) string {
	encry := md5.Sum([]byte(password))
	return hex.EncodeToString(encry[:])
}

func IsUserNameValid(name string) bool {
	reg := regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,255}$`)
	return reg.MatchString(name)
}

func GetWaitTime(execTime time.Time) int64 {
	one_day := 24 * 60 * 60
	currenTime := time.Now()

	execInt := execTime.Unix()
	currentInt := currenTime.Unix()

	var waitTime int64
	if currentInt <= execInt {
		waitTime = execInt - currentInt
	} else {
		waitTime = (execInt + int64(one_day)) % currentInt
	}

	return waitTime
}

//default expire time is 6 hours
func GenerateExpireTime(expire int64) float64 {
	t := time.Now().Unix()

	t += expire

	return float64(t)
}

func GetClusterByUser(userid string, x_auth_token string) (cluster []entity.Cluster, err error) {
	clusterurl := common.UTIL.Props.GetString("nodebanlancer.url","")
	url := strings.Join([]string{"http://", clusterurl, ":10002","/v1/cluster?user_id=", userid,"&status=unterminated"}, "")
	logrus.Debugln("get cluster url=" + url)

	resp, err := httpclient.Http_get(url, "",
		httpclient.Header{"Content-Type", "application/json"}, httpclient.Header{"X-Auth-Token", x_auth_token})
	if err != nil {
		logrus.Errorf("http get cluster error %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("get cluster by username failed %v", string(data))
		return nil, errors.New("get cluster by username failed")
	}

	cluster = []entity.Cluster{}
	err = getRetFromResponse(data, cluster)
	return

}

func getRetFromResponse(data []byte, obj interface{}) (err error) {
	var resp *response.Response
	resp = new(response.Response)
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	jsonout, err := json.Marshal(resp.Data)
	if err != nil {
		return err
	}

	json.Unmarshal(jsonout, obj)

	return
}
