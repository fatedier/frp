package model

import (
	"time"
)

type User struct {
	Id         int       `xorm:"not null pk autoincr INT(11)"`
	Username   string    `xorm:"not null default '' VARCHAR(128)"`
	Password   string    `xorm:"not null default '' VARCHAR(128)"`
	Serssion   string    `xorm:"VARCHAR(128)"`
	CreateTime time.Time `xorm:"DATETIME"`
	Status     int       `xorm:"INT(11)"`
}
