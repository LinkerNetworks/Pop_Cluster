package services

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/pborman/uuid"
	"linkernetworks.com/linker_common_lib/entity"
	"os"
	"sync"

	command "linkernetworks.com/linker_dcos_deploy/command"
)

const (
	DOCKERMACHINE_ERROR_STORAGEPATH_CREATE string = "E61001"

	DOCKERMACHINE_STORAGEPATH_PREFIX string = "/linker/docker/"
)

var (
	dockerMachineService *DockerMachineService = nil
	onceDockerMachine    sync.Once
)

type DockerMachineService struct {
	serviceName string
}

func GetDockerMachineService() *DockerMachineService {
	onceDockerMachine.Do(func() {
		logrus.Debugf("Once called from DockerMachineService ......................................")
		dockerMachineService = &DockerMachineService{"DockerMachineService"}
	})
	return dockerMachineService

}

func (p *DockerMachineService) Create(username, clusername string, swarm, swarmMaster bool, consulHost string, provider entity.ProviderInfo, labels []entity.Label) (
	server entity.Server, errorCode string, err error) {

	logrus.Infof("start to create Docker Machine...")

	providerType := ""
	hostname := username + "." + clusername + "." + uuid.New()
	storagePath := p.ComposeStoragePath(username, clusername)

	err = os.MkdirAll(storagePath, os.ModePerm)
	if err != nil {
		errorCode = DOCKERMACHINE_ERROR_STORAGEPATH_CREATE
		logrus.Errorf("DOCKERMACHINE_ERROR_STORAGEPATH_CREATE: %v", err)
		return
	}

	_, _, err = command.CreateMachine(providerType, hostname, storagePath, swarm, swarmMaster, consulHost, provider, labels)

	if err != nil {
		logrus.Errorf("Create docker machine failed: %v", err)
		return
	}

	ipAddress, err := command.GetMachinePublicIPAddress(hostname, storagePath)
	if err != nil {
		logrus.Errorf("GetMachinePublicIPAddress failed , err is %v", err)
		return
	}

	privateIPAddress, err := command.GetMachinePrivateIPAddress(hostname, storagePath)
	if err != nil {
		logrus.Errorf("GetMachinePrivateIPAddress failed , err is %v", err)
		return
	}

	server = entity.Server{hostname, ipAddress, privateIPAddress, false, false, false, storagePath, false, false, false}
	//	server.Hostname = hostname
	//	server.IpAddress = ipAddress
	//	server.PrivateIpAddress = privateIPAddress

	// here, change host first to let docker-machine connect machine first.
	errorCode, err = p.ChangeHost(hostname, privateIPAddress, storagePath)
	if err != nil {
		logrus.Errorf("replace ssh key failed , err is %v", err)
		return
	}

	err = p.ReplaceKey(hostname, provider.Provider.SshUser, storagePath, server.IpAddress)
	// errorCode, err = p.ChangeHost(hostname, privateIPAddress, storagePath)

	if err == nil {
		server.IsFullfilled = true
	}

	return
}

func (p *DockerMachineService) DeleteMachine(username, clusername, hostname string) (err error) {
	storagePath := p.ComposeStoragePath(username, clusername)
	_, _, err = command.DeleteMachine(hostname, storagePath)
	if err != nil {
		return err
	}
	return
}

func (p *DockerMachineService) ReplaceKey(hostname, sshUser, storagePath, publicip string) (err error) {
	commandStr := "eval `ssh-agent` && ssh-add " + storagePath + "/machines" + "/" + hostname + "/id_rsa && " + "/linker/copy-ssh-id.sh " + sshUser + " " + publicip + " /linker/key/id_rsa.pub"

	logrus.Infof("Executing add key and copy id command: %s", commandStr)
	_, _, err = command.ExecCommand(commandStr)
	if err != nil {
		logrus.Errorf("Call ssh-add failed , err is %v", err)
		return
	}

	return
}

func (p *DockerMachineService) ChangeHost(hostname, ipAddress, storagePath string) (errCode string, err error) {
	//prepare command
	commandStr := fmt.Sprintf(`
	if grep -xq .*%s /etc/hosts; then
		if grep -xq 127.0.1.1.* /etc/hosts; then 
			sudo sed -i 's/^127.0.1.1.*/%s %s/g' /etc/hosts; 
		else 
			echo '%s %s' | sudo tee -a /etc/hosts; 
		fi
	else
		echo '%s %s' | sudo tee -a /etc/hosts; 
	fi`,
		hostname, ipAddress, hostname,
		ipAddress, hostname, ipAddress, hostname)

	_, _, err = command.ExecCommandOnMachine(hostname, commandStr, storagePath)
	if err != nil {
		errCode = DEPLOY_ERROR_CHANGE_HOST
		logrus.Errorf("change hosts failed for server [%v], error is %v", ipAddress, err)
		return
	}
	return
}

func (p *DockerMachineService) ComposeStoragePath(username, clusername string) string {
	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + username + "/" + clusername + ""
	return storagePath
}
