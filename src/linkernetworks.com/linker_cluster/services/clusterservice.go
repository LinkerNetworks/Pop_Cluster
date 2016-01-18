package services

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"

	"linkernetworks.com/linker_common_lib/persistence/dao"
	"linkernetworks.com/linker_common_lib/persistence/entity"
)

const (
	CLUSTER_STATUS_TERMINATED  = "TERMINATED"
	CLUSTER_STATUS_RUNNING     = "RUNNING"
	CLUSTER_STATUS_DEPLOYING   = "DEPLOYING"
	CLUSTER_STATUS_FAILED      = "FAILED"
	CLUSTER_STATUS_TERMINATING = "TERMINATING"

	CLUSTER_ERROR_CREATE string = "E50000"
	CLUSTER_ERROR_UPDATE string = "E50001"
	CLUSTER_ERROR_DELETE string = "E50002"
	CLUSTER_ERROR_UNIQUE string = "E50003"
	CLUSTER_ERROR_QUERY  string = "E50004"

	CLUSTER_ERROR_PARSE_NUMBER       string = "E50010"
	CLUSTER_ERROR_NO_SUCH_MANY_NODES string = "E50011"
)

var (
	clusterService *ClusterService = nil
	onceCluster    sync.Once
)

type ClusterService struct {
	collectionName string
}

func GetClusterService() *ClusterService {
	onceCluster.Do(func() {
		logrus.Debugf("Once called from clusterService ......................................")
		clusterService = &ClusterService{"cluster"}
	})
	return clusterService

}

func (p *ClusterService) Create(cluster entity.Cluster, x_auth_token string) (newCluster *entity.Cluster,
	errorCode string, err error) {
	logrus.Infof("start to create cluster [%v]", cluster)

	// do authorize first
	if authorized := GetAuthService().Authorize("create_cluster", x_auth_token, "", p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("create cluster [%v] error is %v", cluster, err)
		return
	}

	//check cluster name
	if !isClusterNameValid(cluster.Name) {
		return nil, CLUSTER_ERROR_CREATE, errors.New("Invalid cluster name.")
	}

	//check instances count
	if cluster.Instances < 5 {
		return nil, CLUSTER_ERROR_CREATE, errors.New("Invalid cluster instances, 5 at least")
	}

	//set cluster type to default
	if cluster.Type == "" {
		cluster.Type = "user"
	}
	if cluster.Type == "user" {
		return p.CreateUserCluster(cluster, x_auth_token)
	} else if cluster.Type == "mgmt" {
		return p.CreateMgmtCluster(cluster, x_auth_token)
	} else {
		return nil, CLUSTER_ERROR_CREATE, errors.New("unsupport cluster type, user|mgmt expected")
	}

}

func (p *ClusterService) CreateUserCluster(cluster entity.Cluster, x_auth_token string) (newCluster *entity.Cluster,
	errorCode string, err error) {

	//check cluster name
	//name of someone's unterminated clusters should be unique
	query := bson.M{}
	query["status"] = bson.M{"$ne": CLUSTER_STATUS_TERMINATED}
	query["user_id"] = cluster.UserId
	query["name"] = cluster.Name
	n, _, errorCode, err := p.queryByQuery(query, 0, 0, "", x_auth_token, false)
	if err != nil {
		logrus.Errorf("query unterminated clusters, error is %v", err)
		return nil, errorCode, err
	}
	if n > 0 {
		//name already exist
		err = errors.New("the name of cluster must be unique!")
		errorCode = CLUSTER_ERROR_UNIQUE
		logrus.Errorf("create cluster [%v] error is %v", cluster, err)
		return
	}

	// generate ObjectId
	cluster.ObjectId = bson.NewObjectId()

	userId := cluster.UserId
	if len(userId) == 0 {
		err = errors.New("user_id not provided")
		errorCode = CLUSTER_ERROR_CREATE
		logrus.Errorf("create cluster [%v] error is %v", cluster, err)
		return
	}

	user, err := GetUserById(userId, x_auth_token)
	if err != nil {
		logrus.Errorf("get user by id err is %v", err)
		errorCode = CLUSTER_ERROR_CREATE
		return nil, errorCode, err
	}
	cluster.TenantId = user.TenantId
	cluster.Owner = user.Username

	// set created_time and updated_time
	cluster.TimeCreate = dao.GetCurrentTime()
	cluster.TimeUpdate = cluster.TimeCreate
	cluster.Status = CLUSTER_STATUS_DEPLOYING
	// insert bson to mongodb
	err = dao.HandleInsert(p.collectionName, cluster)
	if err != nil {
		errorCode = CLUSTER_ERROR_CREATE
		logrus.Errorf("create cluster [%v] to bson error is %v", cluster, err)
		return
	}

	for i := 0; i < cluster.Instances; i++ {
		host := entity.Host{}
		host.ClusterId = cluster.ObjectId.Hex()
		host.ClusterName = cluster.Name
		host.Status = HOST_STATUS_DEPLOYING
		host.UserId = cluster.UserId
		host.TimeCreate = dao.GetCurrentTime()
		host.TimeUpdate = host.TimeCreate

		go GetHostService().Create(host, x_auth_token)
	}

	return
}

