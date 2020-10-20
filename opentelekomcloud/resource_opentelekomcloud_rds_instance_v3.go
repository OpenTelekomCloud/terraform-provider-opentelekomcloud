package opentelekomcloud

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v1/subnets"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/ports"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v1/datastores"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v1/flavors"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v1/instances"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v1/tags"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v3/backups"
)

func resourceRdsInstanceV3() *schema.Resource {
	return &schema.Resource{
		Create: resourceRdsInstanceV3Create,
		Read:   resourceRdsInstanceV3Read,
		Update: resourceRdsInstanceV3Update,
		Delete: resourceRdsInstanceV3Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"availability_zone": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"db": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"password": {
							Type:      schema.TypeString,
							Sensitive: true,
							Required:  true,
							ForceNew:  true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"version": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"port": {
							Type:     schema.TypeInt,
							Computed: true,
							Optional: true,
							ForceNew: true,
						},
						"user_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"flavor": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"security_group_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"subnet_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"volume": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: false,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"size": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: false,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"disk_encryption_id": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"vpc_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"backup_strategy": {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				ForceNew: false,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"start_time": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},
						"keep_days": {
							Type:     schema.TypeInt,
							Computed: true,
							Optional: true,
							ForceNew: false,
						},
					},
				},
			},

			"ha_replication_mode": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},

			"tag": {
				Type:         schema.TypeMap,
				Optional:     true,
				ValidateFunc: validateECSTagValue,
			},

			"param_group_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"created": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"nodes": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"availability_zone": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"role": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"status": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"private_ips": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"public_ips": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateIP,
				},
			},
		},
	}
}

func resourceRdsInstanceV3UserInputParams(d *schema.ResourceData) map[string]interface{} {
	return map[string]interface{}{
		"terraform_resource_data": d,
		"availability_zone":       d.Get("availability_zone"),
		"backup_strategy":         d.Get("backup_strategy"),
		"db":                      d.Get("db"),
		"flavor":                  d.Get("flavor"),
		"ha_replication_mode":     d.Get("ha_replication_mode"),
		"name":                    d.Get("name"),
		"param_group_id":          d.Get("param_group_id"),
		"security_group_id":       d.Get("security_group_id"),
		"subnet_id":               d.Get("subnet_id"),
		"volume":                  d.Get("volume"),
		"vpc_id":                  d.Get("vpc_id"),
	}
}

func resourceRdsInstanceV3Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := config.rdsV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating sdk client, err=%s", err)
	}
	opts := resourceRdsInstanceV3UserInputParams(d)
	opts["region"] = GetRegion(d, config)

	arrayIndex := map[string]int{
		"backup_strategy": 0,
		"db":              0,
		"volume":          0,
	}
	publicIPs := d.Get("public_ips").([]interface{})

	params, err := buildRdsInstanceV3CreateParameters(opts, arrayIndex)
	if err != nil {
		return fmt.Errorf("Error building the request body of api(create), err=%s", err)
	}
	r, err := sendRdsInstanceV3CreateRequest(d, params, client)
	if err != nil {
		return fmt.Errorf("Error creating RdsInstanceV3, err=%s", err)
	}

	timeout := d.Timeout(schema.TimeoutCreate)
	obj, err := asyncWaitRdsInstanceV3Create(d, config, r, client, timeout)
	if err != nil {
		return err
	}
	id, err := navigateValue(obj, []string{"job", "instance", "id"}, nil)
	if err != nil {
		return fmt.Errorf("Error constructing id, err=%s", err)
	}
	d.SetId(id.(string))

	if hasFilledOpt(d, "tag") {
		var nodeID string
		res := make(map[string]interface{})
		v, err := fetchRdsInstanceV3ByList(d, client)
		if err != nil {
			return err
		}
		res["list"] = v
		err = setRdsInstanceV3Properties(d, res, config)
		if err != nil {
			return err
		}

		nodeID = getMasterID(d.Get("nodes").([]interface{}))
		if nodeID == "" {
			log.Printf("[WARN] Error setting tag(key/value) of instance:%s", id.(string))
			return nil
		}
		tagClient, err := config.rdsTagV1Client(GetRegion(d, config))
		if err != nil {
			return fmt.Errorf("Error creating OpenTelekomCloud rds tag client: %s ", err)
		}
		tagmap := d.Get("tag").(map[string]interface{})
		log.Printf("[DEBUG] Setting tag(key/value): %v", tagmap)
		for key, val := range tagmap {
			tagOpts := tags.CreateOpts{
				Key:   key,
				Value: val.(string),
			}
			err = tags.Create(tagClient, nodeID, tagOpts).ExtractErr()
			if err != nil {
				log.Printf("[WARN] Error setting tag(key/value) of instance:%s, err=%s", id.(string), err)
			}
		}
	}

	if len(publicIPs) > 0 {
		if err := resourceRdsInstanceV3Read(d, meta); err != nil {
			return err
		}
		nw, err := config.networkingV2Client(GetRegion(d, config))
		if err != nil {
			return err
		}
		subnetID, err := getSubnetSubnetID(d, config)
		if err != nil {
			return err
		}
		if err := assignEipToInstance(nw, publicIPs[0].(string), getPrivateIP(d), subnetID); err != nil {
			log.Printf("[WARN] failed to assign public IP: %s", err)
		}
	}

	return resourceRdsInstanceV3Read(d, meta)
}

