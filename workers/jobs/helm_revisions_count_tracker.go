//go:build ee

/*

                            === Helm Release Revisions Tracker Job ===

This job keeps a track of helm releases and their revisions and deletes older revisions once they are
backed up to an S3 bucket.

  - The job looks for clusters which have the `monitor_helm_releases` set to true.
  - The clusters are then checked for old helm release revisions.
  - In a cluster, list of all namespaces is fetched.
  - For every namespace, the list of releases is fetched.
  - For every release, its revision history is fetched.
  - If the number of revisions exceeds 100, then we intend to only keep the most recent 100 revisions.
  - For this, the older revisions are first backed up to an S3 bucket and then deleted.

*/

package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/karagatandev/porter/api/server/shared/config/env"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/pkg/logger"
	"github.com/karagatandev/porter/provisioner/integrations/storage/s3"
	"github.com/karagatandev/porter/workers/utils"

	"github.com/karagatandev/porter/ee/integrations/vault"
	"github.com/karagatandev/porter/internal/helm"
	"github.com/karagatandev/porter/internal/kubernetes"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/oauth"
	"github.com/karagatandev/porter/internal/repository"
	rcreds "github.com/karagatandev/porter/internal/repository/credentials"
	rgorm "github.com/karagatandev/porter/internal/repository/gorm"
	"github.com/stefanmcshane/helm/pkg/releaseutil"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type helmRevisionsCountTracker struct {
	enqueueTime        time.Time
	db                 *gorm.DB
	repo               repository.Repository
	doConf             *oauth2.Config
	awsAccessKeyID     string
	awsSecretAccessKey string
	awsRegion          string
	s3BucketName       string
	encryptionKey      *[32]byte
	revisionsCount     int
}

// HelmRevisionsCountTrackerOpts holds the options required to run this job
type HelmRevisionsCountTrackerOpts struct {
	DBConf             *env.DBConf
	DOClientID         string
	DOClientSecret     string
	DOScopes           []string
	ServerURL          string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AWSRegion          string
	S3BucketName       string
	EncryptionKey      string
	RevisionsCount     int
}

func NewHelmRevisionsCountTracker(
	ctx context.Context,
	db *gorm.DB,
	enqueueTime time.Time,
	opts *HelmRevisionsCountTrackerOpts,
) (*helmRevisionsCountTracker, error) {
	var credBackend rcreds.CredentialStorage

	if opts.DBConf.VaultAPIKey != "" && opts.DBConf.VaultServerURL != "" && opts.DBConf.VaultPrefix != "" {
		credBackend = vault.NewClient(
			opts.DBConf.VaultServerURL,
			opts.DBConf.VaultAPIKey,
			opts.DBConf.VaultPrefix,
		)
	}

	var key [32]byte

	for i, b := range []byte(opts.DBConf.EncryptionKey) {
		key[i] = b
	}

	repo := rgorm.NewRepository(db, &key, credBackend)

	doConf := oauth.NewDigitalOceanClient(&oauth.Config{
		ClientID:     opts.DOClientID,
		ClientSecret: opts.DOClientSecret,
		Scopes:       opts.DOScopes,
		BaseURL:      opts.ServerURL,
	})

	var s3Key [32]byte

	for i, b := range []byte(opts.EncryptionKey) {
		s3Key[i] = b
	}

	return &helmRevisionsCountTracker{
		enqueueTime, db, repo, doConf, opts.AWSAccessKeyID, opts.AWSSecretAccessKey, opts.AWSRegion,
		opts.S3BucketName, &s3Key, opts.RevisionsCount,
	}, nil
}

func (t *helmRevisionsCountTracker) ID() string {
	return "helm-revisions-count-tracker"
}

func (t *helmRevisionsCountTracker) EnqueueTime() time.Time {
	return t.enqueueTime
}

