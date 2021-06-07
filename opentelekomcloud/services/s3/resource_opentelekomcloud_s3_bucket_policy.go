package s3

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/fmterr"
)

func ResourceS3BucketPolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceS3BucketPolicyPut,
		ReadContext:   resourceS3BucketPolicyRead,
		UpdateContext: resourceS3BucketPolicyPut,
		DeleteContext: resourceS3BucketPolicyDelete,

		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"policy": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     common.ValidateJsonString,
				DiffSuppressFunc: common.SuppressEquivalentAwsPolicyDiffs,
			},
		},
	}
}

func resourceS3BucketPolicyPut(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	s3conn, err := config.S3Client(config.GetRegion(d))
	if err != nil {
		return fmterr.Errorf("Error creating OpenTelekomCloud s3 client: %s", err)
	}

	bucket := d.Get("bucket").(string)
	policy := d.Get("policy").(string)

	d.SetId(bucket)

	log.Printf("[DEBUG] S3 bucket: %s, put policy: %s", bucket, policy)

	params := &s3.PutBucketPolicyInput{
		Bucket: aws.String(bucket),
		Policy: aws.String(policy),
	}

	err = resource.Retry(1*time.Minute, func() *resource.RetryError {
		if _, err := s3conn.PutBucketPolicy(params); err != nil {
			if awserr, ok := err.(awserr.Error); ok {
				if awserr.Code() == "MalformedPolicy" {
					return resource.RetryableError(awserr)
				}
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return fmterr.Errorf("Error putting S3 policy: %s", err)
	}

	return nil
}

func resourceS3BucketPolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	s3conn, err := config.S3Client(config.GetRegion(d))
	if err != nil {
		return fmterr.Errorf("Error creating OpenTelekomCloud s3 client: %s", err)
	}

	log.Printf("[DEBUG] S3 bucket policy, read for bucket: %s", d.Id())
	pol, err := s3conn.GetBucketPolicy(&s3.GetBucketPolicyInput{
		Bucket: aws.String(d.Id()),
	})

	v := ""
	if err == nil && pol.Policy != nil {
		v = *pol.Policy
	}
	if err := d.Set("policy", v); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceS3BucketPolicyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	s3conn, err := config.S3Client(config.GetRegion(d))
	if err != nil {
		return fmterr.Errorf("Error creating OpenTelekomCloud s3 client: %s", err)
	}

	bucket := d.Get("bucket").(string)

	log.Printf("[DEBUG] S3 bucket: %s, delete policy", bucket)
	_, err = s3conn.DeleteBucketPolicy(&s3.DeleteBucketPolicyInput{
		Bucket: aws.String(bucket),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoSuchBucket" {
			return nil
		}
		return fmterr.Errorf("Error deleting S3 policy: %s", err)
	}

	return nil
}
