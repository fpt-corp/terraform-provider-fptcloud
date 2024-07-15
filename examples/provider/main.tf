terraform {
  required_providers {
    fptcloud = {
      source = "github.com/terraform-providers/fptcloud"
    }
  }
}

provider "fptcloud" {
  region="your_region"
  token="your_token"
  tenant_name="your_tenant_name"
}