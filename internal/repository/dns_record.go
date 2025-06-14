package repository

import (
	"github.com/karagatandev/porter/internal/models"
)

// DNSRecordRepository represents the set of queries on the
// DNSRecord model
type DNSRecordRepository interface {
	CreateDNSRecord(record *models.DNSRecord) (*models.DNSRecord, error)
}
