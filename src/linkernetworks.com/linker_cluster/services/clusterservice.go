package services

import (
	"errors"
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

	CLUSTER_ERROR_CREATE     string = "E50000"
	CLUSTER_ERROR_UPDATE     string = "E50001"
	CLUSTER_ERROR_DELETE     string = "E50002"
	CLUSTER_ERROR_NAME_EXIST string = "E50003"
	CLUSTER_ERROR_QUERY      string = "E50004"

	CLUSTER_ERROR_INVALID_NUMBER     string = "E50010"
	CLUSTER_ERROR_INVALID_NAME       string = "E50011"
	CLUSTER_ERROR_INVALID_TYPE       string = "E50012"
	CLUSTER_ERROR_CALL_USERMGMT      string = "E50013"
	CLUSTER_ERROR_CALL_DEPLOYMENT    string = "E50014"
	CLUSTER_ERROR_CALL_MONGODB       string = "E50015"
	CLUSTER_ERROR_INVALID_STATUS     string = "E50016"
	CLUSTER_ERROR_DELETE_NOT_ALLOWED string = "E50017"
	CLUSTER_ERROR_DELETE_NODE_NUM    string = "E50018"
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

func (p *ClusterService) CheckClusterName(userId string, clusterName string, x_auth_token string) (errorCode string, err error) {
	logrus.Infof("checking clustername [%s] for user with id [%s]", clusterName, userId)

	// authorization
	if authorized := GetAuthService().Authorize("list_cluster", x_auth_token, "", p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("check cluster name auth failure [%v]", err)
		return
	}

	// if !IsClusterNameValid(clusterName) {
	// 	logrus.Errorf("clustername [%s] do not match with regex\n", clusterName)
	// 	return CLUSTER_ERROR_INVALID_NAME, errors.New("Invalid clustername,does not match with regex")
	// }

	//check userId(must be a objectId at least)
	if !bson.IsObjectIdHex(userId) {
		logrus.Errorf("invalid userid [%s],not a object id\n", clusterName)
		return COMMON_ERROR_INVALIDATE, errors.New("Invalid userid,not a object id")
	}

	ok, errorCode, err := p.isClusterNameUnique(userId, clusterName)
	if err != nil {
		return errorCode, err
	}
	if !ok {
		logrus.Errorf("clustername [%s] already exist for user with id [%s]\n", clusterName, userId)
		return CLUSTER_ERROR_NAME_EXIST, errors.New("Invalid clustername,conflict")
	}
	return
}

//check if name of a user's clusters is conflict
func (p *ClusterService) isClusterNameUnique(userId string, clusterName string) (ok bool, errorCode string, err error) {
	//check cluster name
	//name of someone's unterminated clusters should be unique
	query := bson.M{}
	query["status"] = bson.M{"$ne": CLUSTER_STATUS_TERMINATED}
	query["user_id"] = userId
	query["name"] = clusterName
	n, _, errorCode, err := p.queryByQuery(query, 0, 0, "", "", true)
	if err != nil {
		return false, errorCode, err
	}
	if n > 0 {
		//name already exist
		return false, "", nil
	}
	return true, "", nil
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
	if !IsClusterNameValid(cluster.Name) {
		return nil, CLUSTER_ERROR_INVALID_NAME, errors.New("Invalid cluster name.")
	}

	//check userId(must be a objectId at least)
	if !bson.IsObjectIdHex(cluster.UserId) {
		logrus.Errorf("invalid userid [%s],not a object id\n", cluster.Name)
		return nil, COMMON_ERROR_INVALIDATE, errors.New("Invalid userid,not a object id")
	}

	//check if cluster name is unique
	ok, errorCode, err := p.isClusterNameUnique(cluster.UserId, cluster.Name)
	if err != nil {
		return nil, errorCode, err
	}
	if !ok {
		logrus.Errorf("clustername [%s] already exist for user with id [%s]\n", cluster.Name, cluster.UserId)
		return nil, CLUSTER_ERROR_NAME_EXIST, errors.New("Conflict clustername")
	}

	//check instances count
	if cluster.Instances < 5 {
		return nil, CLUSTER_ERROR_INVALID_NUMBER, errors.New("Invalid cluster instances, 5 at least")
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
		return nil, CLUSTER_ERROR_INVALID_TYPE, errors.New("unsupport cluster type, user|mgmt expected")
	}

}

func (p *ClusterService) CreateUserCluster(cluster entity.Cluster, x_auth_token string) (newCluster *entity.Cluster,
	errorCode string, err error) {

	// generate ObjectId
	cluster.ObjectId = bson.NewObjectId()

	userId := cluster.UserId
	if len(userId) == 0 {
		err = errors.New("user_id not provided")
		errorCode = COMMON_ERROR_INVALIDATE
		logrus.Errorf("create cluster [%v] error is %v", cluster, err)
		return
	}

	user, err := GetUserById(userId, x_auth_token)
	if err != nil {
		logrus.Errorf("get user by id err is %v", err)
		errorCode = CLUSTER_ERROR_CALL_USERMGMT
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
		errorCode = CLUSTER_ERROR_CALL_MONGODB
		logrus.Errorf("create cluster [%v] to bson error is %v", cluster, err)
		return
	}

	//add records of hosts in db
	for i := 0; i < cluster.Instances; i++ {
		host := entity.Host{}
		host.ClusterId = cluster.ObjectId.Hex()
		host.ClusterName = cluster.Name
		host.Status = HOST_STATUS_DEPLOYING
		host.UserId = cluster.UserId
		host.TimeCreate = dao.GetCurrentTime()
		host.TimeUpdate = host.TimeCreate

		_, _, err := GetHostService().Create(host, x_auth_token)
		if err != nil {
			logrus.Errorf("insert host to db error is [%v]", err)
		}
	}

	if IsDeploymentEnabled() {
		//call deployment
		go CreateCluster(cluster, x_auth_token)
	}

	newCluster = &cluster
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
		return nil, CLUSTER_ERROR_CALL_MONGODB, errors.New("mgmt cluster already exist")
	}

	if cluster.Name == "" {
		cluster.Name = "Management"
	}
	token, err := GetTokenById(x_auth_token)
	if err != nil {
		logrus.Errorf("get token by id error is %v", err)
		errorCode = CLUSTER_ERROR_CALL_USERMGMT
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
		errorCode = CLUSTER_ERROR_CALL_MONGODB
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

		_, _, err := GetHostService().Create(host, x_auth_token)
		if err != nil {
			logrus.Errorf("insert host to db error is [%v]", err)
		}
	}

	newCluster = &cluster

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
		errorCode = COMMON_ERROR_INVALIDATE
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
		return
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

func (p *ClusterService) UpdateStatusById(objectId string, status string, x_auth_token string) (created bool,
	errorCode string, err error) {
	logrus.Infof("start to update cluster by objectId [%v] status to %v", objectId, status)
	// do authorize first
	if authorized := GetAuthService().Authorize("update_cluster", x_auth_token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("update cluster with objectId [%v] status to [%v] failed, error is %v", objectId, status, err)
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
	if cluster.Status == status {
		logrus.Infof("this cluster [%v] is already in state [%v]", cluster, status)
		return false, "", nil
	}
	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)

	change := bson.M{"status": status, "time_update": dao.GetCurrentTime()}
	err = dao.HandleUpdateByQueryPartial(p.collectionName, selector, change)
	if err != nil {
		logrus.Errorf("update cluster with objectId [%v] status to [%v] failed, error is %v", objectId, status, err)
	}
	created = true
	return

}

//update cluster in db
func (p *ClusterService) UpdateCluster(cluster entity.Cluster, x_auth_token string) (created bool,
	errorCode string, err error) {
	query := bson.M{}
	query["_id"] = cluster.ObjectId
	created, err = dao.HandleUpdateOne(cluster, dao.QueryStruct{p.collectionName, query, 0, 0, ""})
	if err != nil {
		errorCode = CLUSTER_ERROR_UPDATE
		return
	}
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

//delete someone's clusters
func (p *ClusterService) DeleteByUserId(userId string, token string) (errorCode string, err error) {
	logrus.Infof("start to delete Cluster with userid [%v]", userId)

	if len(userId) == 0 {
		errorCode := COMMON_ERROR_INVALIDATE
		err = errors.New("Invalid parameter userid")
		return errorCode, err
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
		errorCode = CLUSTER_ERROR_QUERY
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
		//call delete cluster
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
		return errorCode, err
	}
	if !bson.IsObjectIdHex(clusterId) {
		err = errors.New("Invalid cluster id.")
		errorCode = COMMON_ERROR_INVALIDATE
		return errorCode, err
	}

	//query cluster
	cluster, errorCode, err := p.QueryById(clusterId, x_auth_token)
	if err != nil {
		logrus.Errorf("query cluster error is %v", err)
		return errorCode, err
	}

	//check status
	switch cluster.Status {
	case CLUSTER_STATUS_DEPLOYING, CLUSTER_STATUS_TERMINATING, CLUSTER_STATUS_TERMINATED:
		logrus.Errorf("Cannot operate on a %s cluster", cluster.Status)
		return CLUSTER_ERROR_INVALID_STATUS, errors.New("Cannot operate on a " + cluster.Status + " cluster")

	case CLUSTER_STATUS_RUNNING, CLUSTER_STATUS_FAILED:
		//query all hosts
		var total int
		total, hosts, errorCode, err := GetHostService().QueryHosts(clusterId, 0, 0, "unterminated", x_auth_token)
		if err != nil {
			logrus.Errorf("query hosts in cluster %s error is %v", clusterId, err)
			return errorCode, err
		}

		successNode := 0
		directDeletedHosts := 0
		//set status of all hosts TERMINATING
		for _, host := range hosts {

			//no host name means current host has not been created by docker-machine
			if len(strings.TrimSpace(host.HostName)) <= 0 {
				hostId := host.ObjectId.Hex()
				logrus.Warnf("cluster[%s] host [%s] has no hostname, will be terminated directly", cluster.Name, hostId)
				_, _, err := GetHostService().UpdateStatusById(hostId, HOST_STATUS_TERMINATED, x_auth_token)
				if err != nil {
					logrus.Warnf("set no hostname host status to termianted error %v", err)
				}

				directDeletedHosts++
				continue
			}

			//deploying and terminating host is not allowed to be terminated
			if host.Status == HOST_STATUS_DEPLOYING || host.Status == HOST_STATUS_TERMINATING {
				logrus.Errorf("status of host [%v] is [%s], cluster can not be deleted!", host.HostName, host.Status)
				return CLUSTER_ERROR_DELETE_NOT_ALLOWED, errors.New("cannot delete cluster because not all nodes are ready")
			}

			//set host status to termianting
			_, _, err = GetHostService().UpdateStatusById(host.ObjectId.Hex(), HOST_STATUS_TERMINATING, x_auth_token)
			if err != nil {
				logrus.Warnf("delete host [objectId=%v] error is %v", host.ObjectId.Hex(), err)
			} else {
				successNode++
			}
		}

		logrus.Infof("Cluster %s has %d hosts, %d successfully terminating", cluster.Name, total, successNode)

		selector := bson.M{}
		selector["_id"] = cluster.ObjectId
		change := bson.M{"status": CLUSTER_STATUS_TERMINATING, "time_update": dao.GetCurrentTime()}
		if directDeletedHosts > 0 {
			logrus.Infof("update cluster instances - %d", directDeletedHosts)
			newvalue := cluster.Instances - directDeletedHosts
			change["instances"] = newvalue
		}
		logrus.Debugf("update cluster status and instance bson[%v]", change)
		erro := dao.HandleUpdateByQueryPartial(p.collectionName, selector, change)
		if erro != nil {
			logrus.Errorf("update cluster with objectId [%v] failed, error is %v", clusterId, erro)
		}

		if IsDeploymentEnabled() {
			//call deployment module API
			go DeleteCluster(cluster, x_auth_token)
		}
	default:
		logrus.Errorf("Unknown cluster status %s", cluster.Status)
		return CLUSTER_ERROR_INVALID_STATUS, errors.New("Unknown cluster status " + cluster.Status)
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
		return
	}

	number, err := strconv.ParseInt(numberStr, 10, 0)
	if err != nil {
		logrus.Errorf("Parse number [objectId=%v] error is %v", numberStr, err)
		errorCode = CLUSTER_ERROR_INVALID_NUMBER
		return
	}

	if number <= 0 {
		//call terminate hosts to do this
		errorCode := CLUSTER_ERROR_INVALID_NUMBER
		err = errors.New("Invalid number, it should be positive")
		return cluster, errorCode, err
	}

	newHosts := []entity.Host{}
	for i := 0; i < int(number); i++ {
		host := entity.Host{}
		host.ClusterId = clusterId
		host.ClusterName = cluster.Name
		host.Status = HOST_STATUS_DEPLOYING
		//insert info to db
		newHost, _, err := GetHostService().Create(host, x_auth_token)
		newHosts = append(newHosts, newHost)
		if err != nil {
			logrus.Errorf("creat host error is %v", err)
		}
	}

	if IsDeploymentEnabled() {
		//call deployment module to add nodes
		go AddNodes(cluster, int(number), newHosts, x_auth_token)
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
		errorCode = COMMON_ERROR_INVALIDATE
		err = errors.New("Empty array of host id")
		return errorCode, err
	}

	//query cluster by clusterId
	cluster := entity.Cluster{}
	clusterSelector := bson.M{}
	clusterSelector["_id"] = bson.ObjectIdHex(clusterId)
	err = dao.HandleQueryOne(&cluster, dao.QueryStruct{p.collectionName, clusterSelector, 0, 0, ""})

	_, currentHosts, errorCode, err := GetHostService().QueryHosts(clusterId, 0, 0, HOST_STATUS_RUNNING, x_auth_token)
	if err != nil {
		logrus.Errorf("get host by clusterId[%v] error [%v]", clusterId, err)
		return errorCode, err
	}

	if !deletable(currentHosts, hostIds) {
		logrus.Errorf("cluster's running node should not less than 5 nodes!")
		return CLUSTER_ERROR_DELETE_NODE_NUM, errors.New("cluster's running node should not less than 5 nodes!")
	}

	hosts := []entity.Host{}
	originStatus := make(map[string]string)
	directDeletedHosts := 0
	for _, hostId := range hostIds {
		//query host
		host, errorCode, err := GetHostService().QueryById(hostId, x_auth_token)
		if err != nil {
			return errorCode, err
		}

		//no host name means current host has not been created by docker-machine
		if len(strings.TrimSpace(host.HostName)) <= 0 {
			logrus.Warnf("host has no hostname, will be terminated directly, hostid: %s", hostId)
			_, _, err := GetHostService().UpdateStatusById(hostId, HOST_STATUS_TERMINATED, x_auth_token)
			if err != nil {
				logrus.Warnf("set no hostname host[%s] status to termianted error %v", hostId, err)
			}
			directDeletedHosts++
			continue
		}

		hosts = append(hosts, host)

		//protect master node
		if host.IsMasterNode {
			return HOST_ERROR_DELETE_MASTER, errors.New("Cannot delete master node")
		}

		originStatus[host.ObjectId.Hex()] = host.Status
		//call API to terminate host(master node cannot be deleted now)
		_, errorCode, err = GetHostService().UpdateStatusById(hostId, HOST_STATUS_TERMINATING, x_auth_token)
		if err != nil {
			logrus.Errorf("terminate host error is %s,%v", errorCode, err)
			continue
		}
	}

	if directDeletedHosts > 0 {
		logrus.Infof("update cluster instances - %d", directDeletedHosts)
		newvalue := cluster.Instances - directDeletedHosts
		selector := bson.M{}
		selector["_id"] = cluster.ObjectId

		change := bson.M{"instances": newvalue, "time_update": dao.GetCurrentTime()}
		erro := dao.HandleUpdateByQueryPartial(p.collectionName, selector, change)
		if erro != nil {
			logrus.Errorf("update cluster with objectId [%v] instances to [%d] failed, error is %v", clusterId, newvalue, erro)
		}
	}

	if len(hosts) <= 0 {
		logrus.Infof("no valid hosts will be deleted!")
		return
	}

	if IsDeploymentEnabled() {
		//call deployment module to delete nodes
		go DeleteNodes(cluster, hosts, originStatus, x_auth_token)
	}

	return
}

func deletable(hosts []entity.Host, hostIds []string) bool {
	if hosts == nil {
		return false
	}

	runningHost := 0
	for i := 0; i < len(hosts); i++ {
		host := hosts[i]
		if host.Status == HOST_STATUS_RUNNING && !StringInSlice(host.ObjectId.Hex(), hostIds) {
			runningHost += 1
		}
	}

	return runningHost >= 5
}
