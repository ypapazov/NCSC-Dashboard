output "vpc_id" {
  value = aws_vpc.main.id
}

output "instance_id" {
  value = aws_instance.app.id
}

output "alb_dns_name" {
  value = aws_lb.main.dns_name
}

output "data_volume_id" {
  value = aws_ebs_volume.data.id
}

output "ssm_connect" {
  description = "SSM Session Manager command to connect to the instance"
  value       = "aws ssm start-session --target ${aws_instance.app.id}"
}
