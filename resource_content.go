package main

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/customdiff"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceServerCreate,
		Read:   resourceServerRead,
		Update: resourceServerUpdate,
		Delete: resourceServerDelete,
		CustomizeDiff: customdiff.If(
			func(d *schema.ResourceDiff, meta interface{}) bool {
				return d.Id() != ""
			},
			func(d *schema.ResourceDiff, meta interface{}) error {
				contentManager, err := NewContentManager(d.Id())
				if err != nil {
					return err
				}
				d.SetNew("files", contentManager.Files)
				return nil
			},
		),
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
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceServerCreate(d *schema.ResourceData, m interface{}) error {
	path := d.Get("path").(string)
	bucket := d.Get("bucket").(string)
	value, _ := d.GetOk("types")
	contentManager, err := NewContentManager(path)
	if err != nil {
		return err
	}

	err = contentManager.Write(d, bucket, contentManager.Files, value.(map[string]interface{}))
	if err != nil {
		return err
	}
	d.Set("files", contentManager.Files)
	d.SetId(path)
	return nil
}

func resourceServerRead(d *schema.ResourceData, m interface{}) error {

	bucket := d.Get("bucket").(string)
	contentManager, err := NewContentManager(d.Id())
	if err != nil {
		return err
	}

	files, err := contentManager.Read(d, bucket)
	if err != nil {
		return err
	}

	d.Set("files", files)
	return nil
}

func resourceServerUpdate(d *schema.ResourceData, m interface{}) error {

	path := d.Get("path").(string)
	bucket := d.Get("bucket").(string)
	mappings, _ := d.GetOk("types")

	if d.HasChange("files") {
		old, new := d.GetChange("files")
		oldValue := old.(map[string]interface{})
		newValue := new.(map[string]interface{})
		contentManager, err := NewContentManager(path)
		if err != nil {
			return err
		}

		remove := make(map[string]interface{})
		for key, value := range oldValue {
			file := fmt.Sprintf("%v", key)
			uri := fmt.Sprintf("%v", value)
			if _, ok := newValue[file]; !ok {
				remove[file] = uri
			}
		}
		err = contentManager.Delete(d, bucket, remove)
		if err != nil {
			return err
		}

		add := make(map[string]interface{})
		for key, value := range newValue {
			file := fmt.Sprintf("%v", key)
			uri := fmt.Sprintf("%v", value)
			if _, ok := oldValue[file]; !ok {
				add[file] = uri
			}
		}
		err = contentManager.Write(d, bucket, add, mappings.(map[string]interface{}))
		if err != nil {
			return err
		}
	}
	return nil
}

func resourceServerDelete(d *schema.ResourceData, m interface{}) error {
	files, _ := d.GetOk("files")
	bucket := d.Get("bucket").(string)
	contentManager, err := NewContentManager(d.Id())
	if err != nil {
		return err
	}
	err = contentManager.Delete(d, bucket, files.(map[string]interface{}))
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}
