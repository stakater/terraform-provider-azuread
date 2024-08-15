// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package applications

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-sdk/sdk/odata"
	"github.com/hashicorp/terraform-provider-azuread/internal/helpers"
	"github.com/hashicorp/terraform-provider-azuread/internal/sdk"
	"github.com/hashicorp/terraform-provider-azuread/internal/services/applications/parse"
	"github.com/hashicorp/terraform-provider-azuread/internal/tf"
	"github.com/hashicorp/terraform-provider-azuread/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azuread/internal/tf/validation"
	"github.com/manicminer/hamilton/msgraph"
)

type ApplicationRegistrationModel struct {
	ClientId                           string   `tfschema:"client_id"`
	Description                        string   `tfschema:"description"`
	DisabledByMicrosoft                string   `tfschema:"disabled_by_microsoft"`
	DisplayName                        string   `tfschema:"display_name"`
	GroupMembershipClaims              []string `tfschema:"group_membership_claims"`
	HomepageUrl                        string   `tfschema:"homepage_url"`
	ImplicitAccessTokenIssuanceEnabled bool     `tfschema:"implicit_access_token_issuance_enabled"`
	ImplicitIdTokenIssuanceEnabled     bool     `tfschema:"implicit_id_token_issuance_enabled"`
	LogoutUrl                          string   `tfschema:"logout_url"`
	MarketingUrl                       string   `tfschema:"marketing_url"`
	Notes                              string   `tfschema:"notes"`
	ObjectId                           string   `tfschema:"object_id"`
	PrivacyStatementUrl                string   `tfschema:"privacy_statement_url"`
	PublisherDomain                    string   `tfschema:"publisher_domain"`
	RequestedAccessTokenVersion        int      `tfschema:"requested_access_token_version"`
	ServiceManagementReference         string   `tfschema:"service_management_reference"`
	SignInAudience                     string   `tfschema:"sign_in_audience"`
	SupportUrl                         string   `tfschema:"support_url"`
	TermsOfServiceUrl                  string   `tfschema:"terms_of_service_url"`
}

var _ sdk.ResourceWithUpdate = ApplicationRegistrationResource{}

type ApplicationRegistrationResource struct{}

func (r ApplicationRegistrationResource) IDValidationFunc() pluginsdk.SchemaValidateFunc {
	return parse.ValidateApplicationID
}

func (r ApplicationRegistrationResource) ResourceType() string {
	return "azuread_application_registration"
}

func (r ApplicationRegistrationResource) ModelObject() interface{} {
	return &ApplicationRegistrationModel{}
}

