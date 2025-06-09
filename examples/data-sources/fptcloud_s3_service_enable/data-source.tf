data "fptcloud_s3_service_enable" "hoanglm32" {
  vpc_id = "your_vpc_id"
}
// All regions formatted
output "all_regions_formatted" {
  value = {
    for region in data.fptcloud_s3_service_enable.hoanglm32.s3_enable_services :
    region.s3_service_name => {
      id          = region.s3_service_id
      platform    = region.s3_platform
      region_name = region.s3_service_name
    }
  }
}
// Region name only, * for all if you want specific index, use [0], [1], ...
output "region_name" {
  value = data.fptcloud_s3_service_enable.hoanglm32.s3_enable_services[*].s3_service_name
}
