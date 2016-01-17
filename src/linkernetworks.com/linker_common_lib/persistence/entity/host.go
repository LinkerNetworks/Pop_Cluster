package entity

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type Host struct {
	ObjectId     bson.ObjectId `bson:"_id" json:"_id"`
	HostName     string        `bson:"host_name" json:"host_name"`
	ClusterId    string        `bson:"cluster_id" json:"cluster_id"`
	ClusterName  string        `bson:"cluster_name" json:"cluster_name"`
	Status       string        `bson:"status" json:"status"`
	IP           string        `bson:"ip" json:"ip"`
	IsMasterNode bool          `bson:"ismasternode" json:"ismasternode"`
	UserId       string        `bson:"user_id" json:"user_id"`
	TenantId     string        `bson:"tenant_id" json:"tenant_id"`
	TimeCreate   time.Time     `bson:"time_create" json:"time_create"`
	TimeUpdate   time.Time     `bson:"time_update" json:"time_update"`
}
