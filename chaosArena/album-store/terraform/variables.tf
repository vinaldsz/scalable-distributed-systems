variable "region" {
  default = "us-west-2"
}

variable "vpc_id" {
  default = "vpc-09fdd07bf39275bd9"
}

variable "subnet_id" {
  default = "subnet-09f5d858804d6d9a7"
}

variable "instance_type" {
  default = "t3.large"
}

variable "s3_bucket" {
  default = "album-store-photos-chaos"
}

variable "subnet_id_2" {
  default = "subnet-0b06252d65d790d93"
}

variable "key_name" {
  description = "EC2 key pair name for SSH access"
  type        = string
}
