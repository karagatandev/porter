package startup_migrations

import (
	"github.com/karagatandev/porter/cmd/migrate/enable_cluster_preview_envs"
	"github.com/karagatandev/porter/internal/features"
	lr "github.com/karagatandev/porter/pkg/logger"
	"gorm.io/gorm"
)

// this should be incremented with every new startup migration script
const LatestMigrationVersion uint = 1

type migrationFunc func(db *gorm.DB, config *features.Client, logger *lr.Logger) error

var StartupMigrations = make(map[uint]migrationFunc)

func init() {
	StartupMigrations[1] = enable_cluster_preview_envs.EnableClusterPreviewEnvs
}
