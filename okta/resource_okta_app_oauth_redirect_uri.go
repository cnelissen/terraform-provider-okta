package okta

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/okta/terraform-provider-okta/sdk"
)

func resourceAppOAuthRedirectURI() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAppOAuthRedirectURICreate,
		ReadContext:   resourceFuncNoOp,
		UpdateContext: resourceAppOAuthRedirectURIUpdate,
		DeleteContext: resourceAppOAuthRedirectURIDelete,
		// The id for this is the uri
		Importer: createCustomNestedResourceImporter([]string{"app_id", "id"}, "Expecting the following format: <app_id>/<uri>"),
		Schema: map[string]*schema.Schema{
			"app_id": {
				Required:    true,
				Type:        schema.TypeString,
				ForceNew:    true,
				Description: "OAuth application ID.",
			},
			"uri": {
				Required:    true,
				Type:        schema.TypeString,
				Description: "Redirect URI to append to Okta OIDC application.",
			},
		},
	}
}

func resourceAppOAuthRedirectURICreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	err := appendRedirectURI(ctx, d, m)
	if err != nil {
		return diag.Errorf("failed to create redirect URI: %v", err)
	}
	d.SetId(d.Get("uri").(string))
	return resourceFuncNoOp(ctx, d, m)
}

func resourceAppOAuthRedirectURIUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := appendRedirectURI(ctx, d, m); err != nil {
		return diag.Errorf("failed to update redirect URI: %v", err)
	}
	// Normally not advisable, but ForceNew generated unnecessary calls
	d.SetId(d.Get("uri").(string))
	return resourceFuncNoOp(ctx, d, m)
}

func resourceAppOAuthRedirectURIDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	appID := d.Get("app_id").(string)

	oktaMutexKV.Lock(appID)
	defer oktaMutexKV.Unlock(appID)

	app := sdk.NewOpenIdConnectApplication()
	err := fetchAppByID(ctx, appID, m, app)
	if err != nil {
		return diag.Errorf("failed to get application: %v", err)
	}
	if app.Id == "" {
		return diag.Errorf("application with id %s does not exist", appID)
	}
	if !contains(app.Settings.OauthClient.RedirectUris, d.Id()) {
		logger(m).Info(fmt.Sprintf("application with appID %s does not have redirect URI %s", appID, d.Id()))
		return nil
	}
	app.Settings.OauthClient.RedirectUris = remove(app.Settings.OauthClient.RedirectUris, d.Id())
	err = updateAppByID(ctx, appID, m, app)
	if err != nil {
		return diag.Errorf("failed to delete redirect URI: %v", err)
	}
	return nil
}

func appendRedirectURI(ctx context.Context, d *schema.ResourceData, m interface{}) error {
	appID := d.Get("app_id").(string)

	oktaMutexKV.Lock(appID)
	defer oktaMutexKV.Unlock(appID)

	app := sdk.NewOpenIdConnectApplication()
	if err := fetchAppByID(ctx, appID, m, app); err != nil {
		return err
	}
	if app.Id == "" {
		return fmt.Errorf("application with id %s does not exist", appID)
	}
	if contains(app.Settings.OauthClient.RedirectUris, d.Id()) {
		logger(m).Info(fmt.Sprintf("application with appID %s already has redirect URI %s", appID, d.Id()))
		return nil
	}
	uri := d.Get("uri").(string)
	app.Settings.OauthClient.RedirectUris = append(app.Settings.OauthClient.RedirectUris, uri)
	return updateAppByID(ctx, appID, m, app)
}
