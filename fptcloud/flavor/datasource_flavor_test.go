package fptcloud_flavor_test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"strconv"
	"terraform-provider-fptcloud/commons/test-helper"
)

//func TestAccDataSourceFlavor_basic(t *testing.T) {
//	datasourceName := "data.fptcloud_flavor.example"
//
//	resource.Test(t, resource.TestCase{
//		PreCheck:          func() { test_helper.TestPreCheck(t) },
//		ProviderFactories: test_helper.TestProviderFactories,
//		Steps: []resource.TestStep{
//			{
//				Config: DataSourceFlavorConfig(),
//				Check: resource.ComposeAggregateTestCheckFunc(
//					DataSourceFlavorExist(datasourceName),
//				),
//			},
//		},
//	})
//}

//func TestAccDataSourceFlavor_withFilterByName(t *testing.T) {
//	datasourceName := "data.fptcloud_flavor.example_with_filter"
//
//	resource.Test(t, resource.TestCase{
//		PreCheck:          func() { test_helper.TestPreCheck(t) },
//		ProviderFactories: test_helper.TestProviderFactories,
//		Steps: []resource.TestStep{
//			{
//				Config: DataSourceFlavorWithFilterConfig(),
//				Check: resource.ComposeAggregateTestCheckFunc(
//					DataSourceFlavorWithFilter(datasourceName),
//				),
//			},
//		},
//	})
//}

func DataSourceFlavorExist(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no Record ID is set")
		}

		rawTotal := rs.Primary.Attributes["flavors.#"]
		total, err := strconv.Atoi(rawTotal)
		if err != nil {
			return err
		}

		if total < 1 {
			return fmt.Errorf("no fptcloud flavor retrieved")
		}

		return nil
	}
}

func DataSourceFlavorWithFilter(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no Record ID is set")
		}

		rawTotal := rs.Primary.Attributes["flavors.#"]
		total, err := strconv.Atoi(rawTotal)
		if err != nil {
			return err
		}

		if total < 1 {
			return fmt.Errorf("no fptcloud flavor retrieved")
		}

		foundFlavorName := rs.Primary.Attributes["flavors.0.name"]

		if foundFlavorName != "Small-1" {
			return fmt.Errorf("flavor name not match")
		}

		return nil
	}
}

// CONFIG TEST
func DataSourceFlavorConfig() string {
	return fmt.Sprintf(`
data "fptcloud_flavor" "example" {
  vpc_id = "%s"
}
`, test_helper.ENV["VPC_ID"])
}

func DataSourceFlavorWithFilterConfig() string {
	return fmt.Sprintf(`
data "fptcloud_flavor" "example_with_filter" {
  vpc_id = "%s"
  filter {
        key = "name"
        values = ["Small-1"]
  }
}
`, test_helper.ENV["VPC_ID"])
}
