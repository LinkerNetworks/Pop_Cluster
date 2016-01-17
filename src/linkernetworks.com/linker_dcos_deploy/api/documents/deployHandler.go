package documents

import (
	"encoding/json"
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
