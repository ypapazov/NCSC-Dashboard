# SES domain identity for sending email (nudge/escalation from Fresnel).
# The Fresnel app calls the SES v2 API using the EC2 instance role (see ec2.tf).
# After apply, add the DKIM CNAME records to your DNS to pass verification.

resource "aws_ses_domain_identity" "main" {
  domain = var.ses_domain != "" ? var.ses_domain : var.domain_name
}

resource "aws_ses_domain_dkim" "main" {
  domain = aws_ses_domain_identity.main.domain
}

# If using Route 53, create the DKIM verification records automatically.
resource "aws_route53_record" "ses_dkim" {
  count   = var.route53_zone_id != "" ? 3 : 0
  zone_id = var.route53_zone_id
  name    = "${aws_ses_domain_dkim.main.dkim_tokens[count.index]}._domainkey.${aws_ses_domain_identity.main.domain}"
  type    = "CNAME"
  ttl     = 300
  records = ["${aws_ses_domain_dkim.main.dkim_tokens[count.index]}.dkim.amazonses.com"]
}
