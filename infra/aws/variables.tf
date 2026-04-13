variable "aws_region" {
  type    = string
  default = "eu-west-2"
}

variable "project" {
  type    = string
  default = "fresnel"
}

variable "environment" {
  type    = string
  default = "poc"
}

variable "instance_type" {
  description = "EC2 instance type — 4 vCPU / 16 GB matches HOSTING_REQUIREMENTS.md"
  type        = string
  default     = "t3.xlarge"
}

variable "data_volume_size_gb" {
  description = "EBS data volume size in GB (Postgres, attachments, backups)"
  type        = number
  default     = 100
}

variable "ssh_allowed_cidrs" {
  description = "CIDRs allowed to SSH (management network). Empty = no SSH from internet."
  type        = list(string)
  default     = []
}

variable "domain_name" {
  description = "FQDN for the platform (e.g. fresnel.example.org). Used for ACM cert and DNS."
  type        = string
}

variable "route53_zone_id" {
  description = "Route 53 hosted zone ID for the domain. Leave empty to skip DNS record creation."
  type        = string
  default     = ""
}

variable "key_pair_name" {
  description = "EC2 key pair name for SSH access. Leave empty to use SSM only."
  type        = string
  default     = ""
}

# --- SES ---

variable "ses_domain" {
  description = "Domain to verify in SES for sending email. Defaults to domain_name if empty."
  type        = string
  default     = ""
}

variable "ses_from_address" {
  description = "Sender address for SES (e.g. noreply@fresnel.example.org). IAM policy restricts sending to this address."
  type        = string
  default     = "noreply@fresnel.example.org"
}
