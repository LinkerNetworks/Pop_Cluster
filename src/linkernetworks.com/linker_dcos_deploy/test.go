package main

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"io/ioutil"
	"linkernetworks.com/linker_common_lib/entity"
)

func main() {
	payload, err := ioutil.ReadFile("/root/marathon-linkercomponents.json")

	if err != nil {
		logrus.Errorf("read linkercomponents.json failed, error is %v", err)
		return
	}

	// logrus.Info("payload is : " + string(payload))

	var serviceGroup *entity.ServiceGroup
	err = json.Unmarshal(payload, &serviceGroup)
	if err != nil {
		logrus.Errorf("Unmarshal linkercomponents.json failed, error is %v", err)
		return
	}

	mongoserverlist := "127.0.0.1,127.0.0.1,127.0.0.1"

	for _, group := range serviceGroup.Groups {
		// There is no case for group embeded group.
		for _, app := range group.Apps {
			if app.Env != nil && app.Env["MONGODB_NODES"] != "" {
				app.Env["MONGODB_NODES"] = mongoserverlist
			}
		}
	}

	payload, err = json.Marshal(*serviceGroup)
	if err != nil {
		logrus.Errorf("Marshal linkercomponents err is %v", err)
		return
	}

	logrus.Info("payload is : " + string(payload))

}
