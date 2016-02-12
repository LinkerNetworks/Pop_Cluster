package documents

import (
	"encoding/json"

	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"linkernetworks.com/linker_cluster/services"
	"linkernetworks.com/linker_common_lib/persistence/entity"
	"linkernetworks.com/linker_common_lib/rest/response"
)

func (p Resource) ProviderWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/provider")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	id := ws.PathParameter(ParamID, "Storage identifier of provider")
	paramID := "{" + ParamID + "}"

	ws.Route(ws.POST("/").To(p.ProviderCreateHandler).
		Doc("Store a provider").
		Operation("ProviderCreateHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.BodyParameter("body", "").DataType("string")))

	// ws.Route(ws.GET("/").To(p.ProviderListHandler).
	// 	Doc("Returns all provider items").
	// 	Operation("ProviderListHandler").
	// 	Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
	// 	Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
	// 	Param(ws.QueryParameter("type", "IaaS provider type: amazonec2, openstack")).
	// 	Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
	// 	Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")).
	// 	Param(ws.QueryParameter("sort", "Comma separated list of field names to sort")).
	// 	Param(ws.QueryParameter("user_id", "The owner ID of the provider")))

	ws.Route(ws.GET("/" + paramID).To(p.ProviderGetHandler).
		Doc("Return a provider").
		Operation("ProviderGetHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	// ws.Route(ws.DELETE("/" + paramID).To(p.ProviderDeleteHandler).
	// 	Doc("Detele a provider by its storage identifier").
	// 	Operation("ProviderDeleteHandler").
	// 	Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
	// 	Param(id))

	return ws

}

func (p *Resource) ProviderCreateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("ProviderCreateHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	// Stub an acluster to be populated from the body
	provider := entity.IaaSProvider{}

	err := json.NewDecoder(req.Request.Body).Decode(&provider)
	if err != nil {
		logrus.Errorf("convert body to provider failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	newProvider, code, err := services.GetProviderService().Create(provider, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: newProvider}
	resp.WriteEntity(res)
	return
}

func (p *Resource) ProviderGetHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("ProviderGetHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")

	objectId := req.PathParameter(ParamID)
	// cluster, code, err := services.GetClusterService().QueryById(objectId, x_auth_token)
	provider, errorCode, err := services.GetProviderService().QueryById(objectId, x_auth_token)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	logrus.Debugf("provider is %v", provider)

	res := response.QueryStruct{Success: true, Data: provider}
	resp.WriteEntity(res)
	return
}
