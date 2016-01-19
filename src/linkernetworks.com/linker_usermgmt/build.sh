#!/bin/bash
USERMGM_HOME=$(cd `dirname $0` && pwd)
export GOPATH=${USERMGM_HOME}/../../../
OUTPUT_DIR=${GOPATH}/bin
ARTIFACT=${OUTPUT_DIR}/usermgmt

echo "Start to go third party code from github.com ..."
echo "Downloading logrus ..."
go get -v -u github.com/Sirupsen/logrus
echo "Downloading properties"
go get -v -u github.com/magiconair/properties
echo "Downloading go-restful ..."
go get -v -u github.com/emicklei/go-restful
echo "Downloading mejson ..."
go get -v -u github.com/compose/mejson
echo "Downloading mgo ..."
go get -v -u gopkg.in/mgo.v2
echo "Downloading jsonq ..."
go get -v -u github.com/jmoiron/jsonq
echo "Downloading zk ..."
go get -v -u github.com/samuel/go-zookeeper/zk
echo "Downloading uuid ..."
go get -v -u code.google.com/p/go-uuid/uuid
echo "Downloading gomail"
go get -v -u gopkg.in/gomail.v2

echo "Start to build linker usermgmt ..."
rm -f ${ARTIFACT}
go build -a -o ${ARTIFACT} ${USERMGM_HOME}/main.go

if [[ $? -ne 0 ]]; then
	#build error
	echo "build ERROR"
	exit 1
fi

echo "Copying properties file to bin/ ..."
cp ${USERMGM_HOME}/usermgmt.properties ${OUTPUT_DIR}/usermgmt.properties
cp ${USERMGM_HOME}/userPolicy.json ${OUTPUT_DIR}/userPolicy.json

# fetch or update Swagger UI
if [ ! -d ${OUTPUT_DIR}/swagger-ui ]; then
	git clone https://github.com/wordnik/swagger-ui.git ${OUTPUT_DIR}/swagger-ui
fi

# reset apidoc.json
sed -i 's/"http:\/\/petstore.swagger.io\/v2\/swagger.json"/"\/apidocs.json"/g' ${OUTPUT_DIR}/swagger-ui/dist/index.html

# ARCHIVE="usermgmt-"`date +"%Y%m%d"`".zip"
ARCHIVE="usermgmt.zip"
cd ${OUTPUT_DIR}
	rm -f ${ARCHIVE}
	zip $ARCHIVE usermgmt
	zip $ARCHIVE usermgmt.properties
	zip $ARCHIVE userPolicy.json
	zip -r $ARCHIVE ./swagger-ui/dist
cd ..

#scp bin/controller root@ansible:/root/Linker_Ansible/linker_ansible_repo/Linker_Mesos_Cluster/roles/controller/files/
# expect <<EOF
# spawn scp bin/controller root@60.248.180.41:/tmp/
# expect "root@60.248.180.41's password: "
# send "lkr@iclk200\r"
# expect eof
# EOF