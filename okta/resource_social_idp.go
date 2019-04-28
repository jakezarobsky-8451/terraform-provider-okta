package okta

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/okta/okta-sdk-golang/okta"
)

func resourceSocialIdp() *schema.Resource {
	return &schema.Resource{
		Create: resourceSocialIdpCreate,
		Read:   resourceSocialIdpRead,
		Update: resourceSocialIdpUpdate,
		Delete: resourceIdpDelete,
		Exists: getIdentityProviderExists(&SAMLIdentityProvider{}),
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		// Note the base schema
		Schema: buildIdpSchema(map[string]*schema.Schema{
			"authorization_url":     optUrlSchema,
			"authorization_binding": optBindingSchema,
			"token_url":             optUrlSchema,
			"token_binding":         optBindingSchema,
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice(
					[]string{"OIDC", "FACEBOOK", "LINKEDIN", "MICROSOFT", "GOOGLE"},
					false,
				),
			},
			"scopes": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
			},
			"protocol_type": &schema.Schema{
				Type:         schema.TypeString,
				Default:      "OIDC",
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"OIDC", "OAUTH2"}, false),
			},
			"client_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"client_secret": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"max_clock_skew": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
		}),
	}
}

func resourceSocialIdpCreate(d *schema.ResourceData, m interface{}) error {
	idp := buildSocialIdp(d)
	if err := createIdp(m, idp); err != nil {
		return err
	}
	d.SetId(idp.ID)

	if err := setIdpStatus(idp.ID, idp.Status, d.Get("status").(string), m); err != nil {
		return err
	}

	return resourceSocialIdpRead(d, m)
}

func resourceSocialIdpRead(d *schema.ResourceData, m interface{}) error {
	idp := &OIDCIdentityProvider{}

	if err := fetchIdp(d.Id(), m, idp); err != nil {
		return err
	}

	d.Set("name", idp.Name)
	d.Set("max_clock_skew", idp.Policy.MaxClockSkew)
	d.Set("provisioning_action", idp.Policy.Provisioning.Action)
	d.Set("deprovisioned_action", idp.Policy.Provisioning.Conditions.Deprovisioned)
	d.Set("suspended_action", idp.Policy.Provisioning.Conditions.Suspended)
	d.Set("profile_master", idp.Policy.Provisioning.ProfileMaster)
	d.Set("groups_action", idp.Policy.Provisioning.Groups.Action)
	d.Set("subject_match_type", idp.Policy.Subject.MatchType)
	d.Set("username_template", idp.Policy.Subject.UserNameTemplate.Template)
	d.Set("client_id", idp.Protocol.Credentials.Client.ClientID)
	d.Set("client_secret", idp.Protocol.Credentials.Client.ClientSecret)

	if idp.IssuerMode != "" {
		d.Set("issuer_mode", idp.IssuerMode)
	}

	if idp.Policy.AccountLink != nil {
		d.Set("account_link_action", idp.Policy.AccountLink.Action)
		d.Set("account_link_group_include", idp.Policy.AccountLink.Filter)
	}

	return setNonPrimitives(d, map[string]interface{}{
		"scopes": convertStringSetToInterface(idp.Protocol.Scopes),
	})
}

func resourceSocialIdpUpdate(d *schema.ResourceData, m interface{}) error {
	idp := buildSocialIdp(d)
	d.Partial(true)

	if err := updateIdp(d.Id(), m, idp); err != nil {
		return err
	}

	d.Partial(false)

	if err := setIdpStatus(idp.ID, idp.Status, d.Get("status").(string), m); err != nil {
		return err
	}

	return resourceSocialIdpRead(d, m)
}

func buildSocialIdp(d *schema.ResourceData) *OIDCIdentityProvider {
	return &OIDCIdentityProvider{
		Name: d.Get("name").(string),
		Type: d.Get("type").(string),
		Policy: &OIDCPolicy{
			AccountLink:  NewAccountLink(d),
			MaxClockSkew: int64(d.Get("max_clock_skew").(int)),
			Provisioning: NewIdpProvisioning(d),
			Subject: &OIDCSubject{
				MatchType: d.Get("subject_match_type").(string),
				UserNameTemplate: &okta.ApplicationCredentialsUsernameTemplate{
					Template: d.Get("username_template").(string),
				},
			},
		},
		Protocol: &OIDCProtocol{
			Scopes: convertInterfaceToStringSet(d.Get("scopes")),
			Type:   d.Get("protocol_type").(string),
			Credentials: &OIDCCredentials{
				Client: &OIDCClient{
					ClientID:     d.Get("client_id").(string),
					ClientSecret: d.Get("client_secret").(string),
				},
			},
		},
	}
}
