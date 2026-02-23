# Fetch the default VPC
data "aws_vpc" "default" {
  default = true
}

# List all subnets in that VPC
data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

# Create a basic security group for ECS tasks
# Note: For Part 3 with ALB, a more specific security group is created in main.tf
resource "aws_security_group" "this" {
  name        = "${var.service_name}-ecs-basic-sg"
  description = "Security group for ECS tasks"
  vpc_id      = data.aws_vpc.default.id

  # Allow traffic on container port (used for ALB health checks)
  ingress {
    from_port   = var.container_port
    to_port     = var.container_port
    protocol    = "tcp"
    cidr_blocks = var.cidr_blocks
    description = "Allow HTTP traffic"
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Allow all outbound"
  }

  tags = {
    Name = "${var.service_name}-ecs-basic-sg"
  }
}
