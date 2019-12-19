package azurerm

import (
	"fmt"

	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/acceptance"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
)

func TestAccAzureRMSiteRecoveryReplicationPolicy_basic(t *testing.T) {
	resourceGroupName := "azurerm_resource_group.test"
	vaultName := "azurerm_recovery_services_vault.test"
	resourceName := "azurerm_site_recovery_replication_policy.test"
	ri := tf.AccRandTimeInt()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMResourceGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMSiteRecoveryReplicationPolicy_basic(ri, acceptance.Location()),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSiteRecoveryReplicationPolicyExists(resourceGroupName, vaultName, resourceName),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAzureRMSiteRecoveryReplicationPolicy_basic(rInt int, location string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_recovery_services_vault" "test" {
  name                = "acctest-vault-%d"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  sku                 = "Standard"
}

resource "azurerm_site_recovery_replication_policy" "test" {
  resource_group_name                                  = "${azurerm_resource_group.test.name}"
  recovery_vault_name                                  = "${azurerm_recovery_services_vault.test.name}"
  name                                                 = "acctest-policy-%d"
  recovery_point_retention_in_minutes                  = "${24 * 60}"
  application_consistent_snapshot_frequency_in_minutes = "${4 * 60}"
}
`, rInt, location, rInt, rInt)
}

func testCheckAzureRMSiteRecoveryReplicationPolicyExists(resourceGroupStateName, vaultStateName string, policyStateName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		resourceGroupState, ok := s.RootModule().Resources[resourceGroupStateName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceGroupStateName)
		}
		vaultState, ok := s.RootModule().Resources[vaultStateName]
		if !ok {
			return fmt.Errorf("Not found: %s", vaultStateName)
		}
		policyState, ok := s.RootModule().Resources[policyStateName]
		if !ok {
			return fmt.Errorf("Not found: %s", policyStateName)
		}

		resourceGroupName := resourceGroupState.Primary.Attributes["name"]
		vaultName := vaultState.Primary.Attributes["name"]
		policyName := policyState.Primary.Attributes["name"]

		// Ensure fabric exists in API
		client := acceptance.AzureProvider.Meta().(*clients.Client).RecoveryServices.ReplicationPoliciesClient(resourceGroupName, vaultName)
		ctx := acceptance.AzureProvider.Meta().(*clients.Client).StopContext

		resp, err := client.Get(ctx, policyName)
		if err != nil {
			return fmt.Errorf("Bad: Get on fabricClient: %+v", err)
		}

		if resp.Response.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: fabric: %q does not exist", policyName)
		}

		return nil
	}
}
