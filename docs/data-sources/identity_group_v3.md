---
subcategory: "Identity and Access Management (IAM)"
---

# opentelekomcloud_identity_group_v3
Use this data source to get the ID of an OpenTelekomCloud group.

-> **Note:** This usually requires admin privileges.

## Example Usage

```hcl
data "opentelekomcloud_identity_group_v3" "admins" {
  name = "admins"
}
```

## Argument Reference

* `name` - (Required) The name of the group.

* `domain_id` - (Optional) The domain the group belongs to.


## Attributes Reference

`id` is set to the ID of the found group. In addition, the following attributes are exported:

* `name` - See Argument Reference above.

* `domain_id` - See Argument Reference above.
