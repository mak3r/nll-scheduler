terraform {
  required_providers {
    aws = {
      source  = "registry.opentofu.org/hashicorp/aws"
      version = "~> 5.0"
    }
  }
  required_version = ">= 1.6.0"
}

provider "aws" {
  region = var.aws_region
}

module "ec2_k3s" {
  source        = "../modules/ec2-k3s"
  tester_name   = var.tester_name
  ami_id        = var.ami_id
  instance_type = var.instance_type
  aws_region    = var.aws_region
  key_name      = var.key_name
  app_version   = var.app_version
}
