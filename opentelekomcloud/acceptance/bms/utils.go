package acceptance

import (
	"testing"

	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/common"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/env"
)

func testAccBmsFlavorPreCheck(t *testing.T) {
	common.TestAccPreCheckRequiredEnvVars(t)
	if env.OS_BMS_FLAVOR_NAME == "" {
		t.Skip("Provide the bms flavor name starting with 'physical'")
	}
}
