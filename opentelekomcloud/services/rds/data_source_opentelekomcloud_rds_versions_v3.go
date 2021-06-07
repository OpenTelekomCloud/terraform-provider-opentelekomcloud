package rds

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v3/datastores"

	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"
)

func DataSourceRdsVersionsV3() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceRdsVersionsV3Read,
		Schema: map[string]*schema.Schema{
			"database_name": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					"MySQL", "PostgreSQL", "SQLServer",
				}, true),
			},

			"versions": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
		},
	}
}

func dataSourceRdsVersionsV3Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	client, err := config.RdsV3Client(config.GetRegion(d))
	if err != nil {
		return diag.Errorf("error creating RDSv3 client: %s", err)
	}
	name := d.Get("database_name").(string)
	stores, err := getRdsV3VersionList(client, name)

	if err := d.Set("versions", stores); err != nil {
		return diag.Errorf("error setting version list: %s", err)
	}
	d.SetId(fmt.Sprintf("%s_versions", name))

	return nil
}

func getRdsV3VersionList(client *golangsdk.ServiceClient, dbName string) ([]string, error) {
	pages, err := datastores.List(client, dbName).AllPages()
	if err != nil {
		return nil, fmt.Errorf("error listing RDSv3 versions: %s", err)
	}
	stores, err := datastores.ExtractDataStores(pages)
	if err != nil {
		return nil, fmt.Errorf("error extracting RDSv3 versions: %s", err)
	}
	result := make([]string, len(stores.DataStores))
	for i, store := range stores.DataStores {
		result[i] = store.Name
	}
	resultSorted := common.SortVersions(result)
	return resultSorted, nil
}
