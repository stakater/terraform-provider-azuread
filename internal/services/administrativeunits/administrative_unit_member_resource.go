// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package administrativeunits

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-sdk/sdk/odata"
	"github.com/hashicorp/terraform-provider-azuread/internal/clients"
	"github.com/hashicorp/terraform-provider-azuread/internal/helpers"
	"github.com/hashicorp/terraform-provider-azuread/internal/services/administrativeunits/parse"
	"github.com/hashicorp/terraform-provider-azuread/internal/tf"
	"github.com/hashicorp/terraform-provider-azuread/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azuread/internal/tf/validation"
	"github.com/manicminer/hamilton/msgraph"
)

func administrativeUnitMemberResource() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		CreateContext: administrativeUnitMemberResourceCreate,
		ReadContext:   administrativeUnitMemberResourceRead,
		DeleteContext: administrativeUnitMemberResourceDelete,

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(5 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(5 * time.Minute),
		},

		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := parse.AdministrativeUnitMemberID(id)
			return err
		}),

		Schema: map[string]*pluginsdk.Schema{
			"administrative_unit_object_id": {
				Description:      "The object ID of the administrative unit",
				Type:             pluginsdk.TypeString,
				Optional:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ValidateDiag(validation.IsUUID),
			},

			"member_object_id": {
				Description:      "The object ID of the member",
				Type:             pluginsdk.TypeString,
				Optional:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ValidateDiag(validation.IsUUID),
			},
		},
	}
}

func administrativeUnitMemberResourceCreate(ctx context.Context, d *pluginsdk.ResourceData, meta interface{}) pluginsdk.Diagnostics {
	client := meta.(*clients.Client).AdministrativeUnits.AdministrativeUnitsClient
	directoryObjectsClient := meta.(*clients.Client).AdministrativeUnits.DirectoryObjectsClient
	tenantId := meta.(*clients.Client).TenantID

	id := parse.NewAdministrativeUnitMemberID(d.Get("administrative_unit_object_id").(string), d.Get("member_object_id").(string))

	tf.LockByName(administrativeUnitResourceName, id.AdministrativeUnitId)
	defer tf.UnlockByName(administrativeUnitResourceName, id.AdministrativeUnitId)

	administrativeUnit, status, err := client.Get(ctx, id.AdministrativeUnitId, odata.Query{})
	if err != nil {
		if status == http.StatusNotFound {
			return tf.ErrorDiagPathF(nil, "object_id", "Administrative unit with object ID %q was not found", id.AdministrativeUnitId)
		}
		return tf.ErrorDiagPathF(err, "object_id", "Retrieving administrative unit with object ID: %q", id.AdministrativeUnitId)
	}

	client.BaseClient.DisableRetries = true
	if _, status, err = client.GetMember(ctx, id.AdministrativeUnitId, id.MemberId); err == nil {
		return tf.ImportAsExistsDiag("azuread_administrative_unit_member", id.String())
	} else if status != http.StatusNotFound {
		return tf.ErrorDiagF(err, "Checking for existing membership of member %q for administrative unit with object ID: %q", id.MemberId, id.AdministrativeUnitId)
	}
	client.BaseClient.DisableRetries = false

	memberObject, _, err := directoryObjectsClient.Get(ctx, id.MemberId, odata.Query{})
	if err != nil {
		return tf.ErrorDiagF(err, "Could not retrieve member principal object %q", id.MemberId)
	}
	if memberObject == nil {
		return tf.ErrorDiagF(errors.New("returned memberObject was nil"), "Could not retrieve member principal object %q", id.MemberId)
	}
	memberObject.ODataId = (*odata.Id)(pointer.To(fmt.Sprintf("%s/v1.0/%s/directoryObjects/%s",
		client.BaseClient.Endpoint, tenantId, id.MemberId)))

	members := &msgraph.Members{*memberObject}

	if _, err := client.AddMembers(ctx, *administrativeUnit.ID, members); err != nil {
		return tf.ErrorDiagF(err, "Adding member %q to administrative unit %q", id.MemberId, id.AdministrativeUnitId)
	}

	// Wait for membership to reflect
	deadline, ok := ctx.Deadline()
	if !ok {
		return tf.ErrorDiagF(errors.New("context has no deadline"), "Waiting for member %q to reflect for administrative unit %q", id.MemberId, id.AdministrativeUnitId)
	}
	timeout := time.Until(deadline)
	_, err = (&pluginsdk.StateChangeConf{ //nolint:staticcheck
		Pending:                   []string{"Waiting"},
		Target:                    []string{"Done"},
		Timeout:                   timeout,
		MinTimeout:                1 * time.Second,
		ContinuousTargetOccurence: 3,
		Refresh: func() (interface{}, string, error) {
			_, status, err := client.GetMember(ctx, id.AdministrativeUnitId, id.MemberId)
			if err != nil {
				if status == http.StatusNotFound {
					return "stub", "Waiting", nil
				}
				return nil, "Error", fmt.Errorf("retrieving member")
			}
			return "stub", "Done", nil
		},
	}).WaitForStateContext(ctx)
	if err != nil {
		return tf.ErrorDiagF(err, "Waiting for member %q to reflect for administrative unit %q", id.MemberId, id.AdministrativeUnitId)
	}

	d.SetId(id.String())

	return administrativeUnitMemberResourceRead(ctx, d, meta)
}

