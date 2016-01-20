package command

import (
	"bytes"
	"errors"
	"github.com/Sirupsen/logrus"

	"linkernetworks.com/linker_common_lib/entity"
)

const (
	PROVIDER_TYPE_OPENSTACK = "openstack"
	PROVIDER_TYPE_AWSEC2    = "amazonec2"
)

func CreateMachine(providerType, hostname, storagePath string, swarm, swarmMaster bool, consulHost string, provider entity.ProviderInfo, labels []entity.Label) (output string, errput string, err error) {
	logrus.Infof("Prepare command to create docker machine: \n")

	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("create ")

	switch provider.Provider.ProviderType {
	case PROVIDER_TYPE_OPENSTACK:
		logrus.Infof("Openstack... \n")
		openstack := provider.OpenstackInfo
		commandTextBuffer.WriteString("--driver " + PROVIDER_TYPE_OPENSTACK + " ")
		commandTextBuffer.WriteString("--openstack-auth-url " + openstack.AuthUrl + " ")
		commandTextBuffer.WriteString("--openstack-username " + openstack.Username + " ")
		commandTextBuffer.WriteString("--openstack-password " + openstack.Password + " ")
		commandTextBuffer.WriteString("--openstack-tenant-name " + openstack.TenantName + " ")
		commandTextBuffer.WriteString("--openstack-flavor-name " + openstack.FlavorName + " ")
		commandTextBuffer.WriteString("--openstack-image-name " + openstack.ImageName + " ")
		if provider.Provider.SshUser != "" {
			commandTextBuffer.WriteString("--openstack-ssh-user " + provider.Provider.SshUser + " ")
		}
		commandTextBuffer.WriteString("--openstack-sec-groups " + openstack.SecurityGroup + " ")
		commandTextBuffer.WriteString("--openstack-floatingip-pool " + openstack.IpPoolName + " ")
		commandTextBuffer.WriteString("--openstack-nova-network " + openstack.NovaNetwork + " ")

	case PROVIDER_TYPE_AWSEC2:
		logrus.Infof("Aws... \n")
		awsec2 := provider.AwsEC2Info
		commandTextBuffer.WriteString("--driver " + PROVIDER_TYPE_AWSEC2 + " ")
		commandTextBuffer.WriteString("--amazonec2-access-key " + awsec2.AccessKey + " ")
		commandTextBuffer.WriteString("--amazonec2-secret-key " + awsec2.SecretKey + " ")
		commandTextBuffer.WriteString("--amazonec2-region " + awsec2.Region + " ")
		commandTextBuffer.WriteString("--amazonec2-vpc-id " + awsec2.VpcId + " ")
		commandTextBuffer.WriteString("--amazonec2-ssh-user " + provider.Provider.SshUser + " ")
		commandTextBuffer.WriteString("--amazonec2-instance-type " + awsec2.InstanceType + " ")
		if awsec2.ImageId != "" {
			commandTextBuffer.WriteString("--amazonec2-ami " + awsec2.ImageId + " ")
		}
		if awsec2.RootSize != "" {
			commandTextBuffer.WriteString("--amazonec2-root-size " + awsec2.RootSize + " ")
		}

		//			commandTextBuffer.WriteString("--amazonec2-instance-type " + awsec2.InstanceType + " ")

	default:
		err = errors.New("Unsupport provider type!")
		logrus.Errorf("Unsupport provider type: %v", err)
		return
	}

	if swarm {
		commandTextBuffer.WriteString("--swarm ")
	}

	if swarmMaster {
		commandTextBuffer.WriteString("--swarm-master ")
	}

	if swarm || swarmMaster {
		commandTextBuffer.WriteString("--swarm-discovery consul://$(docker-machine --storage-path " + storagePath + " ip " + consulHost + "):8500 ")
	}

	for _, label := range labels {
		commandTextBuffer.WriteString("--engine-label=\"" + label.Key + "=" + label.Value + "\" ")
	}

	commandTextBuffer.WriteString(hostname)

	logrus.Infof("Executing craete machine command: %s", commandTextBuffer.String())
	output, errput, err = ExecCommand(commandTextBuffer.String())
	return
}

func DeleteMachine(hostname, storagePath string) (output string, errput string, err error) {
	logrus.Infof("Prepare command to delete docker machine: %s \n", hostname)
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("rm -y ")
	commandTextBuffer.WriteString(hostname)
	
	logrus.Infof("Executing delete machine command: %s", commandTextBuffer.String())
	output, errput, err = ExecCommand(commandTextBuffer.String())
	return output, errput, err
}

func ExecCommandOnMachine(hostname, command, storagePath string) (output string, errput string, err error) {
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("ssh ")
	commandTextBuffer.WriteString(hostname)
	commandTextBuffer.WriteString(" -t \"")
	commandTextBuffer.WriteString(command)
	commandTextBuffer.WriteString("\"")

	logrus.Infof("Executing ssh command: %s", commandTextBuffer.String())
	output, errput, err = ExecCommand(commandTextBuffer.String())
	return
}

func ScpToMachine(hostname, localpath, remotepath, storagePath string) (output string, errput string, err error) {
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("scp ")
	commandTextBuffer.WriteString(localpath + " ")
	commandTextBuffer.WriteString(hostname + ":")
	commandTextBuffer.WriteString(remotepath)
	commandTextBuffer.WriteString("")

	logrus.Infof("Executing scp command: %s", commandTextBuffer.String())
	output, errput, err = ExecCommand(commandTextBuffer.String())
	return
}

func InspectMachineByKey(hostname, key, storagePath string) (output string, errput string, err error) {
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("inspect ")
	commandTextBuffer.WriteString(hostname + " ")
	commandTextBuffer.WriteString("-f ")
	commandTextBuffer.WriteString("{{" + key + "}}")

	logrus.Infof("Executing inspect command: %s", commandTextBuffer.String())
	output, errput, err = ExecCommand(commandTextBuffer.String())
	return
}

func GetMachinePublicIPAddress(hostname, storagePath string) (ipaddress string, err error) {
	output, errput, err := InspectMachineByKey(hostname, ".Driver.IPAddress", storagePath)
	logrus.Debugf("PublicIP output: %s", output)
	logrus.Debugf("PublicIP errput: %s", errput)

	if err != nil {
		logrus.Errorf("GetMachinePublicIPAddress failed , err is %v", err)
		return "", err
	}

	return output, nil
}

func GetMachinePrivateIPAddress(hostname, storagePath string) (ipaddress string, err error) {
	output, errput, err := InspectMachineByKey(hostname, ".Driver.PrivateIPAddress", storagePath)
	logrus.Debugf("PrivateIP output: %s", output)
	logrus.Debugf("PrivateIP errput: %s", errput)

	if err != nil {
		logrus.Errorf("GetMachinePrivateIPAddress failed , err is %v", err)
		return "", err
	}

	return output, nil
}

func BootUpConsul(hostname, storagePath string) (output, errput string, err error) {
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker $(docker-machine  ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("config ")
	commandTextBuffer.WriteString(hostname + ") ")
	commandTextBuffer.WriteString("run -d -p '8500:8500' --name='consul' -h 'consul' progrium/consul -server -bootstrap")

	logrus.Infof("Executing BootUpConsul command: %s", commandTextBuffer.String())
	output, errput, err = ExecCommand(commandTextBuffer.String())
	return
}
