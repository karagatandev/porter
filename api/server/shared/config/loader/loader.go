package loader

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	gorillaws "github.com/gorilla/websocket"
	"github.com/karagatandev/porter/api/server/shared/apierrors/alerter"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/server/shared/config/env"
	"github.com/karagatandev/porter/api/server/shared/config/envloader"
	"github.com/karagatandev/porter/api/server/shared/websocket"
	"github.com/karagatandev/porter/ee/integrations/vault"
	"github.com/karagatandev/porter/internal/adapter"
	"github.com/karagatandev/porter/internal/analytics"
	"github.com/karagatandev/porter/internal/auth/sessionstore"
	"github.com/karagatandev/porter/internal/auth/token"
	"github.com/karagatandev/porter/internal/billing"
	"github.com/karagatandev/porter/internal/features"
	"github.com/karagatandev/porter/internal/helm/urlcache"
	"github.com/karagatandev/porter/internal/integrations/cloudflare"
	"github.com/karagatandev/porter/internal/integrations/dns"
	"github.com/karagatandev/porter/internal/integrations/powerdns"
	"github.com/karagatandev/porter/internal/notifier"
	"github.com/karagatandev/porter/internal/notifier/sendgrid"
	"github.com/karagatandev/porter/internal/oauth"
	"github.com/karagatandev/porter/internal/repository/credentials"
	"github.com/karagatandev/porter/internal/repository/gorm"
	"github.com/karagatandev/porter/internal/telemetry"
	lr "github.com/karagatandev/porter/pkg/logger"
	"github.com/karagatandev/porter/provisioner/client"
	ory "github.com/ory/client-go"
	"github.com/porter-dev/api-contracts/generated/go/porter/v1/porterv1connect"
	pgorm "gorm.io/gorm"
)

var (
	// InstanceEnvConf holds the environment configuration
	InstanceEnvConf *envloader.EnvConf
	// InstanceDB holds the config for connecting to the database
	InstanceDB *pgorm.DB
)

type EnvConfigLoader struct {
	version string
}

func NewEnvLoader(version string) config.ConfigLoader {
	return &EnvConfigLoader{version}
}

func sharedInit() {
	var err error
	InstanceEnvConf, _ = envloader.FromEnv()

	InstanceDB, err = adapter.New(InstanceEnvConf.DBConf)
	if err != nil {
		panic(err)
	}
}

