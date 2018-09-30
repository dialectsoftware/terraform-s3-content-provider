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

// This struct is lowercased. Packages that import `fileManager` will not be able
// to instantiate a literal fileManager.
type contentManager struct {
	// This will not be exported.
	Path         string
	Files        map[string]interface{}
	contentTypes map[string]string
}

// This func is uppercased. Packages that import `fileManager` can instantiate a
// new fileManager using this method. This allows us more control around fileManager creation.
func NewContentManager(path string) (*contentManager, error) {
	manager := new(contentManager)
	manager.contentTypes = map[string]string{
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
	err := manager.enumerateFiles(path)
	if err == nil {
		return manager, nil
	}
	return nil, err
}

func (this contentManager) Read(d *schema.ResourceData, bucket string) (map[string]interface{}, error) {
	// The session the S3 Uploader will use
	client := s3.New(this.createSession(d))

	files := make(map[string]interface{})

	err := client.ListObjectsPages(&s3.ListObjectsInput{Bucket: aws.String(bucket)},
		func(page *s3.ListObjectsOutput, lastPage bool) bool {
			for _, value := range page.Contents {
				key := fmt.Sprintf("%v", *value.Key)
				path := d.Id() + "\\" + strings.Replace(key, "/", "\\", -1)
				files[path] = key
			}
			return !lastPage
		})

	if err != nil {
		return nil, fmt.Errorf("Unable to list items in bucket %q, %v", bucket, err)
	}

	return files, nil

}

func (this *contentManager) Write(d *schema.ResourceData, bucket string, files map[string]interface{}, mappings map[string]interface{}) error {

	for key, value := range mappings {
		strKey := fmt.Sprintf("%v", key)
		strValue := fmt.Sprintf("%v", value)
		this.contentTypes[strKey] = strValue
	}
	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(this.createSession(d))

	for key, value := range files {
		file := fmt.Sprintf("%v", key)
		uri := fmt.Sprintf("%v", value)
		extension := filepath.Ext(file)

		f, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("failed to open file %q, %v", file, err)
		}
		_, err = uploader.Upload(&s3manager.UploadInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(uri),
			Body:        f,
			ContentType: aws.String(this.contentTypes[extension]),
		})
		f.Close()

		if err != nil {
			return fmt.Errorf("%q upload to s3 failed %v", file, err)
		}
	}
	return nil
}

func (this *contentManager) Delete(d *schema.ResourceData, bucket string, files map[string]interface{}) error {

	// Create an batcher with the session and default options
	batcher := s3manager.NewBatchDelete(this.createSession(d))
	objects := []s3manager.BatchDeleteObject{}

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

	return nil
}

func (this *contentManager) enumerateFiles(path string) error {
	this.Path = path
	this.Files = make(map[string]interface{})
	err := filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			var filename = strings.Replace(strings.Replace(file, path+"\\", "", 1), "\\", "/", -1)
			this.Files[file] = filename
		}
		return nil
	})

	return err

}

func (this *contentManager) createSession(d *schema.ResourceData) *session.Session {
	options := session.Options{SharedConfigState: session.SharedConfigEnable}
	value, exists := d.GetOk("profile")
	if exists {
		options.Profile = fmt.Sprintf("%v", value)
	}
	value, exists = d.GetOk("region")
	if exists {
		options.Config = aws.Config{Region: aws.String(fmt.Sprintf("%v", value))}
	}

	// The session the S3 Uploader will use
	return session.Must(session.NewSessionWithOptions(options))
}
