package command

import (
	"bytes"
	"github.com/Sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

func clusterCompose(userName, clusterName, swarmName, storagePath string, scale int) (output, errput string, err error) {
	tmpOutPut, tmpErrPut, tmpErr := changeSwarm(userName, clusterName, swarmName, storagePath, scale)
	output = output + tmpOutPut + "\n"
	errput = errput + tmpErrPut + "\n"
	if tmpErr != nil {
		err = tmpErr
		return
	}
	return
}
func changeSwarm(userName, clusterName, swarmName, storagePath string, scale int) (output, errput string, err error) {
	var commandTextBuffer bytes.Buffer
	str := strconv.Itoa(scale)
	commandTextBuffer.WriteString("eval ")
	commandTextBuffer.WriteString("`docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("env --swarm ")
	commandTextBuffer.WriteString(swarmName + "` &&")
	commandTextBuffer.WriteString("docker-compose -f ")
	commandTextBuffer.WriteString("./" + userName + "/" + clusterName + "/docker-compose.yml ")
	commandTextBuffer.WriteString("scale ")
	commandTextBuffer.WriteString("zookeeper")
	commandTextBuffer.WriteString("=3 ")
	commandTextBuffer.WriteString("mesosmaster")
	commandTextBuffer.WriteString("=3 ")
	commandTextBuffer.WriteString("marathon")
	commandTextBuffer.WriteString("=3 ")
	commandTextBuffer.WriteString("mesosslave")
	commandTextBuffer.WriteString("=" + str)
	logrus.Infof(commandTextBuffer.String())
	logrus.Infof("Change Swarm to: %s", userName)
	output, errput, err = ExecCommand(commandTextBuffer.String())
	return
}

// masterList: ["10.10.10.1", "10.10.10.2", "10.10.10.3"]
// nodeList node=publicIp: ["hostname1=10.10.10.1", "hostname2=10.10.10.2", "hostname2=10.10.10.3"]
// scale mesosslave
func InstallCluster(userName, clusterName, swarmName, storagePath string, masterList []string, nodeList []string, scale int) error {
	err := fillEnvFile(userName, clusterName, masterList, nodeList)
	if err != nil {
		return err
	}
	_, tmpErr, err := clusterCompose(userName, clusterName, swarmName, storagePath, scale)
	if err != nil {
		logrus.Infof(tmpErr)
		return err
	}
	return nil
}

// addNode node=publicIp: "hostname1=10.10.10.1"
// scale mesosslave
func AddSlaveToCluster(userName, clusterName, swarmName, storagePath string, addNodes []string, scale int) error {
	envFile, err := createOrGetEnvFile(userName, clusterName)

	if err != nil {
		logrus.Error(err)
		return err
	}
	defer envFile.Close()

	for _, tmpStr := range addNodes {
		envFile.WriteString(tmpStr + "\n")
	}

	_, tmpErr, err := clusterCompose(userName, clusterName, swarmName, storagePath, scale)
	if err != nil {
		logrus.Infof(tmpErr)
		return err
	}

	return nil
}

// deleteNode node=publicIp: "hostname2"
func RemoveSlaveFromCluster(userName, clusterName, deleteNode string) error {
	envFile, err := createOrGetEnvFile(userName, clusterName)

	if err != nil {
		logrus.Error(err)
		return err
	}
	defer envFile.Close()

	content, err := ioutil.ReadFile(envFile.Name())
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")

	envFile.Truncate(0)
	for _, line := range lines {
		if strings.HasPrefix(line, deleteNode+"=") == false {
			envFile.WriteString(line)
		}
	}

	return nil
}

func fillEnvFile(userName string, clusterName string, masterList []string, nodeList []string) error {
	envFile, err := createOrGetEnvFile(userName, clusterName)
	if err != nil {
		logrus.Error(err)
		return err
	}
	defer envFile.Close()
	ZOOKEEPERLIST := masterList[0] + ":2888:3888," + masterList[1] + ":2888:3888," + masterList[2] + ":2888:3888"
	MESOS_ZK := "zk://" + masterList[0] + ":2181," + masterList[1] + ":2181," + masterList[2] + ":2181" + "/mesos"
	MARATHON_MASTER := MESOS_ZK
	MESOS_MASTER := MESOS_ZK
	MARATHON_ZK := "zk://" + masterList[0] + ":2181," + masterList[1] + ":2181," + masterList[2] + ":2181" + "/marathon"
	envFile.WriteString("ZOOKEEPERLIST=" + ZOOKEEPERLIST + "\n")
	envFile.WriteString("MESOS_ZK=" + MESOS_ZK + "\n")
	envFile.WriteString("MESOS_MASTER=" + MESOS_MASTER + "\n")
	envFile.WriteString("MARATHON_MASTER=" + MARATHON_MASTER + "\n")
	envFile.WriteString("MARATHON_ZK=" + MARATHON_ZK + "\n")
	envFile.WriteString("MESOS_HOSTNAME_LOOKUP=false\n")
	for _, tmpStr := range nodeList {
		envFile.WriteString(tmpStr + "\n")
	}
	return nil
}

func createOrGetEnvFile(userName string, clusterName string) (envFile *os.File, err error) {
	absolutePath := "./" + userName + "/" + clusterName
	absoluteFilePath := absolutePath + "/.env"
	_, errInfile := os.Stat(absoluteFilePath)
	isExisted := errInfile == nil || os.IsExist(errInfile)
	if isExisted {
		envFile, err = os.OpenFile(absoluteFilePath, os.O_RDWR, 0)
	} else {
		os.MkdirAll(absolutePath, os.ModePerm)
		_, err = copyFile(absolutePath+"/docker-compose.yml", "/linker/config/docker-compose.yml")
		if err != nil {
			return
		}
		envFile, err = os.Create(absoluteFilePath)
	}
	return
}

//Copyfile
func copyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}
