terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # Uncomment after first apply to migrate state to S3.
  # backend "s3" {
  #   bucket         = "fresnel-tfstate-CHANGEME"
  #   key            = "fresnel/terraform.tfstate"
  #   region         = "eu-west-2"
  #   dynamodb_table = "fresnel-tflock"
  #   encrypt        = true
  # }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project   = "fresnel"
      ManagedBy = "terraform"
    }
  }
}
