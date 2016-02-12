package services

import (
	"errors"

	"github.com/Sirupsen/logrus"
	// "gopkg.in/mgo.v2/bson"

	dcosentity "linkernetworks.com/linker_common_lib/entity"
	"linkernetworks.com/linker_common_lib/persistence/dao"
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
		AccessKey:    "AKIAJZHML45JIDQYSBFA",
		SecretKey:    "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",
		ImageId:      "ami-faf13d99",
		InstanceType: "t2.large",
		RootSize:     "100",
		Region:       "ap-southeast-1",
		VpcId:        "vpc-76831913",
	}
)

//call dcos deployment module to create a cluster
func CreateCluster(cluster entity.Cluster, x_auth_token string) (err error) {
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

	request.ProviderInfo = getProvider(cluster.UserId, x_auth_token)

	clusterId := cluster.ObjectId.Hex()

	var servers *[]dcosentity.Server
	//send request to deployment module
	servers, err = SendCreateClusterRequest(request)
	if err != nil {
		logrus.Errorf("send request to dcos deploy error is %v", err)
		//update status of all hosts in the cluster FAILED
		updateHostStatusInCluster(clusterId, HOST_STATUS_FAILED, x_auth_token)
		//update cluster status FAILED
		logrus.Infoln("set cluster status to FAILED")
		GetClusterService().UpdateStatusById(clusterId, CLUSTER_STATUS_FAILED, x_auth_token)
		return
	}

	//loop servers and update hosts in database
	logrus.Infoln("update cluster host status wo RUNNING")
	err = updateHosts(clusterId, servers, x_auth_token)
	if err != nil {
		logrus.Errorf("update hosts error is %v", err)
		return
	}

	//update status RUNNING
	logrus.Infoln("set cluster status to RUNNING and endpoint")
	currentCluster, _, err := GetClusterService().QueryById(clusterId, x_auth_token)
	if err != nil {
		logrus.Errorf("get cluster by id [%v] error %v", clusterId, err)
		return err
	}

	currentCluster.Status = CLUSTER_STATUS_RUNNING
	currentCluster.Endpoint = buildEndPoint(servers, currentCluster.Name)

	GetClusterService().UpdateCluster(currentCluster, x_auth_token)

	// GetClusterService().UpdateStatusById(clusterId, CLUSTER_STATUS_RUNNING, x_auth_token)
	return
}

//TODO: temperary method ,will be removed future!!!!!
func getProvider(userId string, token string) (providerinfo dcosentity.ProviderInfo) {
	logrus.Infof("get ec2 provider by userId [%s]", userId)

	_, providers, _, err := GetProviderService().QueryProvider(EC2_TYPE, userId, 0, 0, "", token)
	if err != nil {
		logrus.Errorf("get ec2 provider error [%v]", err)
		return providerInfo
	}

	if providers != nil && len(providers) > 0 {
		value := providers[0]
		providerinfo = dcosentity.ProviderInfo{}

		provider = dcosentity.Provider{
			ProviderType: value.Type,
			SshUser:      value.SshUser}

		providerinfo.Provider = provider
		providerinfo.AwsEC2Info = value.AwsEC2Info

		logrus.Debugf("ec2 providerInfo is %v", providerinfo)

		return providerinfo
	} else {
		logrus.Infof("no ec2 provider info in db, will use default value")
		return providerInfo
	}
}

