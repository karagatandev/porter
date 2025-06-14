package test

import (
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/repository"
)

type DatabaseRepository struct{}

func NewDatabaseRepository() repository.DatabaseRepository {
	return &DatabaseRepository{}
}

func (repo *DatabaseRepository) CreateDatabase(database *models.Database) (*models.Database, error) {
	panic("unimplemented")
}

func (repo *DatabaseRepository) ReadDatabase(projectID, clusterID, databaseID uint) (*models.Database, error) {
	panic("unimplemented")
}

func (repo *DatabaseRepository) ReadDatabaseByInfraID(projectID, infraID uint) (*models.Database, error) {
	panic("unimplemented")
}

func (repo *DatabaseRepository) UpdateDatabase(database *models.Database) (*models.Database, error) {
	panic("unimplemented")
}

func (repo *DatabaseRepository) DeleteDatabase(projectID, clusterID, databaseID uint) error {
	panic("unimplemented")
}

func (repo *DatabaseRepository) ListDatabases(projectID, clusterID uint) ([]*models.Database, error) {
	panic("unimplemented")
}
