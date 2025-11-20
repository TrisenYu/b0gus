// SPDX-LICENSE-IDENTIFIER: GPL2.0
//
// (C) All rights reserved. Author: <kisfg@hotmail.com> in 2025
// Created at 2025/11/15 星期六 17:19:53
// Last modified at 2025/11/15 星期六 22:22:34
package datatypes

// recommends for struct tags:
// 	1. multiple struct tags in one field should split by space key
// 	2. read manual and code, or debug multiple times

/*
	too abstract and also too precious
	can not modify their definition in the future due to the labrious work and efforts on audit

	{ssh -V}

	# {remote IP}:{remote Port}
	ssh {nobody's name} @ HostIP
	--> {publicKey fingerprint} ---> Host
	[+] {nobody's password}

	$ {command}
	$ ...

	IP as a basic table
	Port -> IP, 	 	 create a foreignKey relation
	name from Port, 	 create a foreignKey relation
	publicKey from Port, create a foreignKey relation
	password from Port,  create a foreignKey relation
	commands from Port,  create a foreignKey relation

	we also want each field is unique
*/

type AddrInfoDef struct {
	ID uint64 `gorm:"primaryKey;autoIncrement;not null;comment: ID as the primary key"`
	IP string `gorm:"unique;uniqueIndex:ip_name_idx;comment: attacker IP address"` // TODO: does Intranet have side effect?
	/* defined for outer foreign key */
	PortNumRelated []PortInfoDef `gorm:"foreignKey:AddrID;comment:Related to different ports but running in the same host"`
}

type PortInfoDef struct {
	// treat IP as a seperate field to decouple port and others
	APIid    uint64 `gorm:"primaryKey;autoIncrement;not null;comment: API stands for Attacker Port Information"`
	LastTime int64  `gorm:"autoUpdateTime:nano;comment: last time that attacker lanuched an attack"`
	/* foreign key definition zone */
	AddrID      uint64
	AddrRelated AddrInfoDef `gorm:"foreignKey:AddrID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	/* defined for outer foreign key */
	CommandRelated          []PortCmdRelated    `gorm:"foreignKey:LoginedID;"`
	PassRelated             []PortPassRelated   `gorm:"foreignKey:PassID;"`
	PubKeyRelated           []PortPubKeyRelated `gorm:"foreignKey:PubKeyID;"`
	SSHClientVersionRelated []PortVerRelated    `gorm:"foreignKey:SSHClientVersionID;"`
	/* For minimize the adjusting count of memory */
	// -=-=-=-=-=-=-=-=-=-=-=-
	Port uint16 /* remote port */
	// -=-=-=-=-=-=-=-=-=-=-=-
}

type SSHClientversionStrDef struct {
	VerID           uint64 `gorm:"primaryKey;autoIncrement;"`
	ClientVersion   string `gorm:"unique;not null"` // SSH client version string
	FirstRecordTime int64  `gorm:"autoCreateTime:nano"`
	UpdatedTime     int64  `gorm:"autoUpdateTime:nano"`

	/* defined for outer foreign key */
	PortRelated []PortVerRelated `gorm:"foreignKey:SSHClientVersionID;"`
}

type UsernameDef struct {
	UserID   uint64 `gorm:"primaryKey"`
	Username string `gorm:"unique;"` // What if someone might send an empty string as username?
	/* Time */
	FirstRecordTime int64 `gorm:"autoCreateTime:nano"`
	UpdatedTime     int64 `gorm:"autoUpdateTime:nano"`
	/* defined for outer foreign key */
	PortRelated []PortNameRelated `gorm:"foreignKey:UsernameID;"`
}

type PassInfoDef struct {
	PasswordID uint64 `gorm:"primaryKey"`
	Password   string `gorm:"unique;uniqueIndex:password_idx;comment: Password attempted to login into system"`
	/* Time */
	FirstRecordTime int64 `gorm:"autoCreateTime:nano"`
	UpdatedTime     int64 `gorm:"autoUpdateTime:nano"`
	/* defined for outer foreign key */
	PortRelated []PortPassRelated `gorm:"foreignKey:PassID;"`
}

type PubInfoDef struct {
	PubKeyID uint64 `gorm:"primaryKey"`
	// TODO: suspect huge perform loss with long public Key and unique restrict
	PubKeyFingerprint string `gorm:"unique;not null;uniqueIndex:public_key_idx;comment: Public key fingerprint used by attacker"`
	/* Time */
	FirstRecordTime int64 `gorm:"autoCreateTime:nano"`
	UpdatedTime     int64 `gorm:"autoUpdateTime:nano"`
	/* defined for outer foreign key */
	PortRelated []PortPubKeyRelated `gorm:"foreignKey:PubKeyID;"`
}

type CommandTextDef struct {
	CmdID uint64 `gorm:"primaryKey;autoIncrement;comment: Command ID"`
	CMD   string `gorm:"unique;not null;"`
	/* defined for outer foreign key */
	CommandRelated []PortCmdRelated `gorm:"foreignKey:CmdID"`
}

type PortVerRelated struct {
	APIid                   uint64
	PortRelated             PortInfoDef `gorm:"foreignKey:APIid;references:APIid;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	SSHClientVersionID      uint64
	SSHClientVersionRelated SSHClientversionStrDef `gorm:"foreignKey:SSHClientVersionID;references:VerID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

type PortNameRelated struct {
	APIid       uint64
	PortRelated PortInfoDef `gorm:"foreignKey:APIid;references:APIid;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	UsernameID  uint64
	NameRelated UsernameDef `gorm:"foreignKey:UsernameID;references:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

type PortPassRelated struct {
	APIid       uint64
	PortRelated PortInfoDef `gorm:"foreignKey:APIid;references:APIid;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	PassID      uint64
	PassRelated PassInfoDef `gorm:"foreignKey:PassID;references:PasswordID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

type PortPubKeyRelated struct {
	APIid         uint64
	PortRelated   PortInfoDef `gorm:"foreignKey:APIid;references:APIid;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	PubKeyID      uint64
	PubKeyRelated PubInfoDef `gorm:"foreignKey:PubKeyID;references:PubKeyID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

type PortCmdRelated struct {
	RequestTime int64 `gorm:"autoUpdateTime:nano;comment: Time that requests for executing command"`
	/* foreign key definition zone */
	CmdID          uint64
	CommandRelated CommandTextDef `gorm:"primaryKey;foreignKey:CmdID;references:CmdID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	LoginedID      uint64
	PortRelated    PortInfoDef `gorm:"primaryKey;foreignKey:LoginedID;references:APIid;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}
