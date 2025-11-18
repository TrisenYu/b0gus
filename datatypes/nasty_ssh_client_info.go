// SPDX-LICENSE-IDENTIFIER: GPL2.0
//
// (C) All rights reserved. Author: <kisfg@hotmail.com> in 2025
// Created at 2025/11/15 星期六 17:19:53
// Last modified at 2025/11/15 星期六 22:22:34
package datatypes

// multiple struct tags in one field should split by space key

type AttackerInfoDef struct {
	ID                uint64 `gorm:"primaryKey;default:autoIncrement;not null"` // identifier as the primary key
	IP                string // attacker IP address
	Port              uint16 // remote port
	AttemptedPassword string // password attempted to login into system
	PubKeyFingerprint string // public key fingerprint used by attacker
	AccessTime        string // time that attacker lanuched an attack
	ClientVersion     string // SSH client version string
}

type AttackerCmdDef struct {
	CmdID       uint64 `gorm:"primaryKey;default:autoIncrement"`
	CMD         string // commands wanted to execute in our machine by attacker
	AttackerID  uint64 // foreign key to AttackerInfoDef.ID
	RequestTime int64  `gorm:"autoUpdateTime:nano"` // time that requests for executing command
}

// SQLite might have similar datatype for easier interaction
