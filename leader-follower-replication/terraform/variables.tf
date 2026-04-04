variable "ssh_cidr" {
  type        = string
  description = "Your home IP in CIDR notation (e.g. 1.2.3.4/32)"
}

variable "ssh_key_name" {
  type        = string
  description = "Name of your existing AWS key pair"
}

variable "instance_type" {
  type        = string
  description = "EC2 instance type"
  default     = "t3.large"
}

variable "region" {
  type    = string
  default = "us-west-2"
}