func (t *helmRevisionsCountTracker) Run(ctx context.Context) error {
	var count int64

	if err := t.db.Model(&models.Cluster{}).Count(&count).Error; err != nil {
		return err
	}

	var wg sync.WaitGroup

	for i := 0; i < (int(count)/stepSize)+1; i++ {
		var clusters []*models.Cluster

		if err := t.db.Order("id asc").Offset(i*stepSize).Limit(stepSize).Find(&clusters, "monitor_helm_releases = ?", "1").
			Error; err != nil {
			return err
		}

		// go through each project
		for _, cluster := range clusters {
			wg.Add(1)

			go func(projID, clusterID uint) {
				defer wg.Done()

				log.Printf("starting release revision monitoring for cluster with ID %d", cluster.ID)

				cluster, err := t.repo.Cluster().ReadCluster(projID, clusterID)
				if err != nil {
					log.Printf("error reading cluster ID %d: %v. skipping cluster ...", clusterID, err)
					return
				}

				// create s3 client to store revisions that need to be deleted
				s3Client, err := s3.NewS3StorageClient(&s3.S3Options{
					t.awsRegion, t.awsAccessKeyID, t.awsSecretAccessKey, t.s3BucketName, t.encryptionKey,
				})
				if err != nil {
					log.Printf("error creating S3 client for cluster ID %d: %v. skipping cluster ...", cluster.ID, err)
					return
				}

				k8sAgent, err := kubernetes.GetAgentOutOfClusterConfig(ctx, &kubernetes.OutOfClusterConfig{
					Cluster:                   cluster,
					Repo:                      t.repo,
					DigitalOceanOAuth:         t.doConf,
					AllowInClusterConnections: false,
					Timeout:                   5 * time.Second,
				})
				if err != nil {
					log.Printf("error getting k8s agent for cluster ID %d: %v. skipping cluster ...", cluster.ID, err)
					return
				}

				namespaces, err := k8sAgent.ListNamespaces()
				if err != nil {
					log.Printf("error fetching namespaces for cluster ID %d: %v. skipping cluster ...", cluster.ID, err)
					return
				}

				log.Printf("fetched %d namespaces for cluster ID %d", len(namespaces.Items), cluster.ID)

				for _, ns := range namespaces.Items {
					agent, err := utils.NewRetryHelmAgent(ctx, &helm.Form{
						Cluster:                   cluster,
						Namespace:                 ns.Name,
						Repo:                      t.repo,
						DigitalOceanOAuth:         t.doConf,
						AllowInClusterConnections: false,
						Timeout:                   5 * time.Second,
					}, logger.New(true, os.Stdout), 3, time.Second)
					if err != nil {
						log.Printf("error fetching helm client for namespace %s in cluster ID %d: %v. "+
							"skipping namespace ...", ns.Name, cluster.ID, err)
						continue
					}

					releases, err := agent.ListReleases(ctx, ns.GetName(), &types.ReleaseListFilter{
						ByDate: true,
						StatusFilter: []string{
							"deployed",
							"pending",
							"pending-install",
							"pending-upgrade",
							"pending-rollback",
							"failed",
						},
					})
					if err != nil {
						log.Printf("error fetching releases for namespace %s in cluster ID %d: %v. skipping namespace ...",
							ns.Name, cluster.ID, err)
						continue
					}

					log.Printf("fetched %d releases for namespace %s in cluster ID %d", len(releases), ns.Name, cluster.ID)

					for _, rel := range releases {
						revisions, err := agent.GetReleaseHistory(ctx, rel.Name)
						if err != nil {
							log.Printf("error fetching release history for release %s in namespace %s of cluster ID %d: %v."+
								" skipping release ...", rel.Name, ns.Name, cluster.ID, err)
							continue
						}

						if len(revisions) <= t.revisionsCount {
							log.Printf("release %s of namespace %s in cluster ID %d has <= %d revisions. "+
								"skipping release...", t.revisionsCount, rel.Name, ns.Name, cluster.ID)
							continue
						}

						log.Printf("release %s of namespace %s in cluster ID %d has more than %d revisions. attempting to "+
							"delete the older ones.", t.revisionsCount, rel.Name, ns.Name, cluster.ID)

						// sort revisions from newest to oldest
						releaseutil.Reverse(revisions, releaseutil.SortByRevision)

						for i := t.revisionsCount; i < len(revisions); i += 1 {
							rev := revisions[i]

							// store the revision in the s3 bucket before deleting it
							data, err := json.Marshal(rev)
							if err != nil {
								log.Printf("error marshalling revision for release %s, number %d: %v. skipping revision ...",
									rev.Name, rev.Version, err)
								continue
							}

							// write to the bucket with key - <project_id>/<cluster_id>/<namespace>/<release_name>/<revision_number>
							err = s3Client.WriteFileWithKey(data, true, fmt.Sprintf("%d/%d/%s/%s/%d", cluster.ProjectID,
								cluster.ID, rel.Namespace, rel.Name, rev.Version))

							if err != nil {
								log.Printf("error backing up revision for release %s, number %d: %v. skipping revision ...",
									rev.Name, rev.Version, err)
								continue
							}

							log.Printf("revision %d of release %s in namespace %s of cluster ID %d was successfully backed up.",
								rev.Version, rel.Name, ns.Name, cluster.ID)

							err = agent.DeleteReleaseRevision(ctx, rev.Name, rev.Version)

							if err != nil {
								log.Printf("error deleting revision %d of release %s in namespace %s of cluster ID %d: %v",
									rev.Version, rel.Name, ns.Name, cluster.ID, err)
								continue
							}

							log.Printf("revision %d of release %s in namespace %s of cluster ID %d was successfully deleted.",
								rev.Version, rel.Name, ns.Name, cluster.ID)
						}
					}
				}
			}(cluster.ProjectID, cluster.ID)
		}

		wg.Wait()
	}

	return nil
}

func (t *helmRevisionsCountTracker) SetData([]byte) {}