func getPrivateIP(d *schema.ResourceData) string {
	return d.Get("private_ips").([]interface{})[0].(string)
}

func findFloatingIP(client *golangsdk.ServiceClient, address, portID string) (id string, err error) {
	var opts = floatingips.ListOpts{}
	if address != "" {
		opts.FloatingIP = address
	} else {
		opts.PortID = portID
	}
	pgFIP, err := floatingips.List(client, opts).AllPages()
	if err != nil {
		return
	}
	floatingIPs, err := floatingips.ExtractFloatingIPs(pgFIP)
	if err != nil {
		return
	}
	if len(floatingIPs) == 0 {
		return
	}

	for _, ip := range floatingIPs {
		if portID != "" && portID != ip.PortID {
			continue
		}
		if address != "" && address != ip.FloatingIP {
			continue
		}
		return floatingIPs[0].ID, nil
	}
	return
}

func findPort(client *golangsdk.ServiceClient, privateIP string, subnetID string) (id string, err error) {

	// find assigned port
	pg, err := ports.List(client, nil).AllPages()

	if err != nil {
		return
	}
	portList, err := ports.ExtractPorts(pg)
	if err != nil {
		return
	}

	for _, port := range portList {
		address := port.FixedIPs[0]
		if address.IPAddress == privateIP && address.SubnetID == subnetID {
			id = port.ID
			return
		}
	}
	return
}

func assignEipToInstance(client *golangsdk.ServiceClient, publicIP, privateIP, subnetID string) error {
	portID, err := findPort(client, privateIP, subnetID)
	if err != nil {
		return err
	}

	ipID, err := findFloatingIP(client, publicIP, "")
	if err != nil {
		return err
	}
	return floatingips.Update(client, ipID, floatingips.UpdateOpts{PortID: &portID}).Err
}

func getSubnetSubnetID(d *schema.ResourceData, config *Config) (id string, err error) {
	subnetClient, err := config.networkingV1Client(GetRegion(d, config))
	if err != nil {
		err = fmt.Errorf("[WARN] Failed to create VPC client")
		return
	}
	sn, err := subnets.Get(subnetClient, d.Get("subnet_id").(string)).Extract()
	if err != nil {
		return
	}
	id = sn.SubnetId
	return
}

func getAssignedEip(d *schema.ResourceData, config *Config) (ip string, err error) {
	nw, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		err = fmt.Errorf("[WARN] Failed to create network client")
		return
	}
	subnetID, err := getSubnetSubnetID(d, config)
	if err != nil {
		return
	}
	privateIP := getPrivateIP(d)
	if privateIP == "" {
		log.Print("[DEBUG] private IP is not yet assigned to RDS instance")
		return
	}
	portID, err := findPort(nw, privateIP, subnetID)
	if err != nil {
		return
	}

	id, err := findFloatingIP(nw, "", portID)
	if err != nil || id == "" {
		return
	}

	ipObj, err := floatingips.Get(nw, id).Extract()
	if err != nil {
		return
	}
	ip = ipObj.FloatingIP
	return
}

func unassignEipFromInstance(client *golangsdk.ServiceClient, oldPublicIP string) error {
	ipID, err := findFloatingIP(client, oldPublicIP, "")
	if err != nil {
		return err
	}
	return floatingips.Update(client, ipID, floatingips.UpdateOpts{PortID: nil}).Err
}