func (p *ClusterService) CreateMgmtCluster(cluster entity.Cluster, x_auth_token string) (newCluster *entity.Cluster,
	errorCode string, err error) {
	//check if mgmt cluster already exist
	//only 1 allowed in database
	query := bson.M{}
	query["type"] = "mgmt"
	n, _, _, err := p.queryByQuery(query, 0, 0, "", x_auth_token, false)
	if n > 0 {
		return nil, CLUSTER_ERROR_CREATE, errors.New("mgmt cluster already exist")
	}

	if cluster.Name == "" {
		cluster.Name = "Management"
	}
	token, err := GetTokenById(x_auth_token)
	if err != nil {
		logrus.Errorf("get token by id error is %v", err)
		errorCode = CLUSTER_ERROR_CREATE
		return nil, errorCode, err
	}
	if cluster.UserId == "" {
		cluster.UserId = token.User.Id
	}
	if cluster.Owner == "" {
		cluster.Owner = token.User.Username
	}
	if cluster.Details == "" {
		cluster.Details = "Cluster to manage other clusters"
	}
	cluster.TimeCreate = dao.GetCurrentTime()
	cluster.TimeUpdate = cluster.TimeCreate

	cluster.Status = CLUSTER_STATUS_DEPLOYING

	// insert bson to mongodb
	err = dao.HandleInsert(p.collectionName, cluster)
	if err != nil {
		errorCode = CLUSTER_ERROR_CREATE
		logrus.Errorf("create cluster [%v] to bson error is %v", cluster, err)
		return
	}

	for i := 0; i < cluster.Instances; i++ {
		host := entity.Host{}
		host.ClusterId = cluster.ObjectId.Hex()
		host.ClusterName = cluster.Name
		host.Status = HOST_STATUS_DEPLOYING
		host.UserId = cluster.UserId
		host.TimeCreate = dao.GetCurrentTime()
		host.TimeUpdate = host.TimeCreate

		go GetHostService().Create(host, x_auth_token)
	}

	return
}

//query unterminated clusters
//filter by cluster id
//filter by user id
func (p *ClusterService) QueryCluster(name string, userId string, status string, skip int,
	limit int, sort string, x_auth_token string) (total int, clusters []entity.Cluster,
	errorCode string, err error) {
	query := bson.M{}
	if len(strings.TrimSpace(name)) > 0 {
		query["name"] = name
	}
	if len(strings.TrimSpace(userId)) > 0 {
		query["user_id"] = userId
	}
	if strings.TrimSpace(status) == "" {
		//query all clusters by default if this parameter is not provided
		//do nothing
	} else if status == "unterminated" {
		//assume a special status
		//"unterminated" means !TERMINATED(DEPLOYING|RUNNING|FAILED|TERMINATING)
		query["status"] = bson.M{"$ne": CLUSTER_STATUS_TERMINATED}
	} else if status == CLUSTER_STATUS_RUNNING || status == CLUSTER_STATUS_DEPLOYING ||
		status == CLUSTER_STATUS_FAILED || status == CLUSTER_STATUS_TERMINATED ||
		status == CLUSTER_STATUS_TERMINATING {
		query["status"] = status
	} else {
		errorCode = CLUSTER_ERROR_QUERY
		err := errors.New("Invalid parameter status")
		return 0, nil, errorCode, err
	}

	return p.queryByQuery(query, skip, limit, sort, x_auth_token, false)
}

