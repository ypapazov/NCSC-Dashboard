output "instance_id" {
  value = aws_instance.app.id
}

output "instance_public_ip" {
  value = aws_instance.app.public_ip
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

# --- SES (Keycloak SMTP credentials) ---

output "ses_smtp_host" {
  description = "SES SMTP endpoint — use in Keycloak realm Email settings."
  value       = "email-smtp.${var.aws_region}.amazonaws.com"
}

output "ses_smtp_username" {
  description = "SMTP username for Keycloak (IAM access key)."
  value       = aws_iam_access_key.ses_smtp.id
}

output "ses_smtp_password" {
  description = "SMTP password for Keycloak (SES-derived from IAM secret)."
  value       = aws_iam_access_key.ses_smtp.ses_smtp_password_v4
  sensitive   = true
}

output "ses_dkim_tokens" {
  description = "DKIM tokens — create CNAME records if not using Route 53."
  value       = aws_ses_domain_dkim.main.dkim_tokens
}