//call dcos deployment module to delete a cluster
func DeleteCluster(cluster entity.Cluster, x_auth_token string) (err error) {
	clusterId := cluster.ObjectId.Hex()
	//query all unterminated hosts
	// _, hosts, _, err := GetHostService().QueryAllByClusterId(clusterId, 0, 0, x_auth_token)
	_, hosts, _, err := GetHostService().QueryHosts(clusterId, 0, 0, "unterminated", x_auth_token)
	if err != nil {
		logrus.Errorf("query hosts in cluster %s error is %v", clusterId, err)
		return
	}

	// var servers []dcosentity.Server
	servers := []dcosentity.Server{}
	for _, host := range hosts {
		//convert entity.Host -> dcosentity.Server
		server := dcosentity.Server{}
		//need hostname only when delete a node
		server.Hostname = host.HostName

		servers = append(servers, server)
	}

	if len(servers) <= 0 {
		logrus.Infof("no running host in cluser %s, the cluster will be terminated directly!", cluster.Name)
		GetClusterService().UpdateStatusById(clusterId, CLUSTER_STATUS_TERMINATED, x_auth_token)
		return
	}

	request := new(dcosentity.DeleteRequest)
	request.UserName = cluster.Owner
	request.ClusterName = cluster.Name
	request.Servers = servers

	_, err = SendDeleteClusterRequest(request)
	if err != nil {
		logrus.Errorf("send delete cluster requst to dcos deployment error [%v] \n", err)
		//update status of all hosts in the cluster FAILED
		updateHostStatusInCluster(clusterId, HOST_STATUS_FAILED, x_auth_token)
		//update status FAILED
		logrus.Infoln("set cluster status to FAILED")
		GetClusterService().UpdateStatusById(clusterId, CLUSTER_STATUS_FAILED, x_auth_token)
		return
	}
	//update status of all hosts in the cluster TERMINATED
	logrus.Infoln("set cluster hosts to TERMINATED")
	updateHostStatusInCluster(clusterId, HOST_STATUS_TERMINATED, x_auth_token)
	//update status TERMINATED
	logrus.Infoln("set cluster status to TERMINATED")
	GetClusterService().UpdateStatusById(clusterId, CLUSTER_STATUS_TERMINATED, x_auth_token)
	return
}

//call dcos deployment module to add nodes
func AddNodes(cluster entity.Cluster, createNumber int, hosts []entity.Host, x_auth_token string) (err error) {
	request := new(dcosentity.AddNodeRequest)
	request.UserName = cluster.Owner
	request.ClusterName = cluster.Name
	request.CreateNumber = createNumber
	request.ExistedNumber = cluster.Instances
	request.ProviderInfo = getProvider(cluster.UserId, x_auth_token)

	//create hosts
	_, currentHosts, _, err := GetHostService().QueryHosts(cluster.ObjectId.Hex(), 0, 0, "unterminated", x_auth_token)
	if err != nil {
		logrus.Errorf("get current hosts by clusterId error %v", err)
		removeAddedNodes(hosts, x_auth_token)
		return err
	}

	request.ConsulServer, request.DnsServers, request.SwarmMaster = getNodeInfo(currentHosts)

	logrus.Debugf("add node request is %v", request)

	//call
	//dcos deployment module returns newly-created nodes info
	var servers *[]dcosentity.Server
	servers, err = SendAddNodesRequest(request)
	if err != nil {
		logrus.Errorf("send request to dcos deploy error is %v", err)
		removeAddedNodes(hosts, x_auth_token)
		return
	}

	ok_count := len(*servers)
	logrus.Infof("try to add %d nodes, %d success", createNumber, ok_count)

	//update newly-created hosts
	for i, server := range *servers {
		host := hosts[i]
		id := host.ObjectId.Hex()
		host.HostName = server.Hostname
		host.Status = HOST_STATUS_RUNNING
		host.IP = server.IpAddress
		host.PrivateIp = server.PrivateIpAddress
		host.IsMasterNode = server.IsMaster
		host.IsSlaveNode = server.IsSlave
		host.IsSwarmMaster = server.IsSwarmMaster
		host.TimeUpdate = dao.GetCurrentTime()
		_, _, err := GetHostService().UpdateById(id, host, x_auth_token)
		if err != nil {
			logrus.Errorf("update host error is [%v] \n", err)
		}
	}

	//update status of failed nodes
	for i := ok_count; i < createNumber; i++ {
		_, _, err := GetHostService().UpdateStatusById(hosts[i].ObjectId.Hex(), HOST_STATUS_FAILED, x_auth_token)
		if err != nil {
			logrus.Errorf("update host error is [%v] \n", err)
		}
	}

	//update cluster nodes number and status
	logrus.Infoln("update cluster number and status")
	cluster.Instances = cluster.Instances + len(hosts)
	cluster.Status = CLUSTER_STATUS_RUNNING
	GetClusterService().UpdateCluster(cluster, x_auth_token)
	// //update cluster status
	// logrus.Infoln("set cluster status to RUNNING")
	// GetClusterService().UpdateStatusById(cluster.ObjectId.Hex(), CLUSTER_STATUS_RUNNING, x_auth_token)
	return
}