func (r ApplicationRegistrationResource) Arguments() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{
		"display_name": {
			Description:  "The display name for the application",
			Type:         pluginsdk.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
		},

		"description": {
			Description:  "Description of the application as shown to end users",
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringLenBetween(0, 1024),
		},

		"group_membership_claims": {
			Description: "Configures the `groups` claim that the app expects issued in a user or OAuth access token",
			Type:        pluginsdk.TypeSet,
			Optional:    true,
			Elem: &pluginsdk.Schema{
				Type: pluginsdk.TypeString,
				ValidateFunc: validation.StringInSlice([]string{
					msgraph.GroupMembershipClaimAll,
					msgraph.GroupMembershipClaimNone,
					msgraph.GroupMembershipClaimApplicationGroup,
					msgraph.GroupMembershipClaimDirectoryRole,
					msgraph.GroupMembershipClaimSecurityGroup,
				}, false),
			},
		},

		"homepage_url": {
			Description:  "URL of the home page for the application",
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ValidateFunc: validation.IsHttpOrHttpsUrl,
		},

		"implicit_access_token_issuance_enabled": {
			Description: "Whether this application can request an access token using OAuth implicit flow",
			Type:        pluginsdk.TypeBool,
			Optional:    true,
		},

		"implicit_id_token_issuance_enabled": {
			Description: "Whether this application can request an ID token using OAuth implicit flow",
			Type:        pluginsdk.TypeBool,
			Optional:    true,
		},

		"logout_url": {
			Description:  "URL of the logout page for the application, where the session is cleared for single sign-out",
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ValidateFunc: validation.IsLogoutUrl,
		},

		"marketing_url": {
			Description:  "URL of the marketing page for the application",
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ValidateFunc: validation.IsHttpOrHttpsUrl,
		},

		"notes": {
			Description:  "User-specified notes relevant for the management of the application",
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringIsNotEmpty,
		},

		"privacy_statement_url": {
			Description:  "URL of the privacy statement for the application",
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ValidateFunc: validation.IsHttpOrHttpsUrl,
		},

		"requested_access_token_version": {
			Description:  "The access token version expected by this resource",
			Type:         pluginsdk.TypeInt,
			Optional:     true,
			Default:      2,
			ValidateFunc: validation.IntBetween(1, 2),
		},

		"service_management_reference": {
			Description:  "References application or contact information from a service or asset management database",
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringIsNotEmpty,
		},

		"sign_in_audience": {
			Description: "The Microsoft account types that are supported for the current application",
			Type:        pluginsdk.TypeString,
			Optional:    true,
			Default:     msgraph.SignInAudienceAzureADMyOrg,
			ValidateFunc: validation.StringInSlice([]string{
				msgraph.SignInAudienceAzureADMyOrg,
				msgraph.SignInAudienceAzureADMultipleOrgs,
				msgraph.SignInAudienceAzureADandPersonalMicrosoftAccount,
				msgraph.SignInAudiencePersonalMicrosoftAccount,
			}, false),
		},

		"support_url": {
			Description:  "URL of the support page for the application",
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ValidateFunc: validation.IsHttpOrHttpsUrl,
		},

		"terms_of_service_url": {
			Description:  "URL of the terms of service statement for the application",
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ValidateFunc: validation.IsHttpOrHttpsUrl,
		},
	}
}

func (r ApplicationRegistrationResource) Attributes() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{
		"client_id": {
			Description: "The Client ID (also called Application ID)",
			Type:        pluginsdk.TypeString,
			Computed:    true,
		},

		"disabled_by_microsoft": {
			Description: "If the application has been disabled by Microsoft, this shows the status or reason",
			Type:        pluginsdk.TypeString,
			Computed:    true,
		},

		"object_id": {
			Description: "The object ID of the application within the tenant",
			Type:        pluginsdk.TypeString,
			Computed:    true,
		},

		"publisher_domain": {
			Description: "The verified publisher domain for the application",
			Type:        pluginsdk.TypeString,
			Computed:    true,
		},
	}
}

func (r ApplicationRegistrationResource) Create() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 10 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			client := metadata.Client.Applications.ApplicationsClient
			client.BaseClient.DisableRetries = true
			defer func() { client.BaseClient.DisableRetries = false }()

			var model ApplicationRegistrationModel
			if err := metadata.Decode(&model); err != nil {
				return fmt.Errorf("decoding: %+v", err)
			}

			properties := msgraph.Application{
				DisplayName:                &model.DisplayName,
				Description:                tf.NullableString(model.Description),
				GroupMembershipClaims:      pointer.To(model.GroupMembershipClaims),
				Notes:                      tf.NullableString(model.Notes),
				ServiceManagementReference: tf.NullableString(model.ServiceManagementReference),
				SignInAudience:             &model.SignInAudience,

				Api: &msgraph.ApplicationApi{
					RequestedAccessTokenVersion: pointer.To(int32(model.RequestedAccessTokenVersion)),
				},

				Info: &msgraph.InformationalUrl{
					MarketingUrl:        tf.NullableString(model.MarketingUrl),
					PrivacyStatementUrl: tf.NullableString(model.PrivacyStatementUrl),
					SupportUrl:          tf.NullableString(model.SupportUrl),
					TermsOfServiceUrl:   tf.NullableString(model.TermsOfServiceUrl),
				},

				Web: &msgraph.ApplicationWeb{
					HomePageUrl: tf.NullableString(model.HomepageUrl),
					LogoutUrl:   tf.NullableString(model.LogoutUrl),

					ImplicitGrantSettings: &msgraph.ImplicitGrantSettings{
						EnableAccessTokenIssuance: pointer.To(model.ImplicitAccessTokenIssuanceEnabled),
						EnableIdTokenIssuance:     pointer.To(model.ImplicitIdTokenIssuanceEnabled),
					},
				},
			}

			result, _, err := client.Create(ctx, properties)
			if err != nil {
				return fmt.Errorf("creating %s: %+v", parse.ApplicationId{}, err)
			}

			if pointer.From(result.ID()) == "" {
				return fmt.Errorf("creating %s: object ID returned for application is nil/empty", parse.ApplicationId{})
			}

			id := parse.NewApplicationID(*result.ID())
			metadata.SetID(id)

			return nil
		},
	}
}

