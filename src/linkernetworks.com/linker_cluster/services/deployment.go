package services

import (
	"errors"
	"strings"

	"github.com/Sirupsen/logrus"

	dcosentity "linkernetworks.com/linker_common_lib/entity"
	"linkernetworks.com/linker_common_lib/persistence/entity"
)

//static config
var (
	providerInfo dcosentity.ProviderInfo = dcosentity.ProviderInfo{
		Provider:      provider,
		OpenstackInfo: dcosentity.Openstack{},
		AwsEC2Info:    awsEC2Info,
	}
	provider dcosentity.Provider = dcosentity.Provider{
		ProviderType: "amazonec2",
		SshUser:      "ec2-user",
	}
	awsEC2Info dcosentity.AwsEC2 = dcosentity.AwsEC2{
		AccessKey:    "AKIAIRZM7G4QZN4TJZQA",
		SecretKey:    "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",
		ImageId:      "ami-faf13d99",
		InstanceType: "t2.medium",
		RootSize:     "100",
		Region:       "ap-southeast-1",
		VpcId:        "vpc-76831913",
	}
)

func CreateCluster(cluster entity.Cluster, x_auth_token string) {
	var request dcosentity.Request
	request.UserName = cluster.Owner
	request.ClusterName = cluster.Name
	//	request.RequestId=
	request.ClusterNumber = cluster.Instances
	if cluster.Type == "mgmt" {
		request.IsLinkerMgmt = true
	}
	if cluster.Type == "user" {
		request.IsLinkerMgmt = false
	}

	request.ProviderInfo = providerInfo

	var servers *[]dcosentity.Server
	servers, err := SendRequest2DcosDeploy(request)
	if err != nil {
		logrus.Errorf("send request to dcos deploy error is %v", err)
		return
	}

	//loop servers and update hosts in database
	err = updateHosts(cluster.ObjectId.Hex(), servers, x_auth_token)
	if err != nil {
		logrus.Errorf("update hosts error is %v", err)
		return
	}
}

//update all hosts of a cluster by userId
//no identifier in entity Server, overwrite hosts created before in db
func updateHosts(clusterId string, servers *[]dcosentity.Server, x_auth_token string) (err error) {
	//check if all nodes successfully deployed
	for i, server := range *servers {
		if !isServerDeployed(server) {
			logrus.Errorf("Not all nodes successfully deployed. Server index [%d] is [%v]", i, server)
			return errors.New("Not all nodes successfully deployed")
		}
	}

	//query all hosts of a cluster from db
	var hosts []entity.Host
	total, hosts, _, err := GetHostService().QueryAllByClusterId(clusterId, 0, 0, x_auth_token)
	if err != nil {
		logrus.Errorf("query all hosts by cluster id error is [%v]", err)
		return err
	}
	if total != len(*servers) {
		logrus.Errorf("len of server array [%d] in dcos_deploy reponse does not equals to number of hosts [%d] in db", len(*servers), total)
		return errors.New("len of server array in dcos_deploy reponse does not equals to number of hosts in db")
	}

	//update variable hosts
	for i, server := range *servers {
		hosts[i].HostName = server.Hostname
		hosts[i].IP = server.IpAddress
		hosts[i].PrivateIp = server.PrivateIpAddress
		if server.IsMaster {
			hosts[i].IsMasterNode = true
		}
		if server.IsSlave {
			hosts[i].IsSlaveNode = true
		}
		if server.IsSwarmMaster {
			hosts[i].IsSwarmMaster = true
		}

		//TODO split string like /linker/docker/sysadmin/cluster-test4
		//		array := strings.Split(server.StoragePath, "/")
		//		if len(array) != 2 {
		//			return errors.New("Cannot parse storage path")
		//		}
		//		hosts[i].UserName = array[0]
		//		hosts[i].ClusterName = array[1]

		//update hosts to db
		_, _, err := GetHostService().UpdateById(hosts[i].ObjectId.Hex(), hosts[i], x_auth_token)
		if err != nil {
			logrus.Errorf("update host by id error is [%v]", err)
			return err
		}
	}

	return
}

//check if the node is successfully deployed
func isServerDeployed(server dcosentity.Server) bool {
	if len(strings.TrimSpace(server.Hostname)) > 0 &&
		IsIpAddressValid(strings.TrimSpace(server.IpAddress)) &&
		IsIpAddressValid(strings.TrimSpace(server.PrivateIpAddress)) &&
		len(strings.TrimSpace(server.StoragePath)) > 0 {
		return true
	} else {
		return false
	}
}