func resourceRdsInstanceV3Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	rdsClient, err := config.rdsV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud RDS Client: %s", err)
	}
	var updateOpts backups.UpdateOpts

	if d.HasChange("backup_strategy") {
		backupRaw := d.Get("backup_strategy").([]interface{})
		rawMap := backupRaw[0].(map[string]interface{})
		keep_days := rawMap["keep_days"].(int)
		updateOpts.KeepDays = &keep_days
		updateOpts.StartTime = rawMap["start_time"].(string)
		// TODO(zhenguo): Make Period configured by users
		updateOpts.Period = "1,2,3,4,5,6,7"
		log.Printf("[DEBUG] updateOpts: %#v", updateOpts)

		err = backups.Update(rdsClient, d.Id(), updateOpts).ExtractErr()
		if err != nil {
			return fmt.Errorf("Error updating OpenTelekomCloud RDS Instance: %s", err)
		}
	}

	// Fetching node id
	var nodeID string
	res := make(map[string]interface{})
	v, err := fetchRdsInstanceV3ByList(d, rdsClient)
	if err != nil {
		return err
	}
	res["list"] = v
	opts := resourceRdsInstanceV3UserInputParams(d)
	v, _ = opts["nodes"]
	v, err = flattenRdsInstanceV3Nodes(res, nil, v)
	if err != nil {
		return err
	}

	nodeID = getMasterID(v.([]interface{}))
	if nodeID == "" {
		log.Printf("[WARN] Error fetching node id of instance:%s", d.Id())
		return nil
	}

	if d.HasChange("tag") {
		oraw, nraw := d.GetChange("tag")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsRDS(o, n)
		tagClient, err := config.rdsTagV1Client(GetRegion(d, config))
		if err != nil {
			return fmt.Errorf("Error creating OpenTelekomCloud rds tag client: %s ", err)
		}

		if len(remove) > 0 {
			for _, opts := range remove {
				err = tags.Delete(tagClient, nodeID, opts).ExtractErr()
				if err != nil {
					log.Printf("[WARN] Error deleting tag(key/value) of instance:%s, err=%s", d.Id(), err)
				}
			}
		}
		if len(create) > 0 {
			for _, opts := range create {
				err = tags.Create(tagClient, nodeID, opts).ExtractErr()
				if err != nil {
					log.Printf("[WARN] Error setting tag(key/value) of instance:%s, err=%s", d.Id(), err)
				}
			}
		}
	}

	if d.HasChange("flavor") {
		_, nflavor := d.GetChange("flavor")
		client, err := config.rdsV1Client(GetRegion(d, config))
		if err != nil {
			return fmt.Errorf("Error creating OpenTelekomCloud rds v1 client: %s ", err)
		}

		// Fetch flavor id
		db := d.Get("db").([]interface{})
		datastoreName := db[0].(map[string]interface{})["type"].(string)
		datastoreVersion := db[0].(map[string]interface{})["version"].(string)
		datastoreList, err := datastores.List(client, datastoreName).Extract()
		if err != nil {
			return fmt.Errorf("Unable to retrieve datastores: %s ", err)
		}
		if len(datastoreList) < 1 {
			return fmt.Errorf("Returned no datastore result. ")
		}
		var datastoreId string
		for _, datastore := range datastoreList {
			if strings.HasPrefix(datastore.Name, datastoreVersion) {
				datastoreId = datastore.ID
				break
			}
		}
		if datastoreId == "" {
			return fmt.Errorf("Returned no datastore ID. ")
		}
		log.Printf("[DEBUG] Received datastore Id: %s", datastoreId)
		flavorsList, err := flavors.List(client, datastoreId, GetRegion(d, config)).Extract()
		if err != nil {
			return fmt.Errorf("Unable to retrieve flavors: %s", err)
		}
		if len(flavorsList) < 1 {
			return fmt.Errorf("Returned no flavor result. ")
		}
		var rdsFlavor flavors.Flavor
		for _, flavor := range flavorsList {
			if flavor.SpecCode == nflavor.(string) {
				rdsFlavor = flavor
				break
			}
		}

		var updateFlavorOpts instances.UpdateFlavorOps

		log.Printf("[DEBUG] Update flavor: %s", nflavor.(string))

		updateFlavorOpts.FlavorRef = rdsFlavor.ID
		_, err = instances.UpdateFlavorRef(client, updateFlavorOpts, nodeID).Extract()
		if err != nil {
			return fmt.Errorf("Error updating instance Flavor from result: %s ", err)
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"MODIFYING"},
			Target:     []string{"ACTIVE"},
			Refresh:    instanceStateFlavorUpdateRefreshFunc(client, nodeID, d.Get("flavor").(string)),
			Timeout:    d.Timeout(schema.TimeoutCreate),
			Delay:      15 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf(
				"Error waiting for instance (%s) flavor to be Updated: %s ",
				nodeID, err)
		}
		log.Printf("[DEBUG] Successfully updated instance %s flavor: %s", nodeID, d.Get("flavor").(string))
	}

	// Update volume
	if d.HasChange("volume") {
		client, err := config.rdsV1Client(GetRegion(d, config))
		if err != nil {
			return fmt.Errorf("Error creating OpenTelekomCloud rds v1 client: %s ", err)
		}
		_, nvolume := d.GetChange("volume")
		var updateOpts instances.UpdateOps
		volume := make(map[string]interface{})
		volumeRaw := nvolume.([]interface{})
		log.Printf("[DEBUG] volumeRaw: %+v", volumeRaw)
		if len(volumeRaw) == 1 {
			if m, ok := volumeRaw[0].(map[string]interface{}); ok {
				volume["size"] = m["size"].(int)
			}
		}
		log.Printf("[DEBUG] volume: %+v", volume)
		updateOpts.Volume = volume
		_, err = instances.UpdateVolumeSize(client, updateOpts, nodeID).Extract()
		if err != nil {
			return fmt.Errorf("Error updating instance volume from result: %s ", err)
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"MODIFYING"},
			Target:     []string{"UPDATED"},
			Refresh:    instanceStateUpdateRefreshFunc(client, nodeID, updateOpts.Volume["size"].(int)),
			Timeout:    d.Timeout(schema.TimeoutCreate),
			Delay:      15 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf(
				"Error waiting for instance (%s) volume to be Updated: %s ",
				nodeID, err)
		}
		log.Printf("[DEBUG] Successfully updated instance %s volume: %+v", nodeID, volume)
	}

	if d.HasChange("public_ips") {
		nw, err := config.networkingV2Client(GetRegion(d, config))
		olds, news := d.GetChange("public_ips")
		oldIPs := olds.([]interface{})
		newIPs := news.([]interface{})
		switch len(newIPs) {
		case 0:
			err = unassignEipFromInstance(nw, oldIPs[0].(string)) // if it become 0, it was 1 before
			break
		case 1:
			if len(oldIPs) > 0 {
				err = unassignEipFromInstance(nw, oldIPs[0].(string))
				if err != nil {
					return err
				}
			}
			privateIP := getPrivateIP(d)
			subnetID, err := getSubnetSubnetID(d, config)
			if err != nil {
				return err
			}
			err = assignEipToInstance(nw, newIPs[0].(string), privateIP, subnetID)
			break
		default:
			return fmt.Errorf("RDS instance can't have more than one public IP")
		}
	}

	return resourceRdsInstanceV3Read(d, meta)
}