func (r ApplicationRegistrationResource) Read() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 5 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			client := metadata.Client.Applications.ApplicationsClient
			client.BaseClient.DisableRetries = true
			defer func() { client.BaseClient.DisableRetries = false }()

			id, err := parse.ParseApplicationID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}

			result, status, err := client.Get(ctx, id.ApplicationId, odata.Query{})
			if err != nil {
				if status == http.StatusNotFound {
					return metadata.MarkAsGone(id)
				}
				return fmt.Errorf("retrieving %s: %+v", id, err)
			}

			if result == nil {
				return fmt.Errorf("retrieving %s: result was nil", id)
			}

			state := ApplicationRegistrationModel{
				ClientId:                   pointer.From(result.AppId),
				Description:                string(pointer.From(result.Description)),
				DisplayName:                pointer.From(result.DisplayName),
				GroupMembershipClaims:      pointer.From(result.GroupMembershipClaims),
				Notes:                      string(pointer.From(result.Notes)),
				ObjectId:                   pointer.From(result.ID()),
				PublisherDomain:            pointer.From(result.PublisherDomain),
				ServiceManagementReference: string(pointer.From(result.ServiceManagementReference)),
				SignInAudience:             pointer.From(result.SignInAudience),
			}

			if api := result.Api; api != nil {
				state.RequestedAccessTokenVersion = int(pointer.From(api.RequestedAccessTokenVersion))
			}

			if info := result.Info; info != nil {
				state.MarketingUrl = string(pointer.From(info.MarketingUrl))
				state.PrivacyStatementUrl = string(pointer.From(info.PrivacyStatementUrl))
				state.SupportUrl = string(pointer.From(info.SupportUrl))
				state.TermsOfServiceUrl = string(pointer.From(info.TermsOfServiceUrl))
			}

			if web := result.Web; web != nil {
				state.HomepageUrl = string(pointer.From(web.HomePageUrl))
				state.LogoutUrl = string(pointer.From(web.LogoutUrl))

				if implicitGrant := web.ImplicitGrantSettings; implicitGrant != nil {
					state.ImplicitAccessTokenIssuanceEnabled = pointer.From(implicitGrant.EnableAccessTokenIssuance)
					state.ImplicitIdTokenIssuanceEnabled = pointer.From(implicitGrant.EnableIdTokenIssuance)
				}
			}

			if result.DisabledByMicrosoftStatus != nil {
				state.DisabledByMicrosoft = fmt.Sprintf("%v", result.DisabledByMicrosoftStatus)
			}

			return metadata.Encode(&state)
		},
	}
}

