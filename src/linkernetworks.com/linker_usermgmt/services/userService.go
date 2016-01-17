package services

import (
	"encoding/json"
	"errors"
	//	"os"
	//	"strconv"
	"strings"
	"sync"
	//	"time"

	"github.com/Sirupsen/logrus"
	"github.com/compose/mejson"
	"gopkg.in/mgo.v2/bson"
	"linkernetworks.com/linker_common_lib/persistence/dao"
	"linkernetworks.com/linker_common_lib/persistence/entity"
)

var sysadmin_user = "sysadmin"
var sysadmin_pass = "password"
var sys_tenant = "sysadmin"
var sys_admin_role = "sysadmin"
var admin_role = "admin"
var common_role = "common"

var USER_ERROR_REG = "E10000"
var USER_ERROR_EXCEED = "E10001"
var USER_ERROR_CREATE = "E10003"
var USER_ERROR_NOEXIST = "E10004"
var USER_ERROR_WRONGPW = "E10005"
var USER_ERROR_UPDATE = "E10007"
var USER_ERROR_EXIST = "E10009"
var USER_ERROR_DELETE = "E10010"
var USER_ERROR_GET = "E10011"
var USER_ERROR_LOGIN = "E10012"
var USER_ERROR_EXISTCLUSTER = "E10013"

var userService *UserService = nil
var userOnce sync.Once

type UserService struct {
	userCollectionName string
}

func GetUserService() *UserService {
	userOnce.Do(func() {
		userService = &UserService{"user"}

		userService.initialize()
	})

	return userService
}

func (p *UserService) initialize() bool {
	logrus.Infoln("UserMgmt initialize...")

	logrus.Infoln("check sysadmin tenant")

	sysTenantId, tenantErr := GetTenantService().createAndInsertTenant(sys_tenant, "system admin tenant")
	if tenantErr != nil {
		logrus.Errorf("create and insert sys admin tenant error,  err is %v", tenantErr)
		return false
	}

	logrus.Infoln("check sysadmin role")
	sysRoleId, roleErr := GetRoleService().createAndInsertRole(sys_admin_role, "sysadmin role")
	if roleErr != nil {
		logrus.Errorf("create and insert sys admin role error,  err is %v", roleErr)
		return false
	}

	logrus.Infoln("check admin role")
	_, roleErr = GetRoleService().createAndInsertRole(admin_role, "admin role")
	if roleErr != nil {
		logrus.Errorf("create and insert admin role error,  err is %v", roleErr)
		return false
	}

	logrus.Infoln("check common role")
	_, roleErr = GetRoleService().createAndInsertRole(common_role, "common role")
	if roleErr != nil {
		logrus.Errorf("create and insert common role error,  err is %v", roleErr)
		return false
	}

	logrus.Infoln("check sysadmin user")
	encryPassword := HashString(sysadmin_pass)
	_, userErr := p.createAndInsertUser(sysadmin_user, encryPassword, sysadmin_user, sysTenantId, sysRoleId, "")
	if userErr != nil {
		logrus.Errorf("create and insert sysadmin user error,  err is %v", userErr)
		return false
	}

	return true
}

func (p *UserService) deleteUserById(userId string) (err error) {
	if !bson.IsObjectIdHex(userId) {
		logrus.Errorln("invalid object id for deleteUserById: ", userId)
		err = errors.New("invalid object id for deleteUserById")
		return
	}

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(userId)

	err = dao.HandleDelete(p.userCollectionName, true, selector)
	return
}

func (p *UserService) Validate(username string, token string) (erorCode string, userid string, err error) {
	code, err := GetTokenService().TokenValidate(token)
	if err != nil {
		return code, "", err
	}

	if authorized := GetAuthService().Authorize("get_user", token, "", p.userCollectionName); !authorized {
		logrus.Errorln("required opertion is not allowed!")
		return COMMON_ERROR_UNAUTHORIZED, "", errors.New("Required opertion is not authorized!")
	}

	currentUser, err := p.getAllUserByName(username)

	if err != nil {
		logrus.Errorf("get all user by name err is %v", err)
		return "", "", err
	}
	if len(currentUser) == 0 {
		return "", "", nil
	} else {
		logrus.Infoln("user already exist! username:", username)
		userId := currentUser[0].ObjectId.Hex()
		return USER_ERROR_EXIST, userId, errors.New("user already exist!")
	}

	return
}

