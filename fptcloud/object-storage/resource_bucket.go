package fptcloud_object_storage

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	common "terraform-provider-fptcloud/commons"
)

func ResourceBucket() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBucketCreate,
		ReadContext:   resourceBucketRead,
		UpdateContext: resourceBucketUpdate,
		DeleteContext: resourceBucketDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"region": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"storage_class": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "STANDARD",
			},
			"versioning": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceBucketCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)

	req := BucketCreateRequest{
		Name:         d.Get("name").(string),
		Region:       d.Get("region").(string),
		StorageClass: d.Get("storage_class").(string),
		Versioning:   d.Get("versioning").(bool),
		Tags:         d.Get("tags").(map[string]string),
	}

	bucket, err := objectStorageService.CreateBucket(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(bucket.ID)
	return resourceBucketRead(ctx, d, m)
}

func resourceBucketRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Implement the read logic
	return nil
}

func resourceBucketUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Implement the update logic
	return resourceBucketRead(ctx, d, m)
}

func resourceBucketDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Implement the delete logic
	return nil
}