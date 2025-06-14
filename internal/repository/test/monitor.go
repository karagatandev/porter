package test

import (
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/repository"
)

type MonitorTestResultRepository struct{}

func NewMonitorTestResultRepository(canQuery bool) repository.MonitorTestResultRepository {
	return &MonitorTestResultRepository{}
}

func (n *MonitorTestResultRepository) CreateMonitorTestResult(monitor *models.MonitorTestResult) (*models.MonitorTestResult, error) {
	panic("not implemented") // TODO: Implement
}

func (n *MonitorTestResultRepository) ReadMonitorTestResult(projectID, clusterID uint, operationID string) (*models.MonitorTestResult, error) {
	panic("not implemented") // TODO: Implement
}

func (n *MonitorTestResultRepository) UpdateMonitorTestResult(monitor *models.MonitorTestResult) (*models.MonitorTestResult, error) {
	panic("not implemented") // TODO: Implement
}

func (n *MonitorTestResultRepository) ArchiveMonitorTestResults(projectID, clusterID uint, recommenderID string) error {
	panic("not implemented") // TODO: Implement
}

func (n *MonitorTestResultRepository) DeleteOldMonitorTestResults(projectID, clusterID uint, recommenderID string) error {
	panic("not implemented") // TODO: Implement
}
