package entity

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type Host struct {
	ObjectId      bson.ObjectId `bson:"_id" json:"_id"`
	HostName      string        `bson:"host_name" json:"host_name"`
	ClusterId     string        `bson:"cluster_id" json:"cluster_id"`
	ClusterName   string        `bson:"cluster_name" json:"cluster_name"`
	Status        string        `bson:"status" json:"status"`
	IP            string        `bson:"ip" json:"ip"`
	PrivateIp     string        `bson:"private_ip" json:"private_ip"`
	IsMasterNode  bool          `bson:"ismasternode" json:"ismasternode"`
	IsSlaveNode   bool          `bson:"isslavenode" json:"isslavenode"`
	IsSwarmMaster bool          `bson:"isswarmmaster" json:"isswarmmaster"`
	IsConsul      bool          `bson:"isconsul" json:"isconsul"`
	IsFullfilled  bool          `bson:"isfullfilled" json:"isfullfilled"`
	IsDnsServer   bool          `bson:"isdnsserver" json:"isdnsserver"`
	StoragePath   string        `bson:"storage_path" json:"storage_path"`
	UserId        string        `bson:"user_id" json:"user_id"`
	UserName      string        `bson:"username" json:"username"`
	TenantId      string        `bson:"tenant_id" json:"tenant_id"`
	TimeCreate    time.Time     `bson:"time_create" json:"time_create"`
	TimeUpdate    time.Time     `bson:"time_update" json:"time_update"`
}
