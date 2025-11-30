// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package databases

type DNSQuery struct {
	DQID            uint64 `gorm:"primaryKey;autoIncrement;not null"`
	DomainName      string
	Counter         uint64 `gorm:"default:0"`
	FirstRecordTime int64  `gorm:"autoCreateTime:nano"`
	UpdatedTime     int64  `gorm:"autoUpdateTime:nano"`
}
