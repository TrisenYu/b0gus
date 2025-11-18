// Last modified at 2025/11/15 星期六 22:22:34
// SPDX-LICENSE-IDENTIFIER: GPL2.0
//
// (C) All rights reserved. Author: <kisfg@hotmail.com> in 2025
// Created at 2025/11/15 星期六 17:19:53
// Last modified at 2025/11/15 星期六 20:04:54
package datatypes

// PG stands for PostGreSQL
type PGattackerInfoDef struct {
	ID                uint64 // identifier as the primary key
	IP                string // attacker IP address
	Port              uint16 // remote port
	AttemptedPassword string // password attempted to login into system
	PubKeyFingerprint string // public key fingerprint used by attacker
	AccessTime        string // time that attacker lanuched an attack
	ClientVersion     string // SSH client version string
}

type PGattackerCmdDef struct {
	CMD        string // commands wanted to execute in our machine by attacker
	AttackerID uint64 // foreign key to AttackerInfoDef.ID
}

// SQLite might have similar datatype for easier interaction