func getMasterID(nodes []interface{}) (nodeID string) {
	for _, node := range nodes {
		nodeObj := node.(map[string]interface{})
		if nodeObj["role"].(string) == "master" {
			nodeID = nodeObj["id"].(string)
		}
	}
	return
}

func resourceRdsInstanceV3Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := config.rdsV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating sdk client, err=%s", err)
	}

	res := make(map[string]interface{})

	v, err := fetchRdsInstanceV3ByList(d, client)
	if err != nil {
		// manually bugfix for #476
		if strings.Index(err.Error(), "Error finding the resource by list api") != -1 {
			log.Printf("[WARN] the rds instance %s can not be found", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	res["list"] = v

	err = setRdsInstanceV3Properties(d, res, config)
	if err != nil {
		return err
	}

	// set instance tag
	var nodeID string
	nodes := d.Get("nodes").([]interface{})
	nodeID = getMasterID(nodes)
	if nodeID == "" {
		log.Printf("[WARN] Error fetching node id of instance:%s", d.Id())
		return nil
	}
	tagClient, err := config.rdsTagV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud rds tag client: %#v", err)
	}
	taglist, err := tags.Get(tagClient, nodeID).Extract()
	if err != nil {
		return fmt.Errorf("Error fetching OpenTelekomCloud rds instance tags: %s", err)
	}
	tagmap := make(map[string]string)
	for _, val := range taglist.Tags {
		tagmap[val.Key] = val.Value
	}
	if err := d.Set("tag", tagmap); err != nil {
		return fmt.Errorf("[DEBUG] Error saving tag to state for OpenTelekomCloud rds instance (%s): %s", d.Id(), err)
	}

	return nil
}

func resourceRdsInstanceV3Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := config.rdsV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating sdk client, err=%s", err)
	}

	url, err := replaceVars(d, "instances/{id}", nil)
	if err != nil {
		return err
	}
	url = client.ServiceURL(url)

	log.Printf("[DEBUG] Deleting Instance %q", d.Id())
	r := golangsdk.Result{}
	_, r.Err = client.Delete(url, &golangsdk.RequestOpts{
		OkCodes:      successHTTPCodes,
		JSONBody:     nil,
		JSONResponse: &r.Body,
		MoreHeaders: map[string]string{
			"Content-Type": "application/json",
			"X-Language":   "en-us",
		},
	})
	if r.Err != nil {
		return fmt.Errorf("Error deleting Instance %q, err=%s", d.Id(), r.Err)
	}

	_, err = waitToFinish(
		[]string{"Done"}, []string{"Pending"},
		d.Timeout(schema.TimeoutCreate),
		1*time.Second,
		func() (interface{}, string, error) {
			_, err := fetchRdsInstanceV3ByList(d, client)
			if err != nil {
				if strings.Index(err.Error(), "Error finding the resource by list api") != -1 {
					return true, "Done", nil
				}
				return nil, "", nil
			}
			return true, "Pending", nil
		},
	)
	return err
}

