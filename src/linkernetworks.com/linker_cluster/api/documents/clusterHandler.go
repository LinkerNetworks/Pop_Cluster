package documents

import (
	"encoding/json"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"linkernetworks.com/linker_cluster/services"
	"linkernetworks.com/linker_common_lib/persistence/entity"
	"linkernetworks.com/linker_common_lib/rest/response"
)

const (
	ParamHostID string = "host_id"
)

func (p Resource) ClusterWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/cluster")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	id := ws.PathParameter(ParamID, "Storage identifier of cluster")
	number := ws.QueryParameter("number", "Change the nubmer of node for a cluster")
	paramID := "{" + ParamID + "}"
	paramHostID := "{" + ParamHostID + "}"

	ws.Route(ws.POST("/").To(p.ClusterCreateHandler).
		Doc("Store a cluster").
		Operation("ClusterCreateHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.GET("/").To(p.ClustersListHandler).
		Doc("Returns all cluster items").
		Operation("ClustersListHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
		Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
		Param(ws.QueryParameter("name", "The name of cluster wanted to query")).
		Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")).
		Param(ws.QueryParameter("sort", "Comma separated list of field names to sort")).
		Param(ws.QueryParameter("user_id", "The owner ID of the cluster")).
		Param(ws.QueryParameter("status", "DEPLOYED,RUNNING,FAILED,TERMINATED,unterminated. Query all clusters by default if not provided")))

	ws.Route(ws.GET("/" + paramID).To(p.ClusterGetHandler).
		Doc("Return a cluster").
		Operation("ClusterGetHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	ws.Route(ws.DELETE("/" + paramID).To(p.ClusterDeleteHandler).
		Doc("Detele a Cluster by its storage identifier").
		Operation("ClusterDeleteHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	ws.Route(ws.DELETE("/").To(p.ClustersDeleteHandler).
		Doc("Detele cluster by their storage identifiers").
		Operation("ClustersDeleteHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.DELETE("/").To(p.UserClusterDeleteHandler).
		Doc("Detele all clusters created by a specific user").
		Operation("ClusterDeleteByQueryHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("user_id", "Storage identifier of an user ")).
		Param(id))

	ws.Route(ws.POST("/" + paramID + "/email").To(p.EmailSendHandler).
		Doc("Send cluster owner an email of endpoint.").
		Operation("EmailSendHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	ws.Route(ws.POST("/" + paramID + "/hosts").To(p.HostsAddHandler).
		Doc("Add hosts for a cluster").
		Operation("HostsAddHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id).
		Param(number))

	ws.Route(ws.DELETE("/" + paramID + "/hosts").To(p.HostsDeleteHandler).
		Doc("Terminate hosts of a cluster").
		Operation("HostsDeleteHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.BodyParameter("body", "").DataType("string")).
		Param(id))

	ws.Route(ws.GET("/" + paramID + "/hosts").To(p.HostsListHandler).
		Doc("List hosts of a cluster").
		Operation("HostsListHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
		Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
		Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")).
		Param(ws.QueryParameter("sort", "Comma separated list of field names to sort")).
		Param(ws.QueryParameter("status", "DEPLOYED,RUNNING,FAILED,TERMINATED,unterminated. Query all hosts by default if not provided")).
		Param(id))

	ws.Route(ws.GET("/" + paramID + "/hosts/" + paramHostID).To(p.HostGetHandler).
		Doc("List detail of a host").
		Operation("HostGetHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.PathParameter(ParamHostID, "Storage identifier of host")).
		Param(id))

	//pre-check if cluster name matches with regex, and if it is conflict
	ws.Route(ws.GET("/validate").To(p.ClusterNameCheckHandler).
		Doc("Check cluster name").
		Operation("ClusterNameCheckHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.QueryParameter("userid", "Storage identifier of user")).
		Param(ws.QueryParameter("clustername", "Name of cluster")))

	return ws

}

//check username and clustername
func (p *Resource) ClusterNameCheckHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("ClusterNameCheckHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	code, err := services.TokenValidation(x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	userId := req.QueryParameter("userid")
	clusterName := req.QueryParameter("clustername")

	errorCode, err := services.GetClusterService().CheckClusterName(userId, clusterName, x_auth_token)

	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	// Write success response
	response.WriteSuccess(resp)
	return
}

//Send cluster owner an email of endpoint
func (p *Resource) EmailSendHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("EmailSendHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	code, err := services.TokenValidation(x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	clusterId := req.PathParameter(ParamID)

	errorCode, err := services.GetEmailService().SendClusterDeployedEmail(clusterId, x_auth_token)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	// Write success response
	response.WriteSuccess(resp)
	return
}

func (p *Resource) ClusterDeleteHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("ClusterDeleteHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	code, err := services.TokenValidation(x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	objectId := req.PathParameter(ParamID)

	code, err = services.GetClusterService().DeleteById(objectId, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	// Write success response
	response.WriteSuccess(resp)
	return
}

type DeleteClustersRequestBody struct {
	ClusterIds []string `json:"cluster_ids"`
}

func (p *Resource) ClustersDeleteHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("UserClusterDeleteHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	code, err := services.TokenValidation(x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	body := DeleteClustersRequestBody{}
	err = json.NewDecoder(req.Request.Body).Decode(&body)

	errorCode, err := services.GetClusterService().DeleteClusters(body.ClusterIds, x_auth_token)
	if err != nil {
		logrus.Errorln("delete clusters error , [%v]", err)
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	// Write success response
	response.WriteSuccess(resp)
	return
}

func (p *Resource) UserClusterDeleteHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("UserClusterDeleteHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	code, err := services.TokenValidation(x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	userId := req.QueryParameter("user_id")

	code, err = services.GetClusterService().DeleteByUserId(userId, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	// Write success response
	response.WriteSuccess(resp)
	return
}

func (p *Resource) ClusterCreateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("ClusterCreateHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	code, err := services.TokenValidation(x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	// Stub an acluster to be populated from the body
	cluster := entity.Cluster{}

	err = json.NewDecoder(req.Request.Body).Decode(&cluster)
	if err != nil {
		logrus.Errorf("convert body to cluster failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	newCluster, code, err := services.GetClusterService().Create(cluster, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: newCluster}
	resp.WriteEntity(res)
	return

}

func (p *Resource) ClustersListHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("ClustersListHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	code, err := services.TokenValidation(x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	var skip int = queryIntParam(req, "skip", 0)
	var limit int = queryIntParam(req, "limit", 0)

	var name string = req.QueryParameter("name")
	var user_id string = req.QueryParameter("user_id")
	var status string = req.QueryParameter("status")
	var sort string = req.QueryParameter("sort")

	total, clusters, code, err := services.GetClusterService().QueryCluster(name, user_id, status, skip, limit, sort, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	res := response.QueryStruct{Success: true, Data: clusters}
	if c, _ := strconv.ParseBool(req.QueryParameter("count")); c {
		res.Count = total
		resp.AddHeader("X-Object-Count", strconv.Itoa(total))
	}
	resp.WriteEntity(res)
	return

}

func (p *Resource) ClusterGetHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("ClusterGetHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	code, err := services.TokenValidation(x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	objectId := req.PathParameter(ParamID)
	cluster, code, err := services.GetClusterService().QueryById(objectId, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	logrus.Debugf("cluster is %v", cluster)

	res := response.QueryStruct{Success: true, Data: cluster}
	resp.WriteEntity(res)
	return

}

func (p *Resource) HostsAddHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("HostsAddHandler is called!")
	number := req.QueryParameter("number")
	logrus.Infof("Number is %s \n", number)

	x_auth_token := req.HeaderParameter("X-Auth-Token")
	code, err := services.TokenValidation(x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	objectId := req.PathParameter(ParamID)
	logrus.Debugf("cluster id is %d \n", objectId)

	cluster, errorCode, err := services.GetClusterService().AddHosts(objectId, number, x_auth_token)

	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: cluster}
	resp.WriteEntity(res)
	return
}

type TerminateHostsRequestBody struct {
	HostIds []string `json:"host_ids"`
}

//terminate specified hosts of a cluster
// Request
// URL:
// 	PUT /v1/cluster/<CLUSTER_ID>/hosts
// Header:
// 	X-Auth-Token
// Except Body:
//{
//    "host_ids":["568e23655d5c3d173019f1ba","568e2be45d5c3d173019f1bb","568e2bfd5d5c3d173019f1bc","568e2c335d5c3d173019f1bd"]
//}
//
// Response:
//{
//  "success": true
//}
//
func (p *Resource) HostsDeleteHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("HostsDeleteHandler is called!")

	x_auth_token := req.HeaderParameter("X-Auth-Token")
	code, err := services.TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorln("token validation error is %v", err)
		response.WriteStatusError(code, err, resp)
		return
	}

	clusterId := req.PathParameter(ParamID)
	body := TerminateHostsRequestBody{}
	err = json.NewDecoder(req.Request.Body).Decode(&body)

	errorCode, err := services.GetClusterService().TerminateHosts(clusterId, body.HostIds, x_auth_token)
	if err != nil {
		logrus.Errorln("terminate hosts error is %v", err)
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	// Write success response
	response.WriteSuccess(resp)
	return
}

func (p *Resource) HostsListHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("HostsListHandler is called!")

	x_auth_token := req.HeaderParameter("X-Auth-Token")
	code, err := services.TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorln("token validation error is %v", err)
		response.WriteStatusError(code, err, resp)
		return
	}

	clusterId := req.PathParameter(ParamID)

	var skip, limit int64
	if param_skip := req.QueryParameter("skip"); len(param_skip) > 0 {
		skip, err = strconv.ParseInt(param_skip, 10, 0)
		if err != nil {
			response.WriteStatusError("E12002", err, resp)
			return
		}
	}
	if param_limit := req.QueryParameter("limit"); len(param_limit) > 0 {
		limit, err = strconv.ParseInt(req.QueryParameter("limit"), 10, 0)
		if err != nil {
			response.WriteStatusError("E12002", err, resp)
			return
		}
	}

	var status string = req.QueryParameter("status")

	total, hosts, errorCode, err := services.GetHostService().QueryHosts(clusterId, int(skip), int(limit), status, x_auth_token)

	if err != nil {
		logrus.Errorln("list hosts error is %v", err)
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: hosts}
	if c, _ := strconv.ParseBool(req.QueryParameter("count")); c {
		res.Count = total
		resp.AddHeader("X-Object-Count", strconv.Itoa(total))
	}
	resp.WriteEntity(res)
	return
}

func (p *Resource) HostGetHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("HostGetHandler is called!")

	x_auth_token := req.HeaderParameter("X-Auth-Token")
	code, err := services.TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorln("token validation error is %v", err)
		response.WriteStatusError(code, err, resp)
		return
	}

	hostId := req.PathParameter(ParamHostID)

	host, code, err := services.GetHostService().QueryById(hostId, x_auth_token)
	if err != nil {
		logrus.Errorln("get host error is %v", err)
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: host}
	resp.WriteEntity(res)
	return
}
