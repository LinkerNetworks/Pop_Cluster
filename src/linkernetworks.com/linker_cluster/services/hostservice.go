package services

import (
	"errors"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
	"linkernetworks.com/linker_common_lib/persistence/dao"
	"linkernetworks.com/linker_common_lib/persistence/entity"
)

var (
	hostService             *HostService = nil
	onceHost                sync.Once
	HOST_STATUS_TERMINATED  = "TERMINATED"
	HOST_STATUS_RUNNING     = "RUNNING"
	HOST_STATUS_DEPLOYING   = "DEPLOYING"
	HOST_STATUS_FAILED      = "FAILED"
	HOST_STATUS_TERMINATING = "TERMINATING"

	HOST_ERROR_CREATE        string = "E50100"
	HOST_ERROR_UPDATE        string = "E50101"
	HOST_ERROR_DELETE        string = "E50102"
	HOST_ERROR_QUERY         string = "E50103"
	HOST_ERROR_TERMINATED    string = "E50104"
	HOST_ERROR_DELETE_MASTER string = "E50105"
)

type HostService struct {
	collectionName string
}

func GetHostService() *HostService {
	onceHost.Do(func() {
		logrus.Debugf("Once called from hostsService ......................................")
		hostService = &HostService{"hosts"}
	})
	return hostService
}

func (p *HostService) Create(host entity.Host, x_auth_token string) (newHost entity.Host,
	errorCode string, err error) {
	logrus.Infof("start to create host [%v]", host)
	// do authorize first
	if authorized := GetAuthService().Authorize("create_host", x_auth_token, "", p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("create host [%v] error is %v", host, err)
		return
	}

	// generate ObjectId
	host.ObjectId = bson.NewObjectId()

	token, err := GetTokenById(x_auth_token)
	if err != nil {
		errorCode = HOST_ERROR_CREATE
		logrus.Errorf("get token failed when create host [%v], error is %v", host, err)
		return
	}

	// set token_id and user_id from token
	host.TenantId = token.Tenant.Id
	host.UserId = token.User.Id

	// set created_time and updated_time
	host.TimeCreate = dao.GetCurrentTime()
	host.TimeUpdate = host.TimeCreate

	// insert bson to mongodb
	err = dao.HandleInsert(p.collectionName, host)
	if err != nil {
		errorCode = HOST_ERROR_CREATE
		logrus.Errorf("insert host [%v] to db error is %v", host, err)
		return
	}

	newHost = host

	return
}

func (p *HostService) QueryHosts(clusterId string, skip int, limit int, status string, x_auth_token string) (total int,
	hosts []entity.Host, errorCode string, err error) {
	query := bson.M{}
	if len(strings.TrimSpace(clusterId)) > 0 {
		query["cluster_id"] = clusterId
	}

	switch strings.TrimSpace(status) {
	case "":
	//query all hosts by default if this parameter is not provided
	//do nothing
	case HOST_STATUS_DEPLOYING, HOST_STATUS_RUNNING, HOST_STATUS_FAILED,
		HOST_STATUS_TERMINATING, HOST_STATUS_TERMINATED:
		query["status"] = status
	case "unterminated":
		query["status"] = bson.M{"$ne": HOST_STATUS_TERMINATED}
	default:
		errorCode = COMMON_ERROR_INVALIDATE
		err := errors.New("Invalid parameter status")
		return 0, nil, errorCode, err
	}
	return p.queryByQuery(query, skip, limit, x_auth_token, false)
}