func buildRdsInstanceV3CreateParameters(opts map[string]interface{}, arrayIndex map[string]int) (interface{}, error) {
	params := make(map[string]interface{})

	v, err := expandRdsInstanceV3CreateAvailabilityZone(opts, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		params["availability_zone"] = v
	}

	v, err = expandRdsInstanceV3CreateBackupStrategy(opts, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		params["backup_strategy"] = v
	}

	v, err = navigateValue(opts, []string{"param_group_id"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		params["configuration_id"] = v
	}

	v, err = expandRdsInstanceV3CreateDatastore(opts, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		params["datastore"] = v
	}

	v, err = navigateValue(opts, []string{"volume", "disk_encryption_id"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		params["disk_encryption_id"] = v
	}

	v, err = navigateValue(opts, []string{"flavor"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		params["flavor_ref"] = v
	}

	v, err = expandRdsInstanceV3CreateHa(opts, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		params["ha"] = v
	}

	v, err = navigateValue(opts, []string{"name"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		params["name"] = v
	}

	v, err = navigateValue(opts, []string{"db", "password"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		params["password"] = v
	}

	v, err = navigateValue(opts, []string{"db", "port"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		params["port"] = v
	}

	v, err = expandRdsInstanceV3CreateRegion(opts, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		params["region"] = v
	}

	v, err = navigateValue(opts, []string{"security_group_id"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		params["security_group_id"] = v
	}

	v, err = navigateValue(opts, []string{"subnet_id"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		params["subnet_id"] = v
	}

	v, err = expandRdsInstanceV3CreateVolume(opts, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		params["volume"] = v
	}

	v, err = navigateValue(opts, []string{"vpc_id"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		params["vpc_id"] = v
	}

	return params, nil
}

func expandRdsInstanceV3CreateAvailabilityZone(d interface{}, arrayIndex map[string]int) (interface{}, error) {
	v, err := navigateValue(d, []string{"availability_zone"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	flavor, err := navigateValue(d, []string{"flavor"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if v1, ok := v.([]interface{}); ok {
		if strings.HasSuffix(flavor.(string), ".ha") {
			if len(v1) != 2 {
				return nil, fmt.Errorf("must input two available zones for primary/standby instance")
			}
			return v1[0].(string) + "," + v1[1].(string), nil
		}
		if len(v1) != 1 {
			return nil, fmt.Errorf("must input only one available zone for single instance")
		}
		return v1[0].(string), nil
	}
	return "", fmt.Errorf("can not convert to array")
}

func expandRdsInstanceV3CreateBackupStrategy(d interface{}, arrayIndex map[string]int) (interface{}, error) {
	req := make(map[string]interface{})

	v, err := navigateValue(d, []string{"backup_strategy", "keep_days"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		req["keep_days"] = v
	}

	v, err = navigateValue(d, []string{"backup_strategy", "start_time"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		req["start_time"] = v
	}

	return req, nil
}

func expandRdsInstanceV3CreateDatastore(d interface{}, arrayIndex map[string]int) (interface{}, error) {
	req := make(map[string]interface{})

	v, err := navigateValue(d, []string{"db", "type"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		req["type"] = v
	}

	v, err = navigateValue(d, []string{"db", "version"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		req["version"] = v
	}

	return req, nil
}

func expandRdsInstanceV3CreateHa(d interface{}, arrayIndex map[string]int) (interface{}, error) {
	req := make(map[string]interface{})

	v, err := expandRdsInstanceV3CreateHaMode(d, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		req["mode"] = v
	}

	v, err = navigateValue(d, []string{"ha_replication_mode"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		req["replication_mode"] = v
	}

	return req, nil
}

func expandRdsInstanceV3CreateHaMode(d interface{}, arrayIndex map[string]int) (interface{}, error) {
	v, err := navigateValue(d, []string{"ha_replication_mode"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if v1, ok := v.(string); ok && v1 != "" {
		return "ha", nil
	}
	return "", nil
}

func expandRdsInstanceV3CreateVolume(d interface{}, arrayIndex map[string]int) (interface{}, error) {
	req := make(map[string]interface{})

	v, err := navigateValue(d, []string{"volume", "size"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		req["size"] = v
	}

	v, err = navigateValue(d, []string{"volume", "type"}, arrayIndex)
	if err != nil {
		return nil, err
	}
	if e, err := isEmptyValue(reflect.ValueOf(v)); err != nil {
		return nil, err
	} else if !e {
		req["type"] = v
	}

	return req, nil
}

func sendRdsInstanceV3CreateRequest(d *schema.ResourceData, params interface{},
	client *golangsdk.ServiceClient) (interface{}, error) {
	url := client.ServiceURL("instances")

	r := golangsdk.Result{}
	_, r.Err = client.Post(url, params, &r.Body, &golangsdk.RequestOpts{
		OkCodes: successHTTPCodes,
		MoreHeaders: map[string]string{
			"X-Language": "en-us",
		},
	})
	if r.Err != nil {
		return nil, fmt.Errorf("Error running api(create), err=%s", r.Err)
	}
	return r.Body, nil
}

func asyncWaitRdsInstanceV3Create(d *schema.ResourceData, config *Config, result interface{},
	client *golangsdk.ServiceClient, timeout time.Duration) (interface{}, error) {

	data := make(map[string]string)
	pathParameters := map[string][]string{
		"id": {"job_id"},
	}
	for key, path := range pathParameters {
		value, err := navigateValue(result, path, nil)
		if err != nil {
			return nil, fmt.Errorf("Error retrieving async operation path parameter, err=%s", err)
		}
		data[key] = value.(string)
	}

	url, err := replaceVars(d, "jobs?id={id}", data)
	if err != nil {
		return nil, err
	}
	url = client.ServiceURL(url)

	return waitToFinish(
		[]string{"Completed"},
		[]string{"Running"},
		timeout, 1*time.Second,
		func() (interface{}, string, error) {
			r := golangsdk.Result{}
			_, r.Err = client.Get(url, &r.Body, &golangsdk.RequestOpts{
				MoreHeaders: map[string]string{
					"Content-Type": "application/json",
					"X-Language":   "en-us",
				}})
			if r.Err != nil {
				return nil, "", nil
			}

			status, err := navigateValue(r.Body, []string{"job", "status"}, nil)
			if err != nil {
				return nil, "", nil
			}
			return r.Body, status.(string), nil
		},
	)
}

func fetchRdsInstanceV3ByList(d *schema.ResourceData, client *golangsdk.ServiceClient) (interface{}, error) {
	identity := map[string]interface{}{"id": d.Id()}

	queryLink := "?id=" + identity["id"].(string)

	link := client.ServiceURL("instances") + queryLink

	return findRdsInstanceV3ByList(client, link, identity)
}

func findRdsInstanceV3ByList(client *golangsdk.ServiceClient, link string, identity map[string]interface{}) (interface{}, error) {
	r, err := sendRdsInstanceV3ListRequest(client, link)
	if err != nil {
		return nil, err
	}

	for _, item := range r.([]interface{}) {
		val := item.(map[string]interface{})

		bingo := true
		for k, v := range identity {
			if val[k] != v {
				bingo = false
				break
			}
		}
		if bingo {
			return item, nil
		}
	}

	return nil, fmt.Errorf("Error finding the resource by list api")
}

func sendRdsInstanceV3ListRequest(client *golangsdk.ServiceClient, url string) (interface{}, error) {
	r := golangsdk.Result{}
	_, r.Err = client.Get(url, &r.Body, &golangsdk.RequestOpts{
		MoreHeaders: map[string]string{
			"Content-Type": "application/json",
			"X-Language":   "en-us",
		}})
	if r.Err != nil {
		return nil, fmt.Errorf("Error running api(list) for resource(RdsInstanceV3), err=%s", r.Err)
	}

	v, err := navigateValue(r.Body, []string{"instances"}, nil)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func setRdsInstanceV3Properties(d *schema.ResourceData, response map[string]interface{}, config *Config) error {
	opts := resourceRdsInstanceV3UserInputParams(d)

	v, err := flattenRdsInstanceV3AvailabilityZone(response)
	if err != nil {
		return fmt.Errorf("Error reading Instance:availability_zone, err: %s", err)
	}
	if err = d.Set("availability_zone", v); err != nil {
		return fmt.Errorf("Error setting Instance:availability_zone, err: %s", err)
	}

	v, _ = opts["backup_strategy"]
	v, err = flattenRdsInstanceV3BackupStrategy(response, nil, v)
	if err != nil {
		return fmt.Errorf("Error reading Instance:backup_strategy, err: %s", err)
	}
	if err = d.Set("backup_strategy", v); err != nil {
		return fmt.Errorf("Error setting Instance:backup_strategy, err: %s", err)
	}

	v, err = navigateValue(response, []string{"list", "created"}, nil)
	if err != nil {
		return fmt.Errorf("Error reading Instance:created, err: %s", err)
	}
	if err = d.Set("created", v); err != nil {
		return fmt.Errorf("Error setting Instance:created, err: %s", err)
	}

	v, _ = opts["db"]
	v, err = flattenRdsInstanceV3Db(response, nil, v)
	if err != nil {
		return fmt.Errorf("Error reading Instance:db, err: %s", err)
	}
	if err = d.Set("db", v); err != nil {
		return fmt.Errorf("Error setting Instance:db, err: %s", err)
	}

	v, err = navigateValue(response, []string{"list", "flavor_ref"}, nil)
	if err != nil {
		return fmt.Errorf("Error reading Instance:flavor, err: %s", err)
	}
	if err = d.Set("flavor", v); err != nil {
		return fmt.Errorf("Error setting Instance:flavor, err: %s", err)
	}

	v, _ = opts["ha_replication_mode"]
	v, err = flattenRdsInstanceV3HAReplicationMode(response, nil, v)
	if err != nil {
		return fmt.Errorf("Error reading Instance:ha_replication_mode, err: %s", err)
	}
	if err = d.Set("ha_replication_mode", v); err != nil {
		return fmt.Errorf("Error setting Instance:ha_replication_mode, err: %s", err)
	}

	v, err = navigateValue(response, []string{"list", "name"}, nil)
	if err != nil {
		return fmt.Errorf("Error reading Instance:name, err: %s", err)
	}
	if err = d.Set("name", v); err != nil {
		return fmt.Errorf("Error setting Instance:name, err: %s", err)
	}

	v, _ = opts["nodes"]
	v, err = flattenRdsInstanceV3Nodes(response, nil, v)
	if err != nil {
		return fmt.Errorf("Error reading Instance:nodes, err: %s", err)
	}
	if err = d.Set("nodes", v); err != nil {
		return fmt.Errorf("Error setting Instance:nodes, err: %s", err)
	}

	v, err = navigateValue(response, []string{"list", "private_ips"}, nil)
	if err != nil {
		return fmt.Errorf("Error reading Instance:private_ips, err: %s", err)
	}
	if err = d.Set("private_ips", v); err != nil {
		return fmt.Errorf("Error setting Instance:private_ips, err: %s", err)
	}

	v, err = navigateValue(response, []string{"list", "public_ips"}, nil)
	if err != nil {
		return fmt.Errorf("Error reading Instance:public_ips, err: %s", err)
	}
	if err = d.Set("public_ips", v); err != nil {
		return fmt.Errorf("Error setting Instance:public_ips, err: %s", err)
	}
	if len(v.([]interface{})) == 0 {
		ip, err := getAssignedEip(d, config)
		if err != nil {
			return fmt.Errorf("Error setting Instance:public_ips, err: %s", err)
		}
		if ip != "" {
			_ = d.Set("public_ips", []string{ip})
		}
	}

	v, err = navigateValue(response, []string{"list", "security_group_id"}, nil)
	if err != nil {
		return fmt.Errorf("Error reading Instance:security_group_id, err: %s", err)
	}
	if err = d.Set("security_group_id", v); err != nil {
		return fmt.Errorf("Error setting Instance:security_group_id, err: %s", err)
	}

	v, err = navigateValue(response, []string{"list", "subnet_id"}, nil)
	if err != nil {
		return fmt.Errorf("Error reading Instance:subnet_id, err: %s", err)
	}
	if err = d.Set("subnet_id", v); err != nil {
		return fmt.Errorf("Error setting Instance:subnet_id, err: %s", err)
	}

	v, _ = opts["volume"]
	v, err = flattenRdsInstanceV3Volume(response, nil, v)
	if err != nil {
		return fmt.Errorf("Error reading Instance:volume, err: %s", err)
	}
	if err = d.Set("volume", v); err != nil {
		return fmt.Errorf("Error setting Instance:volume, err: %s", err)
	}

	v, err = navigateValue(response, []string{"list", "vpc_id"}, nil)
	if err != nil {
		return fmt.Errorf("Error reading Instance:vpc_id, err: %s", err)
	}
	if err = d.Set("vpc_id", v); err != nil {
		return fmt.Errorf("Error setting Instance:vpc_id, err: %s", err)
	}

	return nil
}

func flattenRdsInstanceV3AvailabilityZone(d interface{}) (interface{}, error) {
	arrayIndex := make(map[string]int)
	arrayIndex["list.nodes"] = 0
	v, err := navigateValue(d, []string{"list", "nodes", "availability_zone"}, arrayIndex)
	if err != nil {
		return nil, fmt.Errorf("Error reading Instance:availability_zone, err: %s", err)
	}
	az1 := v.(string)

	v, err = navigateValue(d, []string{"list", "flavor_ref"}, nil)
	if err != nil {
		return nil, fmt.Errorf("Error reading Instance:flavor, err: %s", err)
	}
	if strings.HasSuffix(v.(string), ".ha") {
		arrayIndex["list.nodes"] = 1
		v, err := navigateValue(d, []string{"list", "nodes", "availability_zone"}, arrayIndex)
		if err != nil {
			return nil, fmt.Errorf("Error reading Instance:availability_zone, err: %s", err)
		}
		az2 := v.(string)

		v, err = navigateValue(d, []string{"list", "nodes", "role"}, arrayIndex)
		if err != nil {
			return nil, fmt.Errorf("Error reading Instance:role, err: %s", err)
		}
		if v.(string) == "master" {
			return []string{az2, az1}, nil
		} else {
			return []string{az1, az2}, nil
		}
	}

	return []string{az1}, nil
}

func flattenRdsInstanceV3BackupStrategy(d interface{}, arrayIndex map[string]int, currentValue interface{}) (interface{}, error) {
	result, ok := currentValue.([]interface{})
	if !ok || len(result) == 0 {
		result = make([]interface{}, 1, 1)
	}
	if result[0] == nil {
		result[0] = make(map[string]interface{})
	}
	r := result[0].(map[string]interface{})

	v, err := navigateValue(d, []string{"list", "backup_strategy", "keep_days"}, arrayIndex)
	if err != nil {
		return nil, fmt.Errorf("Error reading Instance:keep_days, err: %s", err)
	}
	r["keep_days"] = v

	v, err = navigateValue(d, []string{"list", "backup_strategy", "start_time"}, arrayIndex)
	if err != nil {
		return nil, fmt.Errorf("Error reading Instance:start_time, err: %s", err)
	}
	r["start_time"] = v

	return result, nil
}

func flattenRdsInstanceV3Db(d interface{}, arrayIndex map[string]int, currentValue interface{}) (interface{}, error) {
	result, ok := currentValue.([]interface{})
	if !ok || len(result) == 0 {
		result = make([]interface{}, 1, 1)
	}
	if result[0] == nil {
		result[0] = make(map[string]interface{})
	}
	r := result[0].(map[string]interface{})

	v, err := navigateValue(d, []string{"list", "port"}, arrayIndex)
	if err != nil {
		return nil, fmt.Errorf("Error reading Instance:port, err: %s", err)
	}
	r["port"] = v

	v, err = navigateValue(d, []string{"list", "datastore", "type"}, arrayIndex)
	if err != nil {
		return nil, fmt.Errorf("Error reading Instance:type, err: %s", err)
	}
	r["type"] = v

	v, err = navigateValue(d, []string{"list", "db_user_name"}, arrayIndex)
	if err != nil {
		return nil, fmt.Errorf("Error reading Instance:user_name, err: %s", err)
	}
	r["user_name"] = v

	v, err = navigateValue(d, []string{"list", "datastore", "version"}, arrayIndex)
	if err != nil {
		return nil, fmt.Errorf("Error reading Instance:version, err: %s", err)
	}
	r["version"] = v

	return result, nil
}

func flattenRdsInstanceV3Nodes(d interface{}, arrayIndex map[string]int, currentValue interface{}) (interface{}, error) {
	result, ok := currentValue.([]interface{})
	if !ok || len(result) == 0 {
		v, err := navigateValue(d, []string{"list", "nodes"}, arrayIndex)
		if err != nil {
			return nil, err
		}
		n := len(v.([]interface{}))
		result = make([]interface{}, n, n)
	}

	newArrayIndex := make(map[string]int)
	if arrayIndex != nil {
		for k, v := range arrayIndex {
			newArrayIndex[k] = v
		}
	}

	for i := 0; i < len(result); i++ {
		newArrayIndex["list.nodes"] = i
		if result[i] == nil {
			result[i] = make(map[string]interface{})
		}
		r := result[i].(map[string]interface{})

		v, err := navigateValue(d, []string{"list", "nodes", "availability_zone"}, newArrayIndex)
		if err != nil {
			return nil, fmt.Errorf("Error reading Instance:availability_zone, err: %s", err)
		}
		r["availability_zone"] = v

		v, err = navigateValue(d, []string{"list", "nodes", "id"}, newArrayIndex)
		if err != nil {
			return nil, fmt.Errorf("Error reading Instance:id, err: %s", err)
		}
		r["id"] = v

		v, err = navigateValue(d, []string{"list", "nodes", "name"}, newArrayIndex)
		if err != nil {
			return nil, fmt.Errorf("Error reading Instance:name, err: %s", err)
		}
		r["name"] = v

		v, err = navigateValue(d, []string{"list", "nodes", "role"}, newArrayIndex)
		if err != nil {
			return nil, fmt.Errorf("Error reading Instance:role, err: %s", err)
		}
		r["role"] = v

		v, err = navigateValue(d, []string{"list", "nodes", "status"}, newArrayIndex)
		if err != nil {
			return nil, fmt.Errorf("Error reading Instance:status, err: %s", err)
		}
		r["status"] = v
	}

	return result, nil
}

func flattenRdsInstanceV3Volume(d interface{}, arrayIndex map[string]int, currentValue interface{}) (interface{}, error) {
	result, ok := currentValue.([]interface{})
	if !ok || len(result) == 0 {
		result = make([]interface{}, 1, 1)
	}
	if result[0] == nil {
		result[0] = make(map[string]interface{})
	}
	r := result[0].(map[string]interface{})

	v, err := navigateValue(d, []string{"list", "disk_encryption_id"}, arrayIndex)
	if err != nil {
		return nil, fmt.Errorf("Error reading Instance:disk_encryption_id, err: %s", err)
	}
	r["disk_encryption_id"] = v

	v, err = navigateValue(d, []string{"list", "volume", "size"}, arrayIndex)
	if err != nil {
		return nil, fmt.Errorf("Error reading Instance:size, err: %s", err)
	}
	r["size"] = v

	v, err = navigateValue(d, []string{"list", "volume", "type"}, arrayIndex)
	if err != nil {
		return nil, fmt.Errorf("Error reading Instance:type, err: %s", err)
	}
	r["type"] = v

	return result, nil
}
