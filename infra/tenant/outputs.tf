output "public_ip" {
  description = "Public IP address of the tester EC2 instance"
  value       = module.ec2_k3s.public_ip
}

output "app_url" {
  description = "URL for the tester — available ~3-5 minutes after apply completes"
  value       = module.ec2_k3s.app_url
}

output "instance_id" {
  description = "EC2 instance ID"
  value       = module.ec2_k3s.instance_id
}
