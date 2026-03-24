output "public_ip" {
  description = "Public IP address of the tester EC2 instance"
  value       = aws_instance.nll_scheduler.public_ip
}

output "app_url" {
  description = "URL for the tester — available ~3-5 minutes after apply completes"
  value       = "http://${aws_instance.nll_scheduler.public_ip}"
}

output "instance_id" {
  description = "EC2 instance ID (for reference or manual operations)"
  value       = aws_instance.nll_scheduler.id
}
