package documents

import (
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"linkernetworks.com/linker_common_lib/rest/response"
	"linkernetworks.com/linker_common_lib/entity"
	"linkernetworks.com/linker_dcos_deploy/services"
)

func (p Resource) DeployWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/deploy")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	// id := ws.PathParameter(ParamID, "Storage identifier of cluster")
	// number := ws.QueryParameter("number", "Change the nubmer of node for a cluster")
	// paramID := "{" + ParamID + "}"

	ws.Route(ws.POST("/").To(p.CreateClusterHandler).
		Doc("create a cluster").
		Operation("CreateClusterHandler").
		Param(ws.BodyParameter("body", "").DataType("string")))
		
	ws.Route(ws.DELETE("/").To(p.DeleteClusterHandler).
		Doc("delete a cluster").
		Operation("DeleteClusterHandler").
		Param(ws.BodyParameter("body", "").DataType("string")))
		
	ws.Route(ws.POST("/nodes").To(p.AddNodesHandler).
		Doc("add nodes").
		Operation("AddNodesHandler").
		Param(ws.BodyParameter("body", "").DataType("string")))
		
	ws.Route(ws.DELETE("/nodes").To(p.DeleteNodesHandler).
		Doc("delete nodes").
		Operation("DeleteNodesHandler").
		Param(ws.BodyParameter("body", "").DataType("string")))

	return ws

}

func (p *Resource) CreateClusterHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("CreateClusterHandler is called!")

	// Stub an repairpolicy to be populated from the body
	request := entity.Request{}

	// Populate the user data
	err := json.NewDecoder(req.Request.Body).Decode(&request)
	
	logrus.Infof("Request is %v", request)
	logrus.Infof("ProviderInfo is %v", request.ProviderInfo)
	logrus.Infof("AwsEC2Info is %v", request.ProviderInfo.AwsEC2Info)

	if err != nil {
		logrus.Errorf("convert body to request failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	servers, code, err := services.GetDeployService().CreateCluster(request)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.Response{Success: true, Data: servers}
	resp.WriteEntity(res)
	return
}

func (p *Resource) DeleteClusterHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("DeleteClusterHandler is called!")

	// Stub an repairpolicy to be populated from the body
	request := entity.DeleteRequest{}

	// Populate the user data
	err := json.NewDecoder(req.Request.Body).Decode(&request)
	
	logrus.Infof("Username is %v", request.UserName)
	logrus.Infof("Cluster is %v", request.ClusterName)
	logrus.Infof("Servers is %v", request.Servers)

	if err != nil {
		logrus.Errorf("convert body to DeleteClusterRequest failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	code, err := services.GetDeployService().DeleteCluster(request.UserName, request.ClusterName, request.Servers)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.Response{Success: true}
	resp.WriteEntity(res)
	return
}

func (p *Resource) AddNodesHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("AddNodesHandler is called!")

	// Stub an repairpolicy to be populated from the body
	request := entity.AddNodeRequest{}

	// Populate the user data
	err := json.NewDecoder(req.Request.Body).Decode(&request)
	
	logrus.Infof("Username is %v", request.UserName)
	logrus.Infof("Cluster is %v", request.ClusterName)
	logrus.Infof("CreateNumber is %v", request.CreateNumber)

	if err != nil {
		logrus.Errorf("convert body to AddNodesRequest failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	servers, code, err := services.GetDeployService().CreateNode(request)

	var res response.Response
	if err != nil {
		errObj := response.Error{Code: code, ErrorMsg: fmt.Sprintf("%v", err)}
		res = response.Response{Success: true, Error: &errObj, Data: servers}
	} else {
		res = response.Response{Success: true, Data: servers}
	}

	resp.WriteEntity(res)
	return
}

func (p *Resource) DeleteNodesHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("DeleteNodesHandler is called!")

	// Stub an repairpolicy to be populated from the body
	request := entity.DeleteRequest{}

	// Populate the user data
	err := json.NewDecoder(req.Request.Body).Decode(&request)

	logrus.Infof("Username is %v", request.UserName)
	logrus.Infof("Cluster is %v", request.ClusterName)
	logrus.Infof("Servers is %v", request.Servers)

	if err != nil {
		logrus.Errorf("convert body to DeleteClusterRequest failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	slaves, _, _ := services.GetDeployService().DeleteNode(request.UserName, request.ClusterName, request.Servers)
	res := response.Response{Success: true, Data: slaves}

	resp.WriteEntity(res)
	return
}
