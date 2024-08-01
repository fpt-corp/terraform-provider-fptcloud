---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "fptcloud_vpc Data Source - terraform-provider-fptcloud"
subcategory: ""
description: |-
  
---

# fptcloud_vpc (Data Source)



## Example Usage

```terraform
data "fptcloud_vpc" "example" {
  name = "vpc-name"
}

output "name" {
  value = data.fptcloud_vpc.example
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `name` (String) The name of VPC
- `status` (String) The status of VPC

### Read-Only

- `id` (String) The ID of this resource.