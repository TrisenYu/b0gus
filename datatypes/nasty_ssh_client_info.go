// SPDX-LICENSE-IDENTIFIER: GPL2.0
//
// (C) All rights reserved. Author: <kisfg@hotmail.com> in 2025
// Created at 2025/11/15 星期六 17:19:53
// Last modified at 2025/11/15 星期六 22:22:34
package datatypes

// 1. multiple struct tags in one field should split by space key
// 2. read manual and code, or debug multiple times

type AttackerAddrDef struct {
	ID             uint64                `gorm:"primaryKey;autoIncrement;not null"` // ID as the primary key
	IP             string                `gorm:"unique;index:ip_name_idx;"`         // attacker IP address TOOD: does Intranet have side effect?
	PortNumRelated []AttackerPortInfoDef `gorm:"foreignKey:AttackerID;comment:Related to different ports but running in the same host"`
}

type AttackerPortInfoDef struct {
	// treat IP as a seperate field to decouple port and other infomation
	APIid         uint64 `gorm:"primaryKey;autoIncrement;not null"`
	StartTime     string `gorm:"autoUpdateTime:nano;comment: start time that attacker lanuched an attack"`
	ClientVersion string // SSH client version string

	// foreign key definition zone
	AttackerID         uint64
	AttackerRelated    AttackerAddrDef  `gorm:"foreignKey:AttackerID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	AttackerCommandDef []AttackerCmdDef `gorm:"foreignKey:LoginedID;"`

	Port uint16 // remote port
}

type AttackerPassInfoDef struct {
	Password        string `gorm:"unique;not null;comment: password attempted to login into system"`
	FirstRecordTime int64  `gorm:"autoCreateTime:nano"`
	UpdatedTime     int64  `gorm:"autoUpdateTime:nano"`
}

type AttackerPubInfoDef struct {
	PubKeyFingerprint string // public key fingerprint used by attacker
}

type AttackerCmdDef struct {
	CmdID       uint64 `gorm:"primaryKey;autoIncrement"`
	CMD         string `gorm:"comment: Commands text that was passed by attacker to harm current machine"`
	RequestTime int64  `gorm:"autoUpdateTime:nano;comment: time that requests for executing command"`

	// foreign key definition zone
	LoginedID       uint64
	AttackerRelated AttackerPortInfoDef `gorm:"foreignKey:LoginedID;references:APIid;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

// SQLite might have similar datatype for easier interaction
