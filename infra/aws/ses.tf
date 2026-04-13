# SES domain identity for sending email (nudge/escalation + Keycloak).
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

# IAM user for Keycloak SMTP credentials only.
# Keycloak has no SES SDK — it must use SMTP AUTH, which requires an IAM access key.
# The Fresnel app uses the SES API via the EC2 instance role (see ec2.tf).
resource "aws_iam_user" "ses_smtp" {
  name = "${var.project}-keycloak-smtp"
  tags = { Name = "${var.project}-keycloak-smtp" }
}

resource "aws_iam_user_policy" "ses_smtp" {
  name = "ses-send"
  user = aws_iam_user.ses_smtp.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["ses:SendRawEmail", "ses:SendEmail"]
      Resource = "*"
      Condition = {
        StringEquals = {
          "ses:FromAddress" = var.ses_from_address
        }
      }
    }]
  })
}

resource "aws_iam_access_key" "ses_smtp" {
  user = aws_iam_user.ses_smtp.name
}
