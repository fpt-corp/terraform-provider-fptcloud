---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "fptcloud_flavor Data Source - terraform-provider-fptcloud"
subcategory: ""
description: |-
  Retrieves information about the flavor that fpt cloud supports, with the ability to filter the results.
---

# fptcloud_flavor (Data Source)

Retrieves information about the flavor that fpt cloud supports, with the ability to filter the results.

## Example Usage

```terraform
data "fptcloud_flavor" "example" {
  vpc_id = "your_vpc_id"
  filter {
    key = "name"
    values = ["Extra-8"]
  }
}

data "fptcloud_flavor" "example_filter" {
  vpc_id = "your_vpc_id"
  filter {
    key = "cpu"
    values = [2]
  }
  filter {
    key = "memory_mb"
    values = [4 * 1024]
  }
  filter {
    key = "type"
    values = ["VM_SIZE"]
  }
}

output "show_value" {
  value = element(data.fptcloud_flavor.example.flavors, 0)
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `vpc_id` (String) The vpc id of the flavor

### Optional

- `filter` (Block Set) One or more key/value pairs on which to filter results (see [below for nested schema](#nestedblock--filter))
- `sort` (Block List) One or more key/direction pairs on which to sort results (see [below for nested schema](#nestedblock--sort))

### Read-Only

- `flavors` (List of Object) (see [below for nested schema](#nestedatt--flavors))
- `id` (String) The ID of this resource.

<a id="nestedblock--filter"></a>
### Nested Schema for `filter`

Required:

- `key` (String) Filter flavors by this key. This may be one of `cpu`, `gpu_memory_gb`, `id`, `memory_mb`, `name`, `type`.
- `values` (List of String) Only retrieves `flavors` which keys has value that matches one of the values provided here

Optional:

- `all` (Boolean) Set to `true` to require that a field match all of the `values` instead of just one or more of them. This is useful when matching against multi-valued fields such as lists or sets where you want to ensure that all of the `values` are present in the list or set.
- `match_by` (String) One of `exact` (default), `re`, or `substring`. For string-typed fields, specify `re` to match by using the `values` as regular expressions, or specify `substring` to match by treating the `values` as substrings to find within the string field.


<a id="nestedblock--sort"></a>
### Nested Schema for `sort`

Required:

- `key` (String) Sort flavors by this key. This may be one of `cpu`, `gpu_memory_gb`, `id`, `memory_mb`, `name`, `type`.

Optional:

- `direction` (String) The sort direction. This may be either `asc` or `desc`.


<a id="nestedatt--flavors"></a>
### Nested Schema for `flavors`

Read-Only:

- `cpu` (Number)
- `gpu_memory_gb` (Number)
- `id` (String)
- `memory_mb` (Number)
- `name` (String)
- `type` (String)