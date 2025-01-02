resource "fptcloud_object_storage_bucket_policy" "example_bucket_policy" {
  vpc_id      = "your_vpc_id"
  region_name = "your_region_name"
  bucket_name = "your_bucket_name"

  // Option 1: Load policy from file
  policy_file = file("${path.module}/your_bucket_policy_json_content.json")

  // Option 2: Inline policy
  // policy = jsonencode({
  //   Version = "2012-10-17"
  //   Statement = [
  //     {
  //       Sid       = "PublicReadGetObject"
  //       Effect    = "Allow"
  //       Principal = "*"
  //       Action    = "s3:GetObject"
  //       Resource  = "arn:aws:s3:::example-bucket/*"
  //     }
  //   ]
  // })
}
// NOTE: In case wanna delete bucket policy, just ignore policy_file and policy fields
output "bucket_policy" {
  value = fptcloud_object_storage_bucket_policy.example_bucket_policy.status
}