func removeAddedNodes(hosts []entity.Host, token string) {
	logrus.Infof("remove hosts for add nodes")
	if hosts == nil || len(hosts) <= 0 {
		logrus.Infoln("no hosts need to be removed")
		return
	}

	for _, host := range hosts {
		_, _, err := GetHostService().UpdateStatusById(host.ObjectId.Hex(), HOST_ERROR_TERMINATED, token)
		logrus.Warnf("remove added host error %v ", err)
		continue
	}

}

func buildEndPoint(servers *[]dcosentity.Server, clusterName string) string {
	logrus.Infof("build endpoint for %v", clusterName)
	address := ""
	for _, server := range *servers {
		if server.IsDnsServer {
			address = server.IpAddress
			break
		}
	}

	if len(address) <= 0 {
		logrus.Warnf("can not find dns server for current cluster %v", clusterName)
		return address
	}

	url := "http://" + address + ":10012"
	logrus.Infof("cluster [%v] endpoint is : %v", clusterName, url)
	return url
}

func getNodeInfo(hosts []entity.Host) (consul string, dns []dcosentity.Server, swarm string) {
	if hosts == nil || len(hosts) <= 0 {
		logrus.Warnf("no node for current cluster!")
		return
	}

	dns = []dcosentity.Server{}
	for i := 0; i < len(hosts); i++ {
		host := hosts[i]
		if host.IsConsul {
			consul = host.HostName
		}
		if host.IsSwarmMaster {
			swarm = host.HostName
		}
		if host.IsDnsServer {
			server := dcosentity.Server{}

			server.Hostname = host.HostName
			server.IpAddress = host.IP
			server.PrivateIpAddress = host.PrivateIp
			server.IsMaster = host.IsMasterNode
			server.IsSlave = host.IsSlaveNode
			server.IsSwarmMaster = host.IsSwarmMaster
			server.IsConsul = host.IsConsul
			server.IsFullfilled = host.IsFullfilled
			server.IsDnsServer = host.IsDnsServer
			server.StoragePath = host.StoragePath

			dns = append(dns, server)
		}

	}

	return
}

