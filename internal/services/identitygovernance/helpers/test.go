// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"time"

	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-azuread/internal/acceptance"
)

func SleepCheck(d time.Duration) acceptance.TestCheckFunc {
	return func(s *terraform.State) error {
		time.Sleep(d)
		return nil
	}
}