func (p *UserService) Create(userParam UserParam, token string) (errorCode string, userId string, err error) {

	if len(userParam.UserName) == 0 || len(userParam.Email) == 0 || len(userParam.Password) == 0 {
		logrus.Error("invalid parameter for user create!")
		return "", COMMON_ERROR_INVALIDATE, errors.New("invalid parameter! parameter should not be null")
	}
	code, err := GetTokenService().TokenValidate(token)
	if err != nil {
		return code, "", err
	}

	if authorized := GetAuthService().Authorize("create_user", token, "", p.userCollectionName); !authorized {
		logrus.Errorln("required opertion is not allowed!")
		return "", COMMON_ERROR_UNAUTHORIZED, errors.New("Required opertion is not authorized!")
	}

	username := userParam.UserName
	email := userParam.Email
	password := userParam.Password
	company := userParam.Company

	_, err = GetTenantService().getTenantByName(username)
	if err == nil {
		logrus.Errorln("user already exist!")
		return USER_ERROR_EXIST, "", errors.New("The username has already been registered, please specified another one!")
	}

	encryPassword := HashString(password)

	tenantId, errte := GetTenantService().createAndInsertTenant(username, username)
	if errte != nil {
		logrus.Errorf("create and insert new tenant error,  err is %v", errte)
		return USER_ERROR_REG, "", errte
	}

	role, errrole := GetRoleService().getRoleByName(admin_role)
	if errrole != nil {
		logrus.Errorf("get role error is %v", errrole)
		return ROLE_ERROR_GET, "", errrole
	}

	userId, err = p.createAndInsertUser(username, encryPassword, email, tenantId, role.ObjectId.Hex(), company)
	if err != nil {
		logrus.Errorf("create and insert new user error,  err is %v", err)
		return USER_ERROR_REG, "", err
	}

	return "", userId, nil
}

