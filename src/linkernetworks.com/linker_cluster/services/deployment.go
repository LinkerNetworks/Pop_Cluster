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
		ImageId:      "",
		InstanceType: "t2.micro",
		RootSize:     "",
		Region:       "ap-southeast-1",
		VpcId:        "vpc-4630a923",
	}
)

func CreateCluster(cluster entity.Cluster) {
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
	err = updateHosts(servers)
	if err != nil {
		logrus.Errorf("update hosts error is %v", err)
		return
	}
}

func updateHosts(servers *[]dcosentity.Server) (err error) {
	for _, server := range *servers {
		var host entity.Host
		host.HostName = server.Hostname
		host.IP = server.IpAddress
		host.PrivateIp = server.PrivateIpAddress
		if server.IsMaster {
			host.IsMasterNode = true
		}
		if server.IsSlave {
			host.IsSlaveNode = true
		}
		if server.IsSwarmMaster {
			host.IsSwarmMaster = true
		}
		array := strings.Split(server.StoragePath, "/")
		if len(array) != 2 {
			return errors.New("Cannot parse storage path")
		}
		host.UserName = array[0]
		host.ClusterName = array[1]

		//update host by clustername
		//		err := updateHost(host.UserName, host.ClusterName)
	}
	return
}