func administrativeUnitMemberResourceRead(ctx context.Context, d *pluginsdk.ResourceData, meta interface{}) pluginsdk.Diagnostics {
	client := meta.(*clients.Client).AdministrativeUnits.AdministrativeUnitsClient

	id, err := parse.AdministrativeUnitMemberID(d.Id())
	if err != nil {
		return tf.ErrorDiagPathF(err, "id", "Parsing Administrative Unit Member ID %q", d.Id())
	}

	if _, status, err := client.GetMember(ctx, id.AdministrativeUnitId, id.MemberId); err != nil {
		if status == http.StatusNotFound {
			log.Printf("[DEBUG] Member with ID %q was not found in administrative unit %q - removing from state", id.MemberId, id.AdministrativeUnitId)
			d.SetId("")
			return nil
		}
		return tf.ErrorDiagF(err, "Retrieving member %q for administrative unit with object ID: %q", id.MemberId, id.AdministrativeUnitId)
	}

	tf.Set(d, "administrative_unit_object_id", id.AdministrativeUnitId)
	tf.Set(d, "member_object_id", id.MemberId)

	return nil
}

func administrativeUnitMemberResourceDelete(ctx context.Context, d *pluginsdk.ResourceData, meta interface{}) pluginsdk.Diagnostics {
	client := meta.(*clients.Client).AdministrativeUnits.AdministrativeUnitsClient

	id, err := parse.AdministrativeUnitMemberID(d.Id())
	if err != nil {
		return tf.ErrorDiagPathF(err, "id", "Parsing Administrative Unit Member ID %q", d.Id())
	}

	tf.LockByName(administrativeUnitResourceName, id.AdministrativeUnitId)
	defer tf.UnlockByName(administrativeUnitResourceName, id.AdministrativeUnitId)

	if _, err := client.RemoveMembers(ctx, id.AdministrativeUnitId, &[]string{id.MemberId}); err != nil {
		return tf.ErrorDiagF(err, "Removing member %q from administrative unit with object ID: %q", id.MemberId, id.AdministrativeUnitId)
	}

	// Wait for membership link to be deleted
	if err := helpers.WaitForDeletion(ctx, func(ctx context.Context) (*bool, error) {
		defer func() { client.BaseClient.DisableRetries = false }()
		client.BaseClient.DisableRetries = true
		if _, status, err := client.GetMember(ctx, id.AdministrativeUnitId, id.MemberId); err != nil {
			if status == http.StatusNotFound {
				return pointer.To(false), nil
			}
			return nil, err
		}
		return pointer.To(true), nil
	}); err != nil {
		return tf.ErrorDiagF(err, "Waiting for removal of member %q from administrative unit with object ID %q", id.MemberId, id.AdministrativeUnitId)
	}

	return nil
}
