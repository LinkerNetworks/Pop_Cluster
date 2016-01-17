package entity

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type Cluster struct {
	ObjectId   bson.ObjectId `bson:"_id" json:"_id"`
	Name       string        `bson:"name" json:"name"`
	Owner      string        `bson:"owner" json:"owner"`
	Endpoint   string        `bson:"endpoint" json:"endpoint"`
	Instances  int           `bson:"instances" json:"instances"`
	Details    string        `bson:"details" json:"details"`
	Status     string        `bson:"status" json:"status"`
	Type       string        `bson:"type" json:"type"` //Type: user(default)|mgmt
	UserId     string        `bson:"user_id" json:"user_id"`
	TenantId   string        `bson:"tenant_id" json:"tenant_id"`
	TimeCreate time.Time     `bson:"time_create" json:"time_create"`
	TimeUpdate time.Time     `bson:"time_update" json:"time_update"`
}
