package fptcloud_image_test

import (
	"fmt"
	"strconv"
	"terraform-provider-fptcloud/commons/test-helper"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSourceImage_basic(t *testing.T) {
	datasourceName := "data.fptcloud_image.example"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { test_helper.TestPreCheck(t) },
		ProviderFactories: test_helper.TestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: DataSourceImageConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					DataSourceImageExist(datasourceName),
				),
			},
		},
	})
}

func TestAccDataSourceImage_withFilterByCatalog(t *testing.T) {
	datasourceName := "data.fptcloud_image.example_with_filter"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { test_helper.TestPreCheck(t) },
		ProviderFactories: test_helper.TestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: DataSourceImageWithFilterConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					DataSourceImageWithFilter(datasourceName),
				),
			},
		},
	})
}

func DataSourceImageExist(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no Record ID is set")
		}

		rawTotal := rs.Primary.Attributes["images.#"]
		total, err := strconv.Atoi(rawTotal)
		if err != nil {
			return err
		}

		if total < 1 {
			return fmt.Errorf("no fptcloud image retrieved")
		}

		return nil
	}
}

func DataSourceImageWithFilter(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no Record ID is set")
		}

		rawTotal := rs.Primary.Attributes["images.#"]
		total, err := strconv.Atoi(rawTotal)
		if err != nil {
			return err
		}

		if total < 1 {
			return fmt.Errorf("no fptcloud image retrieved")
		}

		invalid := false
		for i := 0; i < total; i++ {
			catalog := rs.Primary.Attributes[fmt.Sprintf("images.%d.catalog", i)]
			if catalog != "Ubuntu" {
				invalid = true
			}
		}

		if invalid {
			return fmt.Errorf("images invalid")
		}

		return nil
	}
}

// CONFIG TEST
func DataSourceImageConfig() string {
	return fmt.Sprintf(`
data "fptcloud_image" "example" {
  vpc_id = "%s"
}
`, test_helper.ENV["VPC_ID"])
}

func DataSourceImageWithFilterConfig() string {
	return fmt.Sprintf(`
data "fptcloud_image" "example_with_filter" {
  vpc_id = "%s"
  filter {
        key = "catalog"
        values = ["Ubuntu"]
  }
}
`, test_helper.ENV["VPC_ID"])
}
