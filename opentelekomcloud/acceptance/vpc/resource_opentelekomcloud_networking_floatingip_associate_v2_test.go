package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"

	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/layer3/floatingips"

	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/common"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/env"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"
)

func TestAccNetworkingV2FloatingIPAssociate_basic(t *testing.T) {
	var fip floatingips.FloatingIP

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { common.TestAccPreCheck(t) },
		ProviderFactories: common.TestAccProviderFactories,
		CheckDestroy:      testAccCheckNetworkingV2FloatingIPAssociateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkingV2FloatingIPAssociate_basic,
				Check: resource.ComposeTestCheckFunc(
					TestAccCheckNetworkingV2FloatingIPExists(
						"opentelekomcloud_networking_floatingip_associate_v2.fip_1", &fip),
					resource.TestCheckResourceAttrPtr(
						"opentelekomcloud_networking_floatingip_associate_v2.fip_1", "floating_ip", &fip.FloatingIP),
					resource.TestCheckResourceAttrPtr(
						"opentelekomcloud_networking_floatingip_associate_v2.fip_1", "port_id", &fip.PortID),
				),
			},
		},
	})
}

func testAccCheckNetworkingV2FloatingIPAssociateDestroy(s *terraform.State) error {
	config := common.TestAccProvider.Meta().(*cfg.Config)
	networkClient, err := config.NetworkingV2Client(env.OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("error creating OpenTelekomCloud floating IP: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opentelekomcloud_networking_floatingip_v2" {
			continue
		}

		fip, err := floatingips.Get(networkClient, rs.Primary.ID).Extract()
		if err != nil {
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				return nil
			}

			return fmt.Errorf("error retrieving floating IP: %s", err)
		}

		if fip.PortID != "" {
			return fmt.Errorf("floating IP is still associated")
		}
	}

	return nil
}

func testAccCheckNetworkingV2FloatingIPAssociateExists(n string, fip *floatingips.FloatingIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := common.TestAccProvider.Meta().(*cfg.Config)
		networkClient, err := config.NetworkingV2Client(env.OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("error creating OpenTelekomCloud networking client: %s", err)
		}

		found, err := floatingips.Get(networkClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("FloatingIP not found")
		}

		*fip = *found

		return nil
	}
}

const testAccNetworkingV2FloatingIPAssociate_basic = `
resource "opentelekomcloud_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "opentelekomcloud_networking_subnet_v2" "subnet_1" {
  name = "subnet_1"
  cidr = "192.168.199.0/24"
  ip_version = 4
  network_id = opentelekomcloud_networking_network_v2.network_1.id
}

resource "opentelekomcloud_networking_router_interface_v2" "router_interface_1" {
  router_id = opentelekomcloud_networking_router_v2.router_1.id
  subnet_id = opentelekomcloud_networking_subnet_v2.subnet_1.id
}

resource "opentelekomcloud_networking_router_v2" "router_1" {
  name = "router_1"
  admin_state_up = "true"
}

resource "opentelekomcloud_networking_port_v2" "port_1" {
  admin_state_up = "true"
  network_id = opentelekomcloud_networking_subnet_v2.subnet_1.network_id

  fixed_ip {
    subnet_id = opentelekomcloud_networking_subnet_v2.subnet_1.id
    ip_address = "192.168.199.20"
  }
}

resource "opentelekomcloud_networking_floatingip_v2" "fip_1" {
}

resource "opentelekomcloud_networking_floatingip_associate_v2" "fip_1" {
  floating_ip = opentelekomcloud_networking_floatingip_v2.fip_1.address
  port_id = opentelekomcloud_networking_port_v2.port_1.id
}
`
