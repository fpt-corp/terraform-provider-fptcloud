# Terraform Provider Fptcloud

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21

## Testing
```sh
export FPTCLOUD_API_URL=local_api_url                                                                  2 ↵  11155  17:25:00
export FPTCLOUD_REGION=your_region
export FPTCLOUD_TENANT_NAME=your_tenant_anme
export FPTCLOUD_TOKEN=your_token
export TF_ACC=1
export VPC_ID=your_vpc_id

# Now run test command
make testacc TESTARGS='-run=test_name'
```