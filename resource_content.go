package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceServerCreate,
		Read:   resourceServerRead,
		Update: resourceServerUpdate,
		Delete: resourceServerDelete,

		Schema: map[string]*schema.Schema{
			"path": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"bucket": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"types": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"profile": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"files": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceServerCreate(d *schema.ResourceData, m interface{}) error {

	mappings := map[string]string{
		".html":  "text/html",
		".htm":   "text/html",
		".css":   "text/css",
		".scss":  "text/less",
		".gif":   "image/gif",
		".ico":   "image/x-icon",
		".jpg":   "image/jpeg",
		".jpeg":  "image/jpeg",
		".js":    "application/javascript",
		".json":  "application/json",
		".mpeg":  "video/mpeg",
		".png":   "image/png",
		".svg":   "image/svg+xml",
		".swf":   "application/x-shockwave-flash",
		".ts":    "application/typescript",
		".woff":  "font/woff",
		".woff2": "font/woff2",
		".xhtml": "application/xhtml+xml",
		".xml":   "application/xml",
	}
	value, exists := d.GetOk("types")
	if exists {
		mapInterface := value.(map[string]interface{})
		for key, value := range mapInterface {
			strKey := fmt.Sprintf("%v", key)
			strValue := fmt.Sprintf("%v", value)
			mappings[strKey] = strValue
		}
	}

	path := d.Get("path").(string)
	bucket := d.Get("bucket").(string)

	options := session.Options{SharedConfigState: session.SharedConfigEnable}
	value, exists = d.GetOk("profile")
	if exists {
		options.Profile = fmt.Sprintf("%v", value)
	}
	value, exists = d.GetOk("region")
	if exists {
		options.Config = aws.Config{Region: aws.String(fmt.Sprintf("%v", value))}
	}

	// The session the S3 Uploader will use
	sess := session.Must(session.NewSessionWithOptions(options))

	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(sess)

	files := map[string]string{}
	err := filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			f, err := os.Open(file)
			if err != nil {
				return fmt.Errorf("failed to open file %q, %v", file, err)
			}
			var extension = filepath.Ext(file)
			var filename = strings.Replace(strings.Replace(file, path+"\\", "", 1), "\\", "/", -1)
			_, err = uploader.Upload(&s3manager.UploadInput{
				Bucket:      aws.String(bucket),
				Key:         aws.String(filename),
				Body:        f,
				ContentType: aws.String(mappings[extension]),
			})
			f.Close()

			if err != nil {
				return fmt.Errorf("%q upload to s3 failed %v", file, err)
			}

			files[file] = filename
		}
		return nil
	})
	d.Set("files", files)
	if err != nil {
		return fmt.Errorf("filepath.Walk of %q failed %v", path, err)
	}
	d.SetId(path)
	return nil
}

func resourceServerRead(d *schema.ResourceData, m interface{}) error {
	return nil
}

func resourceServerUpdate(d *schema.ResourceData, m interface{}) error {
	return nil
}

func resourceServerDelete(d *schema.ResourceData, m interface{}) error {
	// d.SetId("") is automatically called assuming delete returns no errors, but
	// it is added here for explicitness.
	value, exists := d.GetOk("files")
	if exists {
		files := value.(map[string]interface{})
		options := session.Options{SharedConfigState: session.SharedConfigEnable}

		value, exists = d.GetOk("profile")
		if exists {
			options.Profile = fmt.Sprintf("%v", value)
		}
		value, exists = d.GetOk("region")
		if exists {
			options.Config = aws.Config{Region: aws.String(fmt.Sprintf("%v", value))}
		}

		// The session the S3 Uploader will use
		sess := session.Must(session.NewSessionWithOptions(options))

		// Create an batcher with the session and default options
		batcher := s3manager.NewBatchDelete(sess)

		objects := []s3manager.BatchDeleteObject{} //s3.DeleteObjectInput{}

		bucket := d.Get("bucket").(string)
		for _, value := range files {
			strValue := fmt.Sprintf("%v", value)
			objects = append(objects, s3manager.BatchDeleteObject{
				Object: &s3.DeleteObjectInput{
					Key:    aws.String(strValue),
					Bucket: aws.String(bucket),
				},
			})
		}

		if err := batcher.Delete(aws.BackgroundContext(), &s3manager.DeleteObjectsIterator{
			Objects: objects,
		}); err != nil {
			return fmt.Errorf("batch of %q failed %v", bucket, err)
		}

	}
	d.SetId("")
	return nil
}