func (p *ClusterService) queryByQuery(query bson.M, skip int, limit int, sort string,
	x_auth_token string, skipAuth bool) (total int, clusters []entity.Cluster,
	errorCode string, err error) {
	authQuery := bson.M{}
	if !skipAuth {
		// get auth query from auth service first
		authQuery, err = GetAuthService().BuildQueryByAuth("list_cluster", x_auth_token)
		if err != nil {
			logrus.Errorf("get auth query by token [%v] error is %v", x_auth_token, err)
			errorCode = COMMON_ERROR_INTERNAL
			return
		}
	}

	selector := generateQueryWithAuth(query, authQuery)
	clusters = []entity.Cluster{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           skip,
		Limit:          limit,
		Sort:           sort,
	}
	total, err = dao.HandleQueryAll(&clusters, queryStruct)
	if err != nil {
		logrus.Errorf("query clusters by query [%v] error is %v", query, err)
		errorCode = CLUSTER_ERROR_QUERY
	}
	return
}

func generateQueryWithAuth(oriQuery bson.M, authQuery bson.M) (query bson.M) {
	if len(authQuery) == 0 {
		query = oriQuery
	} else {
		query = bson.M{}
		query["$and"] = []bson.M{oriQuery, authQuery}
	}
	logrus.Debugf("generated query [%v] with auth [%v], result is [%v]", oriQuery, authQuery, query)
	return
}

func (p *ClusterService) UpdateStateById(objectId string, newState string, x_auth_token string) (created bool,
	errorCode string, err error) {
	logrus.Infof("start to update cluster by objectId [%v] status to %v", objectId, newState)
	// do authorize first
	if authorized := GetAuthService().Authorize("update_cluster", x_auth_token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("update cluster with objectId [%v] status to [%v] failed, error is %v", objectId, newState, err)
		return
	}
	// validate objectId
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}
	cluster, _, err := p.QueryById(objectId, x_auth_token)
	if err != nil {
		logrus.Errorf("get cluster by objeceId [%v] failed, error is %v", objectId, err)
		return
	}
	if cluster.Status == newState {
		logrus.Infof("this cluster [%v] is already in state [%v]", cluster, newState)
		return false, "", nil
	}
	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)

	change := bson.M{"status": newState, "time_update": dao.GetCurrentTime()}
	err = dao.HandleUpdateByQueryPartial(p.collectionName, selector, change)
	if err != nil {
		logrus.Errorf("update cluster with objectId [%v] status to [%v] failed, error is %v", objectId, newState, err)
	}
	created = true
	return

}