func (p *UserService) GetUserByUserId(userId string) (user *entity.User, err error) {
	if !bson.IsObjectIdHex(userId) {
		logrus.Errorln("invalid object id for getUseerById: ", userId)
		err = errors.New("invalid object id for getUserById")
		return nil, err
	}
	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(userId)

	user = new(entity.User)
	queryStruct := dao.QueryStruct{
		CollectionName: p.userCollectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	err = dao.HandleQueryOne(user, queryStruct)

	if err != nil {
		logrus.Warnln("failed to get user by id %v", err)
		return
	}

	return
}

func (p *UserService) UserLogin(username string, password string) (errorCode string, login *entity.LoginResponse, err error) {
	currentAllUser, err := p.getAllUserByName(username)
	if err != nil {
		return "", nil, err
	}
	if len(currentAllUser) == 0 {
		return USER_ERROR_NOEXIST, nil, errors.New("user is not exist")
	}
	currentUser := currentAllUser[0]

	encryPassword := HashString(password)
	if !strings.EqualFold(encryPassword, currentUser.Password) {
		logrus.Errorln("invalid password!")
		return USER_ERROR_WRONGPW, nil, errors.New("Invalid password!")

	}

	tenantid := currentUser.TenantId
	token, err := GetTokenService().checkAndGenerateToken(username, password, tenantid, true)
	if err != nil {
		logrus.Errorf("failed to generate token, error is %s", err)
		return USER_ERROR_LOGIN, nil, err
	}

	var loginRes *entity.LoginResponse
	loginRes = new(entity.LoginResponse)
	loginRes.Id = token
	loginRes.UserId = currentUser.ObjectId.Hex()

	return "", loginRes, nil
}

func (p *UserService) UserUpdate(token string, newuser entity.User, userId string) (created bool, id string, errorCode string, err error) {
	code, err := GetTokenService().TokenValidate(token)
	if err != nil {
		return false, userId, code, err
	}

	if authorized := GetAuthService().Authorize("update_user", token, userId, p.userCollectionName); !authorized {
		logrus.Errorln("required opertion is not allowed!")
		return false, userId, COMMON_ERROR_UNAUTHORIZED, errors.New("Required opertion is not authorized!")
	}

	if !bson.IsObjectIdHex(userId) {
		logrus.Errorf("invalid user id format for user update %v", userId)
		return false, "", COMMON_ERROR_INVALIDATE, errors.New("Invalid object Id for user update")
	}

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(userId)

	queryStruct := dao.QueryStruct{
		CollectionName: p.userCollectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	user := new(entity.User)
	err = dao.HandleQueryOne(user, queryStruct)
	if err != nil {
		logrus.Errorf("get user by id error %v", err)
		return false, "", USER_ERROR_UPDATE, err
	}

	if len(newuser.Company) > 0 {
		user.Company = newuser.Company
	}
	if len(newuser.Email) > 0 {
		user.Email = newuser.Email
	}

	user.TimeUpdate = dao.GetCurrentTime()

	created, err = dao.HandleUpdateOne(user, queryStruct)
	return created, userId, "", nil
}

func (p *UserService) UserDelete(token string, userId string) (errorCode string, err error) {
	if !bson.IsObjectIdHex(userId) {
		logrus.Errorln("invalid object id for UserDelete: ", userId)
		err = errors.New("invalid object id for UserDelete")
		return USER_ERROR_DELETE, err
	}

	code, err := GetTokenService().TokenValidate(token)
	if err != nil {
		return code, err
	}

	if authorized := GetAuthService().Authorize("delete_user", token, userId, p.userCollectionName); !authorized {
		logrus.Errorln("required opertion is not allowed!")
		return COMMON_ERROR_UNAUTHORIZED, errors.New("Required opertion is not authorized!")
	}

	clusters, errquery := GetClusterByUser(userId, token)
	if errquery != nil {
		logrus.Errorf("query cluster err is %v", errquery)
		return "", errors.New("query cluster is err")
	}
	if len(clusters) != 0 {
		logrus.Errorf("user has unterminated cluster")
		return USER_ERROR_EXISTCLUSTER, errors.New("Please terminated cluster first!")
	}

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(userId)

	user, err := p.GetUserById(userId)
	tenantid := user.TenantId

	err = dao.HandleDelete(p.userCollectionName, true, selector)
	if err != nil {
		logrus.Warnln("delete user error %v", err)
		return USER_ERROR_DELETE, err
	}

	err = GetTenantService().deleteTenantById(tenantid)
	if err != nil {
		logrus.Warnln("delete tenant error %v", err)
		return TENANT_ERROR_DELETE, err
	}

	return "", nil
}

func (p *UserService) UserChangePassword(token string, id string, password string, newpassword string, confirm_newpassword string) (created bool, errorCode string, err error) {
	code, err := GetTokenService().TokenValidate(token)
	if err != nil {
		return false, code, err
	}

	if authorized := GetAuthService().Authorize("change_password", token, id, p.userCollectionName); !authorized {
		logrus.Errorln("required opertion is not allowed!")
		return false, COMMON_ERROR_UNAUTHORIZED, errors.New("Required opertion is not authorized!")
	}

	user, err := p.GetUserByUserId(id)
	if err != nil {
		logrus.Errorln("user does exist %v", err)
		return false, COMMON_ERROR_INTERNAL, errors.New("User does not exist!")
	}

	pwdEncry := HashString(password)
	if !strings.EqualFold(pwdEncry, user.Password) {
		logrus.Errorln("incorrect password!")
		return false, USER_ERROR_WRONGPW, errors.New("Incorrect password!")
	}

	if !strings.EqualFold(newpassword, confirm_newpassword) {
		logrus.Errorln("inconsistence new password!")
		return false, USER_ERROR_WRONGPW, errors.New("Inconsistent new password!")
	}

	newpasswordEncry := HashString(newpassword)
	user.Password = newpasswordEncry

	user.TimeUpdate = dao.GetCurrentTime()

	// userDoc := ConvertToBson(*user)
	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(id)

	queryStruct := dao.QueryStruct{
		CollectionName: p.userCollectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	created, err = dao.HandleUpdateOne(user, queryStruct)
	if err != nil {
		logrus.Error("update user password error! %v", err)
		return created, USER_ERROR_UPDATE, err
	}

	return created, "", nil
}

func (p *UserService) UserList(token string, limit int, skip int, sort string) (ret []entity.User, count int, errorCode string, err error) {
	code, err := GetTokenService().TokenValidate(token)
	if err != nil {
		return nil, 0, code, err
	}

	query, err := GetAuthService().BuildQueryByAuth("list_users", token)
	if err != nil {
		logrus.Error("auth failed during query all user: %v", err)
		return nil, 0, USER_ERROR_GET, err
	}

	result := []entity.User{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.userCollectionName,
		Selector:       query,
		Skip:           skip,
		Limit:          limit,
		Sort:           sort}
	count, err = dao.HandleQueryAll(&result, queryStruct)

	return result, count, "", err
}

func (p *UserService) UserDetail(token string, userId string) (ret interface{}, errorCode string, err error) {
	code, err := GetTokenService().TokenValidate(token)
	if err != nil {
		return nil, code, err
	}

	if authorized := GetAuthService().Authorize("get_user", token, userId, p.userCollectionName); !authorized {
		logrus.Errorln("required opertion is not allowed!")
		return nil, COMMON_ERROR_UNAUTHORIZED, errors.New("Required opertion is not authorized!")
	}

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(userId)

	ret = new(entity.User)
	queryStruct := dao.QueryStruct{
		CollectionName: p.userCollectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	err = dao.HandleQueryOne(ret, queryStruct)
	logrus.Errorln(ret)
	return
}

func (p *UserService) createAndInsertUser(userName string, password string, email string, tenanId string, roleId string, company string) (userId string, err error) {
	// var jsondocument interface{}
	currentUser, erro := p.getAllUserByName(userName)
	if erro != nil {
		logrus.Error("get all user by username err is %v", erro)
		return "", erro
	}
	if len(currentUser) != 0 {
		logrus.Infoln("user already exist! username:", userName)
		userId = currentUser[0].ObjectId.Hex()
		return
	}

	currentTime := dao.GetCurrentTime()
	user := new(entity.User)
	user.ObjectId = bson.NewObjectId()
	user.Username = userName
	user.Password = password
	user.TenantId = tenanId
	user.RoleId = roleId
	user.Email = email
	user.Company = company
	user.TimeCreate = currentTime
	user.TimeUpdate = currentTime

	err = dao.HandleInsert(p.userCollectionName, user)
	if err != nil {
		logrus.Warnln("create user error %v", err)
		return
	}
	userId = user.ObjectId.Hex()

	return
}

func (p *UserService) getAllUserByName(username string) (user []entity.User, err error) {
	query := strings.Join([]string{"{\"username\": \"", username, "\"}"}, "")

	selector := make(bson.M)
	err = json.Unmarshal([]byte(query), &selector)
	if err != nil {
		return
	}
	selector, err = mejson.Unmarshal(selector)
	if err != nil {
		return
	}

	user = []entity.User{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.userCollectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	_, err = dao.HandleQueryAll(&user, queryStruct)

	return
}

func (p *UserService) GetUserById(userid string) (currentUser *entity.User, err error) {
	validId := bson.IsObjectIdHex(userid)
	if !validId {
		return nil, errors.New("invalid token!")
	}

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(userid)

	currentUser = new(entity.User)
	queryStruct := dao.QueryStruct{
		CollectionName: p.userCollectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	err = dao.HandleQueryOne(currentUser, queryStruct)
	if err != nil {
		logrus.Infoln("user does not exist! %v", err)
		return nil, err
	}

	return
}
