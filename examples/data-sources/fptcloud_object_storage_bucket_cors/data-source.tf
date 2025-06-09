data "fptcloud_object_storage_bucket_cors" "example_bucket_cors" {
  vpc_id      = "1b413c55-b752-4183-abad-06c4b5aca6ad"
  region_name = "HCM-02"
  bucket_name = "hoanglm3-test-terraform-static-website"
  page        = 1
  page_size   = 100
}

output "bucket_cors" {
  value = data.fptcloud_object_storage_bucket_cors.example_bucket_cors
}
