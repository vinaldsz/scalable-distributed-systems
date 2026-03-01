# ───────────────────────────────────────────────────────────────────────────────
# HW7 Crashing and Recovering - Modularized Terraform Configuration
# ───────────────────────────────────────────────────────────────────────────────
# This configuration demonstrates the Bulkhead Pattern using isolated thread pools
# across multiple ECS tasks behind an ALB with health checks and auto-scaling.
# ───────────────────────────────────────────────────────────────────────────────

# ── Fetch existing LabRole (has required permissions in lab environment) ───────
data "aws_iam_role" "lab_role" {
  name = "LabRole"
}

# ── ECR Module - Container Image Repository ────────────────────────────────────
module "ecr" {
  source = "./modules/ecr"

  repository_name = "hw7-search-service"
  force_delete    = true
  common_tags     = local.common_tags
}

# ── Build & Push Docker Image to ECR ───────────────────────────────────────────
resource "null_resource" "docker_build_push" {
  triggers = {
    dockerfile_hash = filemd5("${path.module}/../src/Dockerfile")
    source_hash     = sha256(join("", [for f in fileset("${path.module}/../src", "*.go") : filemd5("${path.module}/../src/${f}")]))
    app_version     = var.app_version
  }

  provisioner "local-exec" {
    command = <<-EOT
      cd ${path.module}/..
      aws ecr get-login-password --region ${var.aws_region} | docker login --username AWS --password-stdin ${module.ecr.repository_url}
      docker build --platform linux/amd64 -t ${module.ecr.repository_url}:latest -f src/Dockerfile .
      docker push ${module.ecr.repository_url}:latest
    EOT
  }

  depends_on = [module.ecr]
}

# ── Logging Module - CloudWatch Logs ───────────────────────────────────────────
module "logging" {
  source = "./modules/logging"

  log_group_name    = "/ecs/hw7-search-${var.app_version}"
  retention_in_days = 7
  common_tags       = local.common_tags
}

# ── Network Module - Security Groups ───────────────────────────────────────────
module "network" {
  source = "./modules/network"

  alb_sg_name    = "hw7-alb-sg"
  ecs_sg_name    = "hw7-ecs-tasks-sg"
  container_port = 8080
  common_tags    = local.common_tags
}

# ── ALB Module - Load Balancer and Target Group ────────────────────────────────
module "alb" {
  source = "./modules/alb"

  alb_name               = "hw7-alb"
  alb_security_group_id = module.network.alb_security_group_id
  subnet_ids            = module.network.subnet_ids
  vpc_id                = module.network.vpc_id
  target_group_name     = "hw7-search-${var.app_version}"
  container_port        = 8080
  health_check_path     = "/health"
  common_tags           = local.common_tags
}

# ── ECS Module - Cluster, Task Definition, and Service ────────────────────────
module "ecs" {
  source = "./modules/ecs"

  cluster_name           = "hw7-cluster"
  task_family            = "hw7-search-${var.app_version}"
  task_cpu               = var.task_cpu
  task_memory            = var.task_memory
  task_count             = var.task_count
  execution_role_arn     = data.aws_iam_role.lab_role.arn
  task_role_arn          = data.aws_iam_role.lab_role.arn
  container_name         = "search-service"
  container_image        = "${module.ecr.repository_url}:latest"
  container_port         = 8080
  container_environment = [
    {
      name  = "REC_LATENCY_MS"
      value = "500"
    },
    {
      name  = "REC_MAX_CONCURRENT"
      value = "10"
    }
  ]
  log_group_name         = module.logging.log_group_name
  aws_region             = var.aws_region
  ecs_security_group_id  = module.network.ecs_security_group_id
  subnet_ids             = module.network.subnet_ids
  target_group_arn       = module.alb.target_group_arn
  service_name           = "hw7-search-${var.app_version}"
  alb_listener_dependency = module.alb.alb_listener_arn
  app_version            = var.app_version
  common_tags            = local.common_tags

  depends_on = [
    module.logging,
    module.alb,
    null_resource.docker_build_push
  ]
}

# ── Auto Scaling Module - Service Scaling Policies ─────────────────────────────
module "autoscaling" {
  source = "./modules/autoscaling"

  cluster_name        = module.ecs.cluster_name
  service_name        = module.ecs.service_name
  min_capacity        = var.task_count
  max_capacity        = var.task_max_count
  cpu_policy_name     = "hw7-search-cpu-scaling"
  memory_policy_name  = "hw7-search-memory-scaling"
  cpu_target_value    = 70.0
  memory_target_value = 80.0

  depends_on = [
    module.ecs
  ]
}

# ── Monitoring Module - CloudWatch Alarms and Dashboard ───────────────────────
module "monitoring" {
  source = "./modules/monitoring"

  health_alarm_name       = "hw7-${var.app_version}-health-check-failures"
  alb_arn_suffix          = module.alb.alb_arn_suffix
  target_group_arn_suffix = module.alb.target_group_arn_suffix
  dashboard_name          = "hw7-${var.app_version}"
  dashboard_title         = "HW7 ${upper(var.app_version)} - Service Metrics"
  aws_region              = var.aws_region
  common_tags             = local.common_tags
}

# ── Local Values ───────────────────────────────────────────────────────────────
locals {
  common_tags = {
    Project     = "HW7-Cascading-Failure"
    Environment = var.environment
    Version     = var.app_version
    CreatedBy   = "Terraform"
  }
}
