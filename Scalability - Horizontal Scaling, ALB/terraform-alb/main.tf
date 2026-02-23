# Wire together modules: network, ecr, logging, alb, ecs, autoscaling

module "network" {
  source         = "./modules/network"
  service_name   = var.service_name
  container_port = var.container_port
}

module "ecr" {
  source          = "./modules/ecr"
  repository_name = var.ecr_repository_name
}

module "logging" {
  source            = "./modules/logging"
  service_name      = var.service_name
  retention_in_days = var.log_retention_days
}

# Reuse an existing IAM role for ECS tasks
data "aws_iam_role" "lab_role" {
  name = "LabRole"
}

# Application Load Balancer module (Part 3)
module "alb" {
  source         = "./modules/alb"
  service_name   = var.service_name
  vpc_id         = module.network.vpc_id
  subnet_ids     = module.network.subnet_ids
  container_port = var.container_port
}

# ECS Security Group to allow traffic from ALB
resource "aws_security_group" "ecs_from_alb" {
  name        = "${var.service_name}-ecs-alb-sg"
  description = "Allow inbound from ALB to ECS tasks"
  vpc_id      = module.network.vpc_id

  ingress {
    description     = "Allow traffic from ALB"
    from_port       = var.container_port
    to_port         = var.container_port
    protocol        = "tcp"
    security_groups = [module.alb.alb_security_group_id]
  }

  egress {
    description = "Allow all outbound"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.service_name}-ecs-alb-sg"
  }
}

module "ecs" {
  source             = "./modules/ecs"
  service_name       = var.service_name
  image              = "${module.ecr.repository_url}:latest"
  container_port     = var.container_port
  subnet_ids         = module.network.subnet_ids
  security_group_ids = [aws_security_group.ecs_from_alb.id]
  execution_role_arn = data.aws_iam_role.lab_role.arn
  task_role_arn      = data.aws_iam_role.lab_role.arn
  log_group_name     = module.logging.log_group_name
  ecs_count          = var.ecs_count
  region             = var.aws_region
  # ALB integration (Part 3)
  target_group_arn   = module.alb.target_group_arn
}

# Auto-scaling module (Part 3)
module "autoscaling" {
  source           = "./modules/autoscaling"
  service_name     = var.service_name
  cluster_name     = module.ecs.cluster_name
  service_name_ecs = module.ecs.service_name
  min_capacity     = 2  # Start with 2 tasks for baseline test
  max_capacity     = 4  # Cap at 4 tasks for breaking point test
  cpu_target       = 70.0
}

# Build & push the Go app image into ECR
resource "docker_image" "app" {
  name = "${module.ecr.repository_url}:latest"

  build {
    context    = "../src"
    dockerfile = "../src/Dockerfile"
    platform   = "linux/amd64"
    no_cache   = true
  }
}

resource "docker_registry_image" "app" {
  name = docker_image.app.name
}

