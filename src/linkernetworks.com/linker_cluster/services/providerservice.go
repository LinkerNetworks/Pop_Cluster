package services

import (
	"errors"
	"sync"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"

	"linkernetworks.com/linker_common_lib/persistence/dao"
	"linkernetworks.com/linker_common_lib/persistence/entity"
)

const (
	PROVIDER_ERROR_CREATE string = "E51000"
	PROVIDER_ERROR_UPDATE string = "E51001"
	PROVIDER_ERROR_DELETE string = "E51002"
	PROVIDER_ERROR_QUERY  string = "E52003"

	EC2_TYPE       = "amazonec2"
	OPENSTACK_TYPE = "openstack"
)

var (
	providerService *ProviderService = nil
	onceProvider    sync.Once
)

type ProviderService struct {
	collectionName string
}

func GetProviderService() *ProviderService {
	onceProvider.Do(func() {
		logrus.Debugf("Once called from providerService ......................................")
		providerService = &ProviderService{"provider"}
	})
	return providerService
}

func (p *ProviderService) Create(provider entity.IaaSProvider, token string) (newProvider entity.IaaSProvider, errorCode string, err error) {
	logrus.Infof("create provider [%v]", provider)
	errorCode, err = TokenValidation(token)
	if err != nil {
		logrus.Errorf("token validation failed for provider creation [%v]", err)
		return
	}

	// do authorize first
	if authorized := GetAuthService().Authorize("create_provider", token, "", p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("create provider [%v] error is %v", provider, err)
		return
	}

	userId := provider.UserId
	if len(userId) == 0 {
		err = errors.New("user_id not provided")
		errorCode = COMMON_ERROR_INVALIDATE
		logrus.Errorf("create provider [%v] error is %v", provider, err)
		return
	}

	user, err := GetUserById(userId, token)
	if err != nil {
		logrus.Errorf("get user by id err is %v", err)
		errorCode = COMMON_ERROR_INTERNAL
		return newProvider, errorCode, err
	}

	provider.ObjectId = bson.NewObjectId()
	provider.TenantId = user.TenantId
	// set created_time and updated_time
	provider.TimeCreate = dao.GetCurrentTime()
	provider.TimeUpdate = provider.TimeCreate

	err = dao.HandleInsert(p.collectionName, provider)
	if err != nil {
		errorCode = PROVIDER_ERROR_CREATE
		logrus.Errorf("create provider [%v] to bson error is %v", provider, err)
		return
	}

	newProvider = provider

	return

}

func (p *ProviderService) QueryProvider(providerType string, userId string, skip int, limit int, sort string, token string) (total int, providers []entity.IaaSProvider,
	errorCode string, err error) {
	query := bson.M{}
	if len(providerType) > 0 {
		query["type"] = providerType
	}
	if !bson.IsObjectIdHex(userId) {
		logrus.Errorf("not a valid object id for query provider operation! userId: %s", userId)
	} else {
		query["user_id"] = userId
	}

	return p.queryByQuery(query, skip, limit, sort, token, false)
}

func (p *ProviderService) QueryById(objectId string, token string) (provider entity.IaaSProvider, errorCode string, err error) {
	logrus.Infof("query provider by id[%s]", objectId)
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	errorCode, err = TokenValidation(token)
	if err != nil {
		logrus.Errorf("token validation failed for provider get [%v]", err)
		return
	}

	// do authorize first
	if authorized := GetAuthService().Authorize("get_provider", token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("get provider with objectId [%v] error is %v", objectId, err)
		return
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)
	provider = entity.IaaSProvider{}
	err = dao.HandleQueryOne(&provider, dao.QueryStruct{p.collectionName, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("query provider [objectId=%v] error is %v", objectId, err)
		errorCode = PROVIDER_ERROR_QUERY
	}
	return

}

func (p *ProviderService) queryByQuery(query bson.M, skip int, limit int, sort string,
	x_auth_token string, skipAuth bool) (total int, providers []entity.IaaSProvider,
	errorCode string, err error) {
	authQuery := bson.M{}
	if !skipAuth {
		// get auth query from auth service first
		authQuery, err = GetAuthService().BuildQueryByAuth("list_provider", x_auth_token)
		if err != nil {
			logrus.Errorf("get auth query by token [%v] error is %v", x_auth_token, err)
			errorCode = COMMON_ERROR_INTERNAL
			return
		}
	}

	selector := generateQueryWithAuth(query, authQuery)
	providers = []entity.IaaSProvider{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           skip,
		Limit:          limit,
		Sort:           sort,
	}
	total, err = dao.HandleQueryAll(&providers, queryStruct)
	if err != nil {
		logrus.Errorf("query providers by query [%v] error is %v", query, err)
		errorCode = PROVIDER_ERROR_QUERY
		return
	}
	return
}