func (e *EnvConfigLoader) LoadConfig() (res *config.Config, err error) {
	// ctx := context.Background()

	envConf := InstanceEnvConf
	sc := envConf.ServerConf

	if envConf == nil {
		return nil, errors.New("nil environment config passed to loader")
	}

	var instanceCredentialBackend credentials.CredentialStorage
	if envConf.DBConf.VaultEnabled {
		if envConf.DBConf.VaultAPIKey == "" || envConf.DBConf.VaultServerURL == "" || envConf.DBConf.VaultPrefix == "" {
			return nil, errors.New("vault is enabled but missing required environment variables [VAULT_API_KEY,VAULT_SERVER_URL,VAULT_PREFIX]")
		}

		instanceCredentialBackend = vault.NewClient(
			envConf.DBConf.VaultServerURL,
			envConf.DBConf.VaultAPIKey,
			envConf.DBConf.VaultPrefix,
		)
	}

	res = &config.Config{
		Logger:            lr.NewConsole(sc.Debug),
		ServerConf:        sc,
		DBConf:            envConf.DBConf,
		RedisConf:         envConf.RedisConf,
		CredentialBackend: instanceCredentialBackend,
	}
	res.Logger.Info().Msg("Loading MetadataFromConf")
	res.Metadata = config.MetadataFromConf(envConf.ServerConf, e.version)
	res.Logger.Info().Msg("Loaded MetadataFromConf")
	res.DB = InstanceDB

	var key [32]byte

	for i, b := range []byte(envConf.DBConf.EncryptionKey) {
		key[i] = b
	}

	res.Logger.Info().Msg("Creating new gorm repository")
	res.Repo = gorm.NewRepository(InstanceDB, &key, instanceCredentialBackend)
	res.Logger.Info().Msg("Created new gorm repository")

	res.Logger.Info().Msg("Creating new session store")
	// create the session store
	res.Store, err = sessionstore.NewStore(
		&sessionstore.NewStoreOpts{
			SessionRepository: res.Repo.Session(),
			CookieSecrets:     envConf.ServerConf.CookieSecrets,
			Insecure:          envConf.ServerConf.CookieInsecure,
		},
	)
	if err != nil {
		return nil, err
	}
	res.Logger.Info().Msg("Created new session store")

	res.TokenConf = &token.TokenGeneratorConf{
		TokenSecret: envConf.ServerConf.TokenGeneratorSecret,
	}

	res.UserNotifier = &notifier.EmptyUserNotifier{}

	if res.Metadata.Email {
		res.Logger.Info().Msg("Creating new user notifier")
		res.UserNotifier = sendgrid.NewUserNotifier(&sendgrid.UserNotifierOpts{
			SharedOpts: &sendgrid.SharedOpts{
				APIKey:      envConf.ServerConf.SendgridAPIKey,
				SenderEmail: envConf.ServerConf.SendgridSenderEmail,
			},
			PWResetTemplateID:       envConf.ServerConf.SendgridPWResetTemplateID,
			PWGHTemplateID:          envConf.ServerConf.SendgridPWGHTemplateID,
			VerifyEmailTemplateID:   envConf.ServerConf.SendgridVerifyEmailTemplateID,
			ProjectInviteTemplateID: envConf.ServerConf.SendgridProjectInviteTemplateID,
			DeleteProjectTemplateID: envConf.ServerConf.SendgridDeleteProjectTemplateID,
		})
		res.Logger.Info().Msg("Created new user notifier")
	}

	res.Alerter = alerter.NoOpAlerter{}

	if envConf.ServerConf.SentryDSN != "" {
		res.Logger.Info().Msg("Creating Sentry Alerter")
		res.Alerter, err = alerter.NewSentryAlerter(envConf.ServerConf.SentryDSN, envConf.ServerConf.SentryEnv)
		if err != nil {
			return nil, fmt.Errorf("failed to create new sentry alerter: %w", err)
		}
		res.Logger.Info().Msg("Created Sentry Alerter")
	}

	if sc.DOClientID != "" && sc.DOClientSecret != "" {
		res.Logger.Info().Msg("Creating Digital Ocean client")
		res.DOConf = oauth.NewDigitalOceanClient(&oauth.Config{
			ClientID:     sc.DOClientID,
			ClientSecret: sc.DOClientSecret,
			Scopes:       []string{"read", "write"},
			BaseURL:      sc.ServerURL,
		})
		res.Logger.Info().Msg("Created Digital Ocean client")
	}

	if sc.GoogleClientID != "" && sc.GoogleClientSecret != "" {
		res.Logger.Info().Msg("Creating Google client")
		res.GoogleConf = oauth.NewGoogleClient(&oauth.Config{
			ClientID:     sc.GoogleClientID,
			ClientSecret: sc.GoogleClientSecret,
			Scopes: []string{
				"openid",
				"profile",
				"email",
			},
			BaseURL: sc.ServerURL,
		})
		res.Logger.Info().Msg(" Google client")
	}

	// TODO: remove this as part of POR-1055
	if sc.GithubClientID != "" && sc.GithubClientSecret != "" {
		res.Logger.Info().Msg("Creating Github client")
		res.GithubConf = oauth.NewGithubClient(&oauth.Config{
			ClientID:     sc.GithubClientID,
			ClientSecret: sc.GithubClientSecret,
			Scopes:       []string{"read:user", "user:email"},
			BaseURL:      sc.ServerURL,
		})
		res.Logger.Info().Msg("Created Github client")
	}

	if sc.GithubAppSecretBase64 != "" {
		if sc.GithubAppSecretPath == "" {
			sc.GithubAppSecretPath = "github-app-secret-key"
		}
		_, err := os.Stat(sc.GithubAppSecretPath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("GITHUB_APP_SECRET_BASE64 provided, but error checking if GITHUB_APP_SECRET_PATH exists: %w", err)
			}
			secret, err := base64.StdEncoding.DecodeString(sc.GithubAppSecretBase64)
			if err != nil {
				return nil, fmt.Errorf("GITHUB_APP_SECRET_BASE64 provided, but error decoding: %w", err)
			}
			_, err = createDirectoryRecursively(sc.GithubAppSecretPath)
			if err != nil {
				return nil, fmt.Errorf("GITHUB_APP_SECRET_BASE64 provided, but error creating directory for GITHUB_APP_SECRET_PATH: %w", err)
			}
			err = os.WriteFile(sc.GithubAppSecretPath, secret, os.ModePerm)
			if err != nil {
				return nil, fmt.Errorf("GITHUB_APP_SECRET_BASE64 provided, but error writing to GITHUB_APP_SECRET_PATH: %w", err)
			}
		}
	}

	if sc.GithubAppClientID != "" &&
		sc.GithubAppClientSecret != "" &&
		sc.GithubAppName != "" &&
		sc.GithubAppWebhookSecret != "" &&
		sc.GithubAppSecretPath != "" &&
		sc.GithubAppID != "" {
		if AppID, err := strconv.ParseInt(sc.GithubAppID, 10, 64); err == nil {
			res.GithubAppConf = oauth.NewGithubAppClient(&oauth.Config{
				ClientID:     sc.GithubAppClientID,
				ClientSecret: sc.GithubAppClientSecret,
				Scopes:       []string{"read:user"},
				BaseURL:      sc.ServerURL,
			}, sc.GithubAppName, sc.GithubAppWebhookSecret, sc.GithubAppSecretPath, AppID)
		}

		secret, err := ioutil.ReadFile(sc.GithubAppSecretPath)
		if err != nil {
			return nil, fmt.Errorf("could not read github app secret: %s", err)
		}

		sc.GithubAppSecret = append(sc.GithubAppSecret, secret...)
	}

	launchDarklyClient, err := features.GetClient(envConf.ServerConf.FeatureFlagClient, envConf.ServerConf.LaunchDarklySDKKey)
	if err != nil {
		return nil, fmt.Errorf("could not create launch darkly client: %s", err)
	}
	res.LaunchDarklyClient = launchDarklyClient

	if sc.SlackClientID != "" && sc.SlackClientSecret != "" {
		res.Logger.Info().Msg("Creating Slack client")
		res.SlackConf = oauth.NewSlackClient(&oauth.Config{
			ClientID:     sc.SlackClientID,
			ClientSecret: sc.SlackClientSecret,
			Scopes: []string{
				"incoming-webhook",
				"team:read",
			},
			BaseURL: sc.ServerURL,
		})
		res.Logger.Info().Msg("Created Slack client")
	}

	if sc.UpstashEnabled && sc.UpstashClientID != "" {
		res.Logger.Info().Msg("Creating Upstash client")
		res.UpstashConf = oauth.NewUpstashClient(&oauth.Config{
			ClientID:     sc.UpstashClientID,
			ClientSecret: "", // Upstash doesn't require a secret
			Scopes:       []string{"offline_access"},
			BaseURL:      sc.ServerURL,
		})
		res.Logger.Info().Msg("Created Upstash client")
	}

	if sc.NeonEnabled && sc.NeonClientID != "" && sc.NeonClientSecret != "" {
		res.Logger.Info().Msg("Creating Neon client")
		res.NeonConf = oauth.NewNeonClient(&oauth.Config{
			ClientID:     sc.NeonClientID,
			ClientSecret: sc.NeonClientSecret,
			Scopes:       []string{"urn:neoncloud:projects:create", "urn:neoncloud:projects:read", "urn:neoncloud:projects:update", "urn:neoncloud:projects:delete", "offline", "offline_access"},
			BaseURL:      sc.ServerURL,
		})
		res.Logger.Info().Msg("Created Neon client")
	}

	res.WSUpgrader = &websocket.Upgrader{
		WSUpgrader: &gorillaws.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				var err error
				defer func() {
					// TODO: this is only used to collect data for removing the `request origin not allowed by Upgrader.CheckOrigin` error
					if err != nil {
						res.Logger.Info().Msgf("error: %s, host: %s, origin: %s, serverURL: %s", err.Error(), r.Host, r.Header.Get("Origin"), sc.ServerURL)
					}
				}()
				return true
			},
		},
	}

	// construct the whitelisted users map
	wlUsers := make(map[uint]uint)

	for _, userID := range sc.WhitelistedUsers {
		wlUsers[userID] = userID
	}

	res.WhitelistedUsers = wlUsers
	res.Logger.Info().Msg("Creating URL Cache")
	res.URLCache = urlcache.Init(sc.DefaultApplicationHelmRepoURL, sc.DefaultAddonHelmRepoURL)
	res.Logger.Info().Msg("Created URL Cache")

	res.Logger.Info().Msg("Creating provisioner service client")
	provClient, err := getProvisionerServiceClient(sc)
	if err == nil && provClient != nil {
		res.ProvisionerClient = provClient
	}
	res.Logger.Info().Msg("Created provisioner service client")

	res.Logger.Info().Msg("Creating analytics client")
	res.AnalyticsClient = analytics.InitializeAnalyticsSegmentClient(sc.SegmentClientKey, res.Logger)
	res.Logger.Info().Msg("Created analytics client")

	switch sc.DnsProvider {
	case "powerdns":
		if sc.PowerDNSAPIKey != "" && sc.PowerDNSAPIServerURL != "" {
			res.DNSClient = &dns.Client{Client: powerdns.NewClient(sc.PowerDNSAPIServerURL, sc.PowerDNSAPIKey, sc.AppRootDomain)}
		}
	case "cloudflare":
		if sc.CloudflareAPIToken != "" {
			cloudflareClient, err := cloudflare.NewClient(sc.CloudflareAPIToken, sc.AppRootDomain)
			if err != nil {
				return res, fmt.Errorf("unable to create cloudflare client: %w", err)
			}

			res.DNSClient = &dns.Client{Client: cloudflareClient}
		}
	}

	res.EnableCAPIProvisioner = sc.EnableCAPIProvisioner
	if sc.EnableCAPIProvisioner {
		res.Logger.Info().Msg("Creating CCP client")
		if sc.ClusterControlPlaneAddress == "" {
			return res, errors.New("must provide CLUSTER_CONTROL_PLANE_ADDRESS")
		}
		client := porterv1connect.NewClusterControlPlaneServiceClient(http.DefaultClient, sc.ClusterControlPlaneAddress)
		res.ClusterControlPlaneClient = client
		res.Logger.Info().Msg("Created CCP client")
	}

	res.TelemetryConfig = telemetry.TracerConfig{
		ServiceName:  sc.TelemetryName,
		CollectorURL: sc.TelemetryCollectorURL,
	}

	var (
		stripeClient  billing.StripeClient
		stripeEnabled bool
		lagoClient    billing.LagoClient
		lagoEnabled   bool
	)
	if sc.StripeSecretKey != "" {
		stripeClient = billing.NewStripeClient(InstanceEnvConf.ServerConf.StripeSecretKey, InstanceEnvConf.ServerConf.StripePublishableKey)
		stripeEnabled = true
		res.Logger.Info().Msg("Stripe configuration loaded")
	} else {
		res.Logger.Info().Msg("STRIPE_SECRET_KEY not set, all Stripe functionality will be disabled")
	}

	if sc.LagoAPIKey != "" && sc.PorterCloudPlanCode != "" && sc.PorterStandardPlanCode != "" && sc.PorterTrialCode != "" {
		lagoClient, err = billing.NewLagoClient(InstanceEnvConf.ServerConf.LagoAPIKey, InstanceEnvConf.ServerConf.PorterCloudPlanCode, InstanceEnvConf.ServerConf.PorterStandardPlanCode, InstanceEnvConf.ServerConf.PorterTrialCode)
		if err != nil {
			return nil, fmt.Errorf("unable to create Lago client: %w", err)
		}
		lagoEnabled = true
		res.Logger.Info().Msg("Lago configuration loaded")
	} else {
		res.Logger.Info().Msg("LAGO_API_KEY, PORTER_CLOUD_PLAN_CODE, PORTER_STANDARD_PLAN_CODE and PORTER_TRIAL_CODE must be set, all Lago functionality will be disabled")
	}

	res.Logger.Info().Msg("Creating billing manager")
	res.BillingManager = billing.Manager{
		StripeClient:       stripeClient,
		StripeConfigLoaded: stripeEnabled,
		LagoClient:         lagoClient,
		LagoConfigLoaded:   lagoEnabled,
	}
	res.Logger.Info().Msg("Created billing manager")

	c := ory.NewConfiguration()
	c.Servers = ory.ServerConfigurations{{
		URL: InstanceEnvConf.ServerConf.OryUrl,
	}}

	if InstanceEnvConf.ServerConf.OryEnabled {
		res.Logger.Info().Msg("Creating Ory client")
		res.Ory = *ory.NewAPIClient(c)
		res.OryApiKeyContextWrapper = func(ctx context.Context) context.Context {
			return context.WithValue(ctx, ory.ContextAccessToken, InstanceEnvConf.ServerConf.OryApiKey)
		}
		res.OryActionKey = InstanceEnvConf.ServerConf.OryActionKey
		res.Logger.Info().Msg("Created Ory client")
	}

	return res, nil
}

func getProvisionerServiceClient(sc *env.ServerConf) (*client.Client, error) {
	if sc.ProvisionerServerURL != "" && sc.ProvisionerToken != "" {
		baseURL := fmt.Sprintf("%s/api/v1", sc.ProvisionerServerURL)

		pClient, err := client.NewClient(baseURL, sc.ProvisionerToken, 0)
		if err != nil {
			return nil, err
		}

		return pClient, nil
	}

	return nil, fmt.Errorf("required env vars not set for provisioner")
}

// createDirectoryRecursively creates a directory and all its parents if they don't exist
func createDirectoryRecursively(p string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(p), 0o770); err != nil {
		return nil, err
	}
	return os.Create(p)
}
