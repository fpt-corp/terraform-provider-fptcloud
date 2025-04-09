terraform {
  required_providers {
    fptcloud = {
      source = "github.com/terraform-providers/fptcloud"
    }
  }
}

provider "fptcloud" {
  region      = "your_region"
  token       = "your_token"
  tenant_name = "your_tenant_name"
}

# Configure the provider for japan region
provider "fptcloud" {
  region       = "JP/JCSI2"
  token        = "your_token"
  tenant_name  = "your_tenant_name"
  api_endpoint = "https://console-api.fptcloud.jp/api"
}

# Configure the provider with custom timeout
provider "fptcloud" {
  region      = "your_region"
  token       = "your_token"
  tenant_name = "your_tenant_name"
  timeout     = 10
}