func (p *HostService) QueryById(objectId string, x_auth_token string) (host entity.Host,
	errorCode string, err error) {
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	// do authorize first
	if authorized := GetAuthService().Authorize("get_host", x_auth_token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("get host with objectId [%v] error is %v", objectId, err)
		return
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)
	host = entity.Host{}
	err = dao.HandleQueryOne(&host, dao.QueryStruct{p.collectionName, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("query host [objectId=%v] error is %v", objectId, err)
		errorCode = HOST_ERROR_QUERY
	}
	return
}

func (p *HostService) QueryAllByClusterId(clusterId string, skip int,
	limit int, x_auth_token string) (total int, hosts []entity.Host,
	errorCode string, err error) {
	if strings.TrimSpace(clusterId) == "" {
		return p.QueryAll(skip, limit, x_auth_token)
	}
	query := bson.M{}
	query["cluster_id"] = clusterId

	return p.queryByQuery(query, skip, limit, x_auth_token, false)
}

func (p *HostService) QueryAll(skip int, limit int, x_auth_token string) (total int,
	hosts []entity.Host, errorCode string, err error) {
	return p.queryByQuery(bson.M{}, skip, limit, x_auth_token, false)
}

func (p *HostService) queryByQuery(query bson.M, skip int, limit int,
	x_auth_token string, skipAuth bool) (total int, hosts []entity.Host,
	errorCode string, err error) {
	authQuery := bson.M{}
	if !skipAuth {
		// get auth query from auth first
		authQuery, err = GetAuthService().BuildQueryByAuth("list_host", x_auth_token)
		if err != nil {
			logrus.Errorf("get auth query by token [%v] error is %v", x_auth_token, err)
			errorCode = COMMON_ERROR_INTERNAL
			return
		}
	}

	selector := generateQueryWithAuth(query, authQuery)
	hosts = []entity.Host{}
	// fix : "...." sort by time_create
	queryStruct := dao.QueryStruct{p.collectionName, selector, skip, limit, "time_create"}
	total, err = dao.HandleQueryAll(&hosts, queryStruct)
	if err != nil {
		logrus.Errorf("query hosts by query [%v] error is %v", query, err)
		errorCode = HOST_ERROR_QUERY

	}
	return
}

func (p *HostService) UpdateById(objectId string, host entity.Host, x_auth_token string) (created bool,
	errorCode string, err error) {
	logrus.Infof("start to update host [%v]", host)
	// do authorize first
	if authorized := GetAuthService().Authorize("update_host", x_auth_token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("update host with objectId [%v] error is %v", objectId, err)
		return
	}

	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	// FIXING
	//	hostquery, _, _  := p.QueryById(objectId, x_auth_token)
	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)

	host.ObjectId = bson.ObjectIdHex(objectId)
	host.TimeUpdate = dao.GetCurrentTime()

	logrus.Infof("start to change host")
	err = dao.HandleUpdateByQueryPartial(p.collectionName, selector, &host)
	//	created, err = dao.HandleUpdateOne(&host, dao.QueryStruct{p.collectionName, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("update host [%v] error is %v", host, err)
		errorCode = HOST_ERROR_UPDATE
	}
	created = true
	return
}

func (p *HostService) UpdateStatusById(objectId string, status string, x_auth_token string) (created bool,
	errorCode string, err error) {
	logrus.Infof("start to update host by objectId [%v] status to %v", objectId, status)
	// do authorize first
	if authorized := GetAuthService().Authorize("update_host", x_auth_token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("update host with objectId [%v] status to [%v] failed, error is %v", objectId, status, err)
		return
	}
	// validate objectId
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}
	host, _, err := p.QueryById(objectId, x_auth_token)
	if err != nil {
		logrus.Errorf("get host by objeceId [%v] failed, error is %v", objectId, err)
		return
	}
	if host.Status == status {
		logrus.Infof("this host [%v] is already in state [%v]", host, status)
		return false, "", nil
	}
	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)

	change := bson.M{"status": status, "time_update": dao.GetCurrentTime()}
	err = dao.HandleUpdateByQueryPartial(p.collectionName, selector, change)
	if err != nil {
		logrus.Errorf("update host with objectId [%v] status to [%v] failed, error is %v", objectId, status, err)
		created = false
		return
	}
	created = true
	return

}
