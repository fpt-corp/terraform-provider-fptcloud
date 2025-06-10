resource "fptcloud_object_storage_bucket_cors" "example_bucket_cors" {
  vpc_id      = "1b413c55-b752-4183-abad-06c4b5aca6ad"
  region_name = "HCM-02"
  bucket_name = "a-hoanglm32-test"

  # Option 1: Load cors config from file
  cors_config_file = file("${path.module}/your_bucket_cors_config.json")

  # Option 2: Inline cors_config
  # cors_config = jsonencode({
  #  {
  #     "ID": "a9099",
  #     "AllowedOrigins": ["http://www.example.com", "http://www.example2.com"],
  #     "AllowedMethods": ["GET", "PUT", "DELETE"],
  #     "MaxAgeSeconds": 3000,
  #     "ExposeHeaders": ["Etag", "x-amz"],
  #     "AllowedHeaders": ["*", "demo"]
  #   }
  # })
}
output "bucket_cors" {
  value = fptcloud_object_storage_bucket_cors.example_bucket_cors.status
}
