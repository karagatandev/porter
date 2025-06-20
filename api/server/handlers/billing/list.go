package billing

import (
	"context"
	"fmt"
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/telemetry"
)

// ListBillingHandler is a handler for listing payment methods
type ListBillingHandler struct {
	handlers.PorterHandlerWriter
}

// CheckPaymentEnabledHandler is a handler for checking if payment is setup
type CheckPaymentEnabledHandler struct {
	handlers.PorterHandlerWriter
}

// NewListBillingHandler will create a new ListBillingHandler
func NewListBillingHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *ListBillingHandler {
	return &ListBillingHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (c *ListBillingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.NewSpan(r.Context(), "serve-list-payment-methods")
	defer span.End()

	proj, _ := ctx.Value(types.ProjectScope).(*models.Project)

	paymentMethods, err := c.Config().BillingManager.StripeClient.ListPaymentMethod(ctx, proj.BillingID)
	if err != nil {
		err := telemetry.Error(ctx, span, err, "error listing payment method")
		c.HandleAPIError(w, r, apierrors.NewErrInternal(fmt.Errorf("error listing payment method: %w", err)))
		return
	}

	c.WriteResult(w, r, paymentMethods)
}

// NewCheckPaymentEnabledHandler will create a new CheckPaymentEnabledHandler
func NewCheckPaymentEnabledHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *CheckPaymentEnabledHandler {
	return &CheckPaymentEnabledHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (c *CheckPaymentEnabledHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.NewSpan(r.Context(), "serve-check-payment-enabled")
	defer span.End()

	proj, _ := ctx.Value(types.ProjectScope).(*models.Project)
	currentUser, _ := ctx.Value(types.UserScope).(*models.User)

	err := c.ensureBillingSetup(ctx, proj, currentUser)
	if err != nil {
		err := telemetry.Error(ctx, span, err, "error ensuring billing setup")
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "project-id", Value: proj.ID},
		telemetry.AttributeKV{Key: "customer-id", Value: proj.BillingID},
	)

	paymentEnabled, err := c.Config().BillingManager.StripeClient.CheckPaymentEnabled(ctx, proj.BillingID)
	if err != nil {
		err := telemetry.Error(ctx, span, err, "error checking if payment enabled")
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "payment-enabled", Value: paymentEnabled},
	)

	c.WriteResult(w, r, paymentEnabled)
}

func (c *CheckPaymentEnabledHandler) ensureBillingSetup(ctx context.Context, proj *models.Project, user *models.User) (err error) {
	ctx, span := telemetry.NewSpan(ctx, "ensure-billing-setup")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "billing-id", Value: proj.BillingID},
	)

	if proj.BillingID == "" {
		adminUser, err := c.getAdminUser(ctx, proj.ID)
		if err != nil {
			return telemetry.Error(ctx, span, err, "error getting admin user")
		}

		// If the admin user is not found, use the current user as last resort
		if adminUser == nil {
			adminUser = user
		}

		// Create billing customer for project and set the billing ID if it doesn't exist
		err = c.ensureStripeCustomerExists(ctx, adminUser.Email, proj)
		if err != nil {
			return telemetry.Error(ctx, span, err, "error ensuring Stripe customer exists")
		}
	}

	lagoCustomerExists := false
	if !lagoCustomerExists {
		adminUser, err := c.getAdminUser(ctx, proj.ID)
		if err != nil {
			return telemetry.Error(ctx, span, err, "error getting admin user")
		}

		// Create usage customer for project and set the usage ID if it doesn't exist
		err = c.ensureLagoCustomerExists(ctx, adminUser.Email, proj)
		if err != nil {
			return telemetry.Error(ctx, span, err, "error ensuring Lago customer exists")
		}
	}

	return nil
}

func (c *CheckPaymentEnabledHandler) getAdminUser(ctx context.Context, projectID uint) (adminUser *models.User, err error) {
	ctx, span := telemetry.NewSpan(ctx, "get-project-admin-role")
	defer span.End()

	// Get project roles
	roles, err := c.Repo().Project().ListProjectRolesOrdered(projectID)
	if err != nil {
		return adminUser, telemetry.Error(ctx, span, err, "error listing project roles")
	}

	// Get the project admin user
	for _, role := range roles {
		if role.Kind != types.RoleAdmin {
			continue
		}

		adminUser, err = c.Repo().User().ReadUser(role.UserID)
		if err != nil {
			// If the user is not found, continue to the next role
			continue
		}
		break
	}

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "admin-user-id", Value: adminUser.ID},
		telemetry.AttributeKV{Key: "admin-user-email", Value: adminUser.Email},
	)

	return adminUser, nil
}

func (c *CheckPaymentEnabledHandler) ensureStripeCustomerExists(ctx context.Context, adminUserEmail string, proj *models.Project) (err error) {
	ctx, span := telemetry.NewSpan(ctx, "ensure-stripe-customer-exists")
	defer span.End()

	if !c.Config().BillingManager.StripeConfigLoaded || !proj.GetFeatureFlag(models.BillingEnabled, c.Config().LaunchDarklyClient) || proj.BillingID != "" {
		return nil
	}

	billingID, err := c.Config().BillingManager.StripeClient.CreateCustomer(ctx, adminUserEmail, proj.ID, proj.Name)
	if err != nil {
		return telemetry.Error(ctx, span, err, "error creating billing customer")
	}

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "billing-id", Value: proj.BillingID},
	)

	proj.BillingID = billingID

	_, err = c.Repo().Project().UpdateProject(proj)
	if err != nil {
		return telemetry.Error(ctx, span, err, "error updating project")
	}

	return nil
}

func (c *CheckPaymentEnabledHandler) ensureLagoCustomerExists(ctx context.Context, adminUserEmail string, proj *models.Project) (err error) {
	ctx, span := telemetry.NewSpan(ctx, "ensure-lago-customer-exists")
	defer span.End()

	if !c.Config().BillingManager.LagoConfigLoaded || !proj.GetFeatureFlag(models.LagoEnabled, c.Config().LaunchDarklyClient) {
		return nil
	}

	// Check if the customer already exists
	exists, err := c.Config().BillingManager.LagoClient.CheckIfCustomerExists(ctx, proj.ID, proj.EnableSandbox)
	if err != nil {
		return telemetry.Error(ctx, span, err, "error while checking if customer exists")
	}

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "customer-exists", Value: exists},
	)

	if exists {
		return nil
	}

	err = c.Config().BillingManager.LagoClient.CreateCustomerWithPlan(ctx, adminUserEmail, proj.Name, proj.ID, proj.BillingID, proj.EnableSandbox)
	if err != nil {
		return telemetry.Error(ctx, span, err, "error creating Lago customer")
	}

	return nil
}
