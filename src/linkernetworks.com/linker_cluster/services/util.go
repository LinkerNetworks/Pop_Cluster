package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"

	"github.com/Sirupsen/logrus"
	"linkernetworks.com/linker_cluster/common"
	"linkernetworks.com/linker_common_lib/httpclient"
	"linkernetworks.com/linker_common_lib/persistence/entity"
	"linkernetworks.com/linker_common_lib/rest/response"
)

var (
	COMMON_ERROR_INVALIDATE   = "E12002"
	COMMON_ERROR_UNAUTHORIZED = "E12004"
	COMMON_ERROR_UNKNOWN      = "E12001"
	COMMON_ERROR_INTERNAL     = "E12003"
)

func getErrorFromResponse(data []byte) (errorCode string, err error) {
	var resp *response.Response
	resp = new(response.Response)
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return COMMON_ERROR_INTERNAL, err
	}

	errorCode = resp.Error.Code
	err = errors.New(resp.Error.ErrorMsg)
	return
}

func TokenValidation(tokenId string) (errorCode string, err error) {
	userUrl, err := common.UTIL.LbClient.GetUserMgmtEndpoint()
	if err != nil {
		logrus.Errorf("get userMgmt endpoint err is %v", err)
		return COMMON_ERROR_INTERNAL, err
	}
	url := strings.Join([]string{"http://", userUrl, "/v1/token/?", "token=", tokenId}, "")
	logrus.Debugln("token validation url=" + url)

	resp, err := httpclient.Http_get(url, "",
		httpclient.Header{"Content-Type", "application/json"})
	if resp == nil {
		return COMMON_ERROR_INTERNAL, err
	}
	defer resp.Body.Close()
	if err != nil {
		logrus.Errorf("http get token validate error %v", err)
		return COMMON_ERROR_INTERNAL, err
	}

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("token validation failed %v", string(data))
		errorCode, err = getErrorFromResponse(data)
		return
	}

	return "", nil
}
func GetTokenById(token string) (currentToken *entity.Token, err error) {
	userUrl, err := common.UTIL.LbClient.GetUserMgmtEndpoint()
	if err != nil {
		logrus.Errorf("get userMgmt endpoint err is %v", err)
		return nil, err
	}
	url := strings.Join([]string{"http://", userUrl, "/v1/token/", token}, "")
	logrus.Debugln("get token url=" + url)

	resp, err := httpclient.Http_get(url, "",
		httpclient.Header{"Content-Type", "application/json"}, httpclient.Header{"X-Auth-Token", token})
	if resp == nil {
		return nil, errors.New("Nil pointer")
	}
	defer resp.Body.Close()
	if err != nil {
		logrus.Errorf("http get token error %v", err)
		return nil, err
	}

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("get token by id failed %v", string(data))
		return nil, errors.New("get token by id failed")
	}

	currentToken = new(entity.Token)
	err = getRetFromResponse(data, currentToken)
	return
}

func GetUserById(userId string, token string) (user *entity.User, err error) {
	userUrl, err := common.UTIL.LbClient.GetUserMgmtEndpoint()
	if err != nil {
		logrus.Errorf("get userMgmt endpoint err is %v", err)
		return nil, err
	}
	url := strings.Join([]string{"http://", userUrl, "/v1/user/", userId}, "")
	logrus.Debugln("get user url=" + url)

	resp, err := httpclient.Http_get(url, "",
		httpclient.Header{"Content-Type", "application/json"}, httpclient.Header{"X-Auth-Token", token})
	if resp == nil {
		return nil, errors.New("Nil pointer")
	}
	defer resp.Body.Close()
	if err != nil {
		logrus.Errorf("http get user error %v", err)
		return nil, err
	}

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("get user by id failed %v", string(data))
		return nil, errors.New("get user by id failed")
	}

	user = new(entity.User)
	err = getRetFromResponse(data, user)
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

func getCountFromResponse(data []byte) (count int, err error) {
	var resp *response.QueryStruct
	resp = new(response.QueryStruct)
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return
	}

	jsonout, err := json.Marshal(resp.Count)
	if err != nil {
		return
	}

	json.Unmarshal(jsonout, &count)

	return
}

func HashString(password string) string {
	encry := sha256.Sum256([]byte(password))
	return hex.EncodeToString(encry[:])
}
