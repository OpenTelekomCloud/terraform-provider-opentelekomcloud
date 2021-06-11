---
subcategory: "Software Repository for Container (SWR)"
---

# opentelekomcloud_swr_organization_v2


## Example Usage

```hcl
resource opentelekomcloud_swr_organization_v2 org_1 {
  name = "organization_1"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the organization (namespace) to be created.
  Enter `1` to `64` characters, starting with a lowercase letter and ending with a lowercase letter or digit.
  Only lowercase letters, digits, periods (`.`), underscores (`_`), and hyphens (`-`) are allowed.
  Periods, underscores, and hyphens cannot be placed next to each other.
  A maximum of two consecutive underscores are allowed.

## Attributes Reference

The following attributes are exported:

* `name` - See Argument Reference above.

* `organization_id` - Numeric ID of the organization.

* `creator_name` - Username of the organization creator.

* `auth` - User permission. The value can be `1`, `3`, or `7`. `7`: manage `3`: write `1`: read

## Import

Organizations can be imported using the `name`, e.g.

```sh
terraform import opentelekomcloud_swr_organization_v2.org_1 organization_1
```