func (p *ClusterService) QueryById(objectId string, x_auth_token string) (cluster entity.Cluster,
	errorCode string, err error) {
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	// do authorize first
	if authorized := GetAuthService().Authorize("get_cluster", x_auth_token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("get cluster with objectId [%v] error is %v", objectId, err)
		return
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)
	cluster = entity.Cluster{}
	err = dao.HandleQueryOne(&cluster, dao.QueryStruct{p.collectionName, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("query cluster [objectId=%v] error is %v", objectId, err)
		errorCode = CLUSTER_ERROR_QUERY
	}
	return

}

func (p *ClusterService) DeleteByUserId(userId string, token string) (errorCode string, err error) {
	logrus.Infof("start to delete Cluster with userid [%v]", userId)

	if len(userId) == 0 {
		error_code := CLUSTER_ERROR_DELETE
		err = errors.New("Invalid parameter userid")
		return error_code, err
	}

	selector, err := GetAuthService().BuildQueryByAuth("delete_clusters", token)
	if err != nil {
		logrus.Errorf("get auth query by token [%v] error is %v", token, err)
		errorCode = COMMON_ERROR_INTERNAL
		return errorCode, err
	}

	selector["user_id"] = userId

	clusters := []entity.Cluster{}
	_, err = dao.HandleQueryAll(&clusters, dao.QueryStruct{p.collectionName, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("get all cluster by userId error %v", err)
		errorCode = CLUSTER_ERROR_UPDATE
		return
	}

	for _, cluster := range clusters {
		_, err := p.DeleteById(cluster.ObjectId.Hex(), token)
		if err != nil {
			logrus.Errorf("delete cluster by id error %v", err)
			continue
		}
	}

	return
}

func (p *ClusterService) DeleteClusters(clusterIds []string, x_auth_token string) (errorCode string, err error) {
	logrus.Infof("start to delete clusters ids [%v]", clusterIds)

	total := len(clusterIds)
	ok_count := total

	for i, clusterId := range clusterIds {
		logrus.Infof("deleting clusters %d of %d", (i + 1), total)
		errorCode, err = p.DeleteById(clusterId, x_auth_token)
		if err != nil {
			logrus.Errorf("delete cluster error is %v", err)
			ok_count--
		}
	}

	logrus.Infof("delete clusters complete , %d OK, %d Total", ok_count, total)

	if ok_count < total {
		logrus.Infof("Not all clusters delete successfully")
		return CLUSTER_ERROR_DELETE, errors.New("not all clusters deleted successfully.")
	}

	return
}

func (p *ClusterService) DeleteById(clusterId string, x_auth_token string) (errorCode string, err error) {
	logrus.Infof("start to delete Cluster with id [%v]", clusterId)
	// do authorize first
	if authorized := GetAuthService().Authorize("delete_cluster", x_auth_token, clusterId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("authorize failure when deleting cluster with id [%v] , error is %v", clusterId, err)
		return
	}
	if !bson.IsObjectIdHex(clusterId) {
		err = errors.New("Invalid cluster id.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	//query cluster
	cluster, errorCode, err := p.QueryById(clusterId, x_auth_token)
	if err != nil {
		logrus.Errorf("query cluster error is %v", err)
		return
	}

	//check status
	switch cluster.Status {
	case CLUSTER_STATUS_DEPLOYING, CLUSTER_STATUS_TERMINATING, CLUSTER_STATUS_TERMINATED:
		logrus.Errorf("Cannot operate on a %s cluster", cluster.Status)
		return CLUSTER_ERROR_DELETE, errors.New("Cannot operate on a " + cluster.Status + "cluster")
	case CLUSTER_STATUS_FAILED:
		//set cluster status TERMINATING
		_, _, err = p.UpdateStateById(clusterId, CLUSTER_STATUS_TERMINATED, x_auth_token)
		if err != nil {
			logrus.Errorf("update cluster [objectId=%v] status error is %v", clusterId, err)
			return CLUSTER_ERROR_DELETE, err
		}
	case CLUSTER_STATUS_RUNNING:
		//terminate the cluster
		return func(clusterId string, x_auth_token string) (errorCode string, err error) {
			//set cluster status TERMINATING
			_, _, err = p.UpdateStateById(clusterId, CLUSTER_STATUS_TERMINATING, x_auth_token)
			if err != nil {
				logrus.Errorf("update cluster [objectId=%v] status error is %v", clusterId, err)
				return CLUSTER_ERROR_DELETE, err
			}

			//query all hosts
			var total int
			total, hosts, errorCode, err := GetHostService().QueryAllByClusterId(clusterId, 0, 0, x_auth_token)
			if err != nil {
				logrus.Errorf("query hosts in cluster %s error is %v", clusterId, err)
				return HOST_ERROR_QUERY, err
			}

			//terminate all hosts
			ok_count := total
			for _, host := range hosts {
				//delete master node now
				_, _, err = GetHostService().TerminateHostById(host.ObjectId.Hex(), true, x_auth_token)
				if err != nil {
					logrus.Errorf("delete host [objectId=%v] error is %v", host.ObjectId.Hex(), err)
					errorCode = HOST_ERROR_DELETE
					ok_count--
				}
				continue
			}

			logrus.Infof("Cluster %s has %d hosts, %d successfully terminated", clusterId, total, ok_count)

			if ok_count < total {
				//FAILED
				_, _, err = p.UpdateStateById(clusterId, CLUSTER_STATUS_FAILED, x_auth_token)
				if err != nil {
					logrus.Errorf("update cluster [objectId=%v] status error is %v", clusterId, err)
					return CLUSTER_ERROR_DELETE, err
				}

				logrus.Errorf("delete cluster failure because %d hosts terminate failed", total-ok_count)
				return CLUSTER_ERROR_DELETE, errors.New("not all hosts terminated")
			}

			//TERMINATED
			_, _, err = p.UpdateStateById(clusterId, CLUSTER_STATUS_TERMINATED, x_auth_token)
			if err != nil {
				logrus.Errorf("delete cluster [objectId=%v] error is %v", clusterId, err)
				return CLUSTER_ERROR_DELETE, err
			}
			return
		}(clusterId, x_auth_token)
	default:
		logrus.Errorf("Unknown cluster status %s", cluster.Status)
		return CLUSTER_ERROR_DELETE, errors.New("Unknown cluster status " + cluster.Status)
	}

	return
}

func (p *ClusterService) AddHosts(clusterId, numberStr string, x_auth_token string) (cluster entity.Cluster, errorCode string, err error) {
	logrus.Infof("start to add hosts")
	logrus.Infof("clusterId is %s, number is %s.", clusterId, numberStr)

	// do authorize first
	if authorized := GetAuthService().Authorize("create_host", x_auth_token, clusterId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("authorize failure when adding hosts in cluster with id [%v] , error is %v", clusterId, err)
		return
	}
	if !bson.IsObjectIdHex(clusterId) {
		err = errors.New("Invalid cluster id.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(clusterId)
	cluster = entity.Cluster{}
	err = dao.HandleQueryOne(&cluster, dao.QueryStruct{p.collectionName, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("query cluster [objectId=%v] error is %v", clusterId, err)
		errorCode = CLUSTER_ERROR_QUERY
	}

	number, err := strconv.ParseInt(numberStr, 10, 0)
	if err != nil {
		logrus.Errorf("Parse number [objectId=%v] error is %v", numberStr, err)
		errorCode = CLUSTER_ERROR_PARSE_NUMBER
	}

	if number <= 0 {
		//call terminate hosts to do this
		error_code := CLUSTER_ERROR_UPDATE
		err = errors.New("Invalid number, it should be positive")
		return cluster, error_code, err
	}

	//create hosts
	var ok_count int = int(number)
	for i := 0; i < int(number); i++ {
		host := entity.Host{}
		host.ClusterId = clusterId
		host.ClusterName = cluster.Name
		host.Status = HOST_STATUS_DEPLOYING
		_, _, err := GetHostService().Create(host, x_auth_token)
		if err != nil {
			logrus.Errorf("creat host error is %v", err)
			ok_count--
		}
	}

	if ok_count < int(number) {
		logrus.Errorf("not all hosts successfully created. %d of %d", ok_count, number)
	}

	//set instances in cluster
	cluster.Instances = cluster.Instances + int(ok_count)
	change := bson.M{"instances": cluster.Instances, "time_update": dao.GetCurrentTime()}
	err = dao.HandleUpdateByQueryPartial(p.collectionName, selector, change)
	if err != nil {
		logrus.Errorf("update cluster's [%v] instances to [%v] failed, error is %v", clusterId, cluster.Instances, err)
		errorCode = CLUSTER_ERROR_UPDATE
		return
	}

	return

}

//terminate specified hosts of a cluster
func (p *ClusterService) TerminateHosts(clusterId string, hostIds []string, x_auth_token string) (errorCode string, err error) {
	logrus.Infof("start to decrease cluster hosts [%v]", hostIds)

	if !bson.IsObjectIdHex(clusterId) {
		err = errors.New("Invalid cluster_id")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	if len(hostIds) == 0 {
		errorCode = HOST_ERROR_DELETE
		err = errors.New("Empty array of host id")
		return errorCode, err
	}

	//query cluster by clusterId
	cluster := entity.Cluster{}
	clusterSelector := bson.M{}
	clusterSelector["_id"] = bson.ObjectIdHex(clusterId)
	err = dao.HandleQueryOne(&cluster, dao.QueryStruct{p.collectionName, clusterSelector, 0, 0, ""})

	instances := cluster.Instances

	for _, hostId := range hostIds {
		//call API to terminate host(master node cannot be deleted now)
		_, errorCode, err := GetHostService().TerminateHostById(hostId, false, x_auth_token)
		if err != nil {
			logrus.Errorf("terminate host error is %s,%v", errorCode, err)
			continue
		}
		//decline the host count only when DeleteById() run success
		instances--
	}

	//update hosts count
	cluster.Instances = instances
	created, err := dao.HandleUpdateOne(&cluster, dao.QueryStruct{p.collectionName, clusterSelector, 0, 0, ""})
	if !created {
		logrus.Errorln("update cluster error is %v", err)
		return CLUSTER_ERROR_UPDATE, err
	}

	return
}

//check cluster name with regex
//letters (upper or lowercase)
//numbers (0-9)
//underscore (_)
//dash (-)
//point (.)
//length 1-255
//no spaces! or other characters
func isClusterNameValid(name string) bool {
	reg := regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,255}$`)
	return reg.MatchString(name)
}