//call dcos deployment module to delete nodes
func DeleteNodes(cluster entity.Cluster, hosts []entity.Host, originStatus map[string]string, x_auth_token string) (err error) {
	// var servers []dcosentity.Server
	servers := []dcosentity.Server{}
	for _, host := range hosts {
		server := dcosentity.Server{}
		server.Hostname = host.HostName
		server.IpAddress = host.IP
		server.PrivateIpAddress = host.PrivateIp
		server.IsMaster = host.IsMasterNode
		server.IsSlave = host.IsSlaveNode
		server.IsSwarmMaster = host.IsSwarmMaster
		server.StoragePath = host.StoragePath

		servers = append(servers, server)
	}

	request := new(dcosentity.DeleteRequest)
	request.UserName = cluster.Owner
	request.ClusterName = cluster.Name
	request.Servers = servers

	var deleted_servers *[]dcosentity.Server
	deleted_servers, err = SendDeleteNodesRequest(request)
	if err != nil {
		logrus.Errorf("send delete nodes requst to dcos deployment error [%v] \n", err)
		logrus.Infoln("rollback hosts status")
		// GetClusterService().UpdateStatusById(cluster.ObjectId.Hex(), CLUSTER_STATUS_RUNNING, x_auth_token)
		for _, host := range hosts {
			preStatus := originStatus[host.ObjectId.Hex()]
			_, _, err := GetHostService().UpdateStatusById(host.ObjectId.Hex(), preStatus, x_auth_token)
			if err != nil {
				logrus.Warnf("rollback host[%v] status to [%v] error [%v]", host.HostName, preStatus, err)
			}
		}

		return
	}

	//send deployment success, may success partially
	var hostnames []string
	for _, server := range *deleted_servers {
		hostnames = append(hostnames, server.Hostname)
	}

	for _, host := range hosts {
		var status string
		if StringInSlice(host.HostName, hostnames) {
			status = HOST_STATUS_TERMINATED
		} else {
			status = HOST_STATUS_FAILED
		}
		//update status
		_, _, err := GetHostService().UpdateStatusById(host.ObjectId.Hex(), status, x_auth_token)
		if err != nil {
			logrus.Errorf("update host status error is [%v]", err)
		}
	}

	//update cluster nodes number and status
	logrus.Infoln("update cluster number and status")
	cluster.Instances = cluster.Instances - len(*deleted_servers)
	cluster.Status = CLUSTER_STATUS_RUNNING
	GetClusterService().UpdateCluster(cluster, x_auth_token)
	// //update cluster status
	// logrus.Infoln("set cluster status to RUNNING")
	// GetClusterService().UpdateStatusById(cluster.ObjectId.Hex(), CLUSTER_STATUS_RUNNING, x_auth_token)
	return
}

//update all hosts of a cluster by userId
//no identifier in entity Server, overwrite hosts created before in db
func updateHosts(clusterId string, servers *[]dcosentity.Server, x_auth_token string) (err error) {
	//query all hosts of a cluster from db
	var hosts []entity.Host
	total, hosts, _, err := GetHostService().QueryHosts(clusterId, 0, 0, "unterminated", x_auth_token)
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
		logrus.Debugf("update hosts %v", server)
		hosts[i].HostName = server.Hostname
		hosts[i].IP = server.IpAddress
		hosts[i].PrivateIp = server.PrivateIpAddress
		hosts[i].IsMasterNode = server.IsMaster
		hosts[i].IsSlaveNode = server.IsSlave
		hosts[i].IsSwarmMaster = server.IsSwarmMaster
		hosts[i].IsConsul = server.IsConsul
		hosts[i].IsFullfilled = server.IsFullfilled
		hosts[i].IsDnsServer = server.IsDnsServer
		hosts[i].StoragePath = server.StoragePath
		hosts[i].Status = HOST_STATUS_RUNNING

		//update hosts to db
		_, _, err := GetHostService().UpdateById(hosts[i].ObjectId.Hex(), hosts[i], x_auth_token)
		if err != nil {
			logrus.Errorf("update host by id error is [%v]", err)
			return err
		}
	}

	return
}

//update status of all hosts in a cluster
func updateHostStatusInCluster(clusterId string, status string, x_auth_token string) (err error) {
	var hosts []entity.Host
	// _, hosts, _, err = GetHostService().QueryAllByClusterId(clusterId, 0, 0, x_auth_token)
	_, hosts, _, err = GetHostService().QueryHosts(clusterId, 0, 0, "unterminated", x_auth_token)
	if err != nil {
		logrus.Errorf("query all hosts by cluster id error is [%v]", err)
		return err
	}

	for _, host := range hosts {
		_, _, err := GetHostService().UpdateStatusById(host.ObjectId.Hex(), status, x_auth_token)
		if err != nil {
			logrus.Errorf("update host by id error is [%v]", err)
			return err
		}
	}
	return
}
