variable "aws_region" {
  type    = string
  default = "us-west-2"
}

variable "environment" {
  type    = string
  default = "dev"
}

variable "existing_sns_topic_name" {
  type    = string
  default = "order-processing-events"
}

variable "lambda_image" {
  type        = string
  description = "ECR image URI for the order processor lambda"
}

variable "lambda_memory_size" {
  type    = number
  default = 512
}

variable "lambda_timeout_seconds" {
  type    = number
  default = 30
}

variable "lambda_reserved_concurrency" {
  type    = number
  default = 5
}
