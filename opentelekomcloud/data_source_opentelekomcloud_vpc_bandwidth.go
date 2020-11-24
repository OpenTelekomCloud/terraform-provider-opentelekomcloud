package opentelekomcloud

import (
	"fmt"
	"log"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v1/bandwidths"
)

func dataSourceBandWidth() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceBandWidthRead,

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"size": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validateIntegerInRange(5, 2000),
			},
			"enterprise_project_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"share_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"bandwidth_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"charge_mode": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceBandWidthRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	vpcClient, err := config.networkingV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("error creating OpenTelekomCloud vpc client: %s", err)
	}

	listOpts := bandwidths.ListOpts{
		ShareType: "WHOLE",
	}

	allBWs, err := bandwidths.List(vpcClient, listOpts).Extract()
	if err != nil {
		return fmt.Errorf("unable to list OpenTelekomCloud bandwidths: %s", err)
	}
	if len(allBWs) == 0 {
		return fmt.Errorf("no OpenTelekomCloud bandwidth was found")
	}

	// Filter bandwidths by "name"
	var bandList []bandwidths.BandWidth
	name := d.Get("name").(string)
	for _, band := range allBWs {
		if name == band.Name {
			bandList = append(bandList, band)
		}
	}
	if len(bandList) == 0 {
		return fmt.Errorf("no OpenTelekomCloud bandwidth was found by name: %s", name)
	}

	// Filter bandwidths by "size"
	result := bandList[0]
	if v, ok := d.GetOk("size"); ok {
		var found bool
		for _, band := range bandList {
			if v.(int) == band.Size {
				found = true
				result = band
				break
			}
		}
		if !found {
			return fmt.Errorf("no OpenTelekomCloud bandwidth was found by size: %d", v.(int))
		}
	}

	log.Printf("[DEBUG] Retrieved OpenTelekomCloud bandwidth %s: %+v", result.ID, result)
	d.SetId(result.ID)
	mErr := multierror.Append(nil,
		d.Set("name", result.Name),
		d.Set("size", result.Size),
		d.Set("share_type", result.ShareType),
		d.Set("bandwidth_type", result.BandwidthType),
		d.Set("charge_mode", result.ChargeMode),
		d.Set("status", result.Status),
	)
	return mErr.ErrorOrNil()
}
