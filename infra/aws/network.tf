# Use the default VPC — no custom networking for a PoC.
data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
  filter {
    name   = "default-for-az"
    values = ["true"]
  }
}

# Look up the subnet we'll place the instance in, so we can read its AZ.
data "aws_subnet" "selected" {
  id = tolist(data.aws_subnets.default.ids)[0]
}