func (r ApplicationRegistrationResource) Update() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 10 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			client := metadata.Client.Applications.ApplicationsClient
			rd := metadata.ResourceData

			id, err := parse.ParseApplicationID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}

			var model ApplicationRegistrationModel
			if err := metadata.Decode(&model); err != nil {
				return fmt.Errorf("decoding: %+v", err)
			}

			tf.LockByName(applicationResourceName, id.ApplicationId)
			defer tf.UnlockByName(applicationResourceName, id.ApplicationId)

			properties := msgraph.Application{
				DirectoryObject: msgraph.DirectoryObject{
					Id: &id.ApplicationId,
				},
			}

			if rd.HasChange("display_name") {
				properties.DisplayName = &model.DisplayName
			}

			if rd.HasChange("description") {
				properties.Description = tf.NullableString(model.Description)
			}

			if rd.HasChange("group_membership_claims") {
				properties.GroupMembershipClaims = pointer.To(model.GroupMembershipClaims)
			}

			if rd.HasChange("notes") {
				properties.Notes = tf.NullableString(model.Notes)
			}

			if rd.HasChange("requested_access_token_version") {
				properties.Api = &msgraph.ApplicationApi{
					RequestedAccessTokenVersion: pointer.To(int32(model.RequestedAccessTokenVersion)),
				}
			}

			if rd.HasChange("service_management_reference") {
				properties.ServiceManagementReference = tf.NullableString(model.ServiceManagementReference)
			}

			if rd.HasChange("sign_in_audience") {
				properties.SignInAudience = &model.SignInAudience
			}

			if rd.HasChange("marketing_url") || rd.HasChange("privacy_statement_url") || rd.HasChange("support_url") || rd.HasChange("terms_of_service_url") {
				properties.Info = &msgraph.InformationalUrl{}

				if rd.HasChange("marketing_url") {
					properties.Info.MarketingUrl = tf.NullableString(model.MarketingUrl)
				}

				if rd.HasChange("privacy_statement_url") {
					properties.Info.PrivacyStatementUrl = tf.NullableString(model.PrivacyStatementUrl)
				}

				if rd.HasChange("support_url") {
					properties.Info.SupportUrl = tf.NullableString(model.SupportUrl)
				}

				if rd.HasChange("terms_of_service_url") {
					properties.Info.TermsOfServiceUrl = tf.NullableString(model.TermsOfServiceUrl)
				}
			}

			if rd.HasChange("implicit_access_token_issuance_enabled") || rd.HasChange("homepage_url") || rd.HasChange("implicit_id_token_issuance_enabled") || rd.HasChange("logout_url") {
				properties.Web = &msgraph.ApplicationWeb{}

				if rd.HasChange("homepage_url") {
					properties.Web.HomePageUrl = tf.NullableString(model.HomepageUrl)
				}

				if rd.HasChange("logout_url") {
					properties.Web.LogoutUrl = tf.NullableString(model.LogoutUrl)
				}

				if rd.HasChange("implicit_access_token_issuance_enabled") || rd.HasChange("implicit_id_token_issuance_enabled") {
					properties.Web.ImplicitGrantSettings = &msgraph.ImplicitGrantSettings{}

					if rd.HasChange("implicit_access_token_issuance_enabled") {
						properties.Web.ImplicitGrantSettings.EnableAccessTokenIssuance = pointer.To(model.ImplicitAccessTokenIssuanceEnabled)
					}

					if rd.HasChange("implicit_id_token_issuance_enabled") {
						properties.Web.ImplicitGrantSettings.EnableIdTokenIssuance = pointer.To(model.ImplicitIdTokenIssuanceEnabled)
					}
				}
			}

			_, err = client.Update(ctx, properties)
			if err != nil {
				return fmt.Errorf("updating %s: %+v", id, err)
			}

			return nil
		},
	}
}

func (r ApplicationRegistrationResource) Delete() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 5 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			client := metadata.Client.Applications.ApplicationsClient
			client.BaseClient.DisableRetries = true
			defer func() { client.BaseClient.DisableRetries = false }()

			id, err := parse.ParseApplicationID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}

			if _, err := client.Delete(ctx, id.ApplicationId); err != nil {
				return fmt.Errorf("deleting %s: %+v", id, err)
			}

			// Wait for application object to be deleted
			if err = helpers.WaitForDeletion(ctx, func(ctx context.Context) (*bool, error) {
				defer func() { client.BaseClient.DisableRetries = false }()
				client.BaseClient.DisableRetries = true
				if _, status, err := client.Get(ctx, id.ApplicationId, odata.Query{}); err != nil {
					if status == http.StatusNotFound {
						return pointer.To(false), nil
					}
					return nil, err
				}
				return pointer.To(true), nil
			}); err != nil {
				return fmt.Errorf("waiting for deletion of %s: %q", id, err)
			}

			return nil
		},
	}
}
