terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.0"
    }
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "HW7-Cascading-Failure"
      Environment = var.environment
      Version     = var.app_version
      CreatedBy   = "Terraform"
    }
  }
}

provider "docker" {
  host = "unix:///var/run/docker.sock"
}
