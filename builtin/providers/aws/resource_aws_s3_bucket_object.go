package aws

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

func resourceAwsS3BucketObject() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsS3BucketObjectPut,
		Read:   resourceAwsS3BucketObjectRead,
		Delete: resourceAwsS3BucketObjectDelete,

		Schema: map[string]*schema.Schema{
			"bucket": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"source": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"content"},
			},

			"content": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"source"},
			},

			"etag": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsS3BucketObjectPut(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	bucket := d.Get("bucket").(string)
	key := d.Get("key").(string)
	var body io.ReadSeeker

	if v, ok := d.GetOk("source"); ok {
		source := v.(string)
		file, err := os.Open(source)
		if err != nil {
			return fmt.Errorf("Error opening S3 bucket object source (%s): %s", source, err)
		}

		body = file
	} else if v, ok := d.GetOk("content"); ok {
		content := v.(string)
		body = bytes.NewReader([]byte(content))
	} else {
		return fmt.Errorf("Must specify \"source\" or \"content\" field")
	}

	resp, err := s3conn.PutObject(
		&s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Body:   body,
		})

	if err != nil {
		return fmt.Errorf("Error putting object in S3 bucket (%s): %s", bucket, err)
	}

	d.Set("etag", resp.ETag)
	d.SetId(key)
	return nil
}

func resourceAwsS3BucketObjectRead(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	bucket := d.Get("bucket").(string)
	key := d.Get("key").(string)
	etag := d.Get("etag").(string)

	resp, err := s3conn.HeadObject(
		&s3.HeadObjectInput{
			Bucket:  aws.String(bucket),
			Key:     aws.String(key),
			IfMatch: aws.String(etag),
		})

	if err != nil {
		// If S3 returns a 404 Request Failure, mark the object as destroyed
		if awsErr, ok := err.(awserr.RequestFailure); ok && awsErr.StatusCode() == 404 {
			d.SetId("")
			log.Printf("[WARN] Error Reading Object (%s), object not found (HTTP status 404)", key)
			return nil
		}
		return err
	}

	log.Printf("[DEBUG] Reading S3 Bucket Object meta: %s", resp)
	return nil
}

func resourceAwsS3BucketObjectDelete(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	bucket := d.Get("bucket").(string)
	key := d.Get("key").(string)

	_, err := s3conn.DeleteObject(
		&s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
	if err != nil {
		return fmt.Errorf("Error deleting S3 bucket object: %s", err)
	}
	return nil
}

