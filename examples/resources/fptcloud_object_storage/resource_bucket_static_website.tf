resource "fptcloud_object_storage_bucket_static_website" "example_static_website" {
  vpc_id                = "your_vpc_id"
  region_name           = "your_region"
  bucket_name           = "your_bucket_name"
  index_document_suffix = "your_index_document_suffix"
  error_document_key    = "your_error_document_suffix"
}
