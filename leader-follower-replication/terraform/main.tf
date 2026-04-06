provider "aws" {
  region = var.region
}

# ─── ECR repository for the KV node image ─────────────────────────────────────

resource "aws_ecr_repository" "kv_node" {
  name                 = "kv-node"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = false
  }
}

# ─── Security group ────────────────────────────────────────────────────────────

resource "aws_security_group" "kv_sg" {
  name        = "kv-cluster-sg"
  description = "SSH + KV cluster ports"

  # SSH from your IP only
  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.ssh_cidr]
  }

  # KV cluster ports (all 4 clusters × 5 nodes = 8010-8044)
  ingress {
    description = "KV cluster API"
    from_port   = 8010
    to_port     = 8044
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# ─── Latest Amazon Linux 2023 AMI ─────────────────────────────────────────────

data "aws_ami" "al2023" {
  most_recent = true
  owners      = ["amazon"]
  filter {
    name   = "name"
    values = ["al2023-ami-*-x86_64-ebs"]
  }
}

# ─── EC2 instance ─────────────────────────────────────────────────────────────

resource "aws_instance" "kv_host" {
  ami                    = data.aws_ami.al2023.id
  instance_type          = var.instance_type
  iam_instance_profile   = "LabInstanceProfile"
  vpc_security_group_ids = [aws_security_group.kv_sg.id]
  key_name               = var.ssh_key_name

  # 20GB root volume — enough for Docker images
  root_block_device {
    volume_size = 30
    volume_type = "gp3"
  }

  user_data = templatefile("${path.module}/user_data.sh", {
    ecr_repo  = aws_ecr_repository.kv_node.repository_url
    region    = var.region
  })

  tags = {
    Name = "kv-cluster-host"
  }
}
