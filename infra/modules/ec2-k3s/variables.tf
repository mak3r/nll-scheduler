variable "tester_name" {
  description = "Name of the tester — used to tag and name all AWS resources (e.g. 'alex')"
  type        = string
}

variable "ami_id" {
  description = "openSUSE Leap Micro 6 aarch64 AMI ID for your target region (find in AWS Marketplace)"
  type        = string
}

variable "instance_type" {
  description = "EC2 instance type — t4g.small is ARM-based and free-tier eligible"
  type        = string
  default     = "t4g.small"
}

variable "aws_region" {
  description = "AWS region to deploy into"
  type        = string
  default     = "us-east-1"
}

variable "key_name" {
  description = "EC2 key pair name for SSH access (leave empty to disable SSH)"
  type        = string
  default     = ""
}

variable "app_version" {
  description = "Git ref (branch, tag, or SHA) for ArgoCD to sync manifests from"
  type        = string
  default     = "main"
}

variable "image_tag" {
  description = "Container image tag to deploy (matches a tag pushed to ghcr.io)"
  type        = string
  default     = "latest"
}
