variable "aws_region" {
  type    = string
  default = "us-west-2"
}

variable "environment" {
  type    = string
  default = "dev"
}

variable "receiver_image" {
  type        = string
  description = "ECR image URI for order-receiver"
}

variable "processor_image" {
  type        = string
  description = "ECR image URI for order-processor"
}

variable "receiver_desired_count" {
  type    = number
  default = 1
}

variable "processor_desired_count" {
  type    = number
  default = 1
}

variable "processor_workers" {
  type    = number
  default = 1
}
