
terraform {
  required_version = "> 0.7.0"
}

variable "aws_region" {
  description = "AWS region to launch servers."
  default     = "us-east-1"
}

provider "aws" {
  version = "~> 1.16"
  region  = "${var.aws_region}"
}

data "aws_region" "current" {}

data "aws_caller_identity" "current" {}


resource "aws_s3_bucket" "s3-content-test" {
  bucket = "terrform-s3-content-test-bucket"
  acl    = "private"
  force_destroy = true

  versioning {
      enabled = false
    }
 
    lifecycle {
      prevent_destroy = false
    }
 
    tags {
      Description = "Test bucket for terraform extension"
    }     
}

resource "s3_content" "my-content" {
    path = "content" 
    bucket = "${aws_s3_bucket.s3-content-test.bucket}"   
    types = {
      ".flux" = "text/flux"

    }
}