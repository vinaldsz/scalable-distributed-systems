provider "aws" {
  region = var.region
}

# Latest Ubuntu 22.04 AMI
data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

# ─── Security Groups ───

# ALB security group — public HTTP
resource "aws_security_group" "alb" {
  name        = "album-store-alb-sg"
  description = "Allow HTTP traffic to ALB"
  vpc_id      = var.vpc_id

  ingress {
    description = "HTTP"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "album-store-alb-sg"
  }
}

# App security group — ALB → App + SSH + PostgreSQL internal
resource "aws_security_group" "app" {
  name        = "album-store-app-sg"
  description = "Allow traffic from ALB, SSH, and internal PostgreSQL"
  vpc_id      = var.vpc_id

  ingress {
    description     = "App port from ALB"
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]
  }

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "PostgreSQL from VPC"
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    self        = true
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "album-store-app-sg"
  }
}

# ─── ALB ───

resource "aws_lb" "album_store" {
  name               = "album-store-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets            = [var.subnet_id, var.subnet_id_2]
  idle_timeout       = 300 # 5 min — prevent timeout on large uploads

  tags = {
    Name = "album-store-alb"
  }
}

resource "aws_lb_target_group" "app" {
  name     = "album-store-tg"
  port     = 8080
  protocol = "HTTP"
  vpc_id   = var.vpc_id

  health_check {
    path                = "/health"
    interval            = 10
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 2
    matcher             = "200"
  }

  tags = {
    Name = "album-store-tg"
  }
}

resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.album_store.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.app.arn
  }
}

# ─── EC2 Instances ───

# EC2-1: App + PostgreSQL
resource "aws_instance" "album_store_1" {
  ami                         = data.aws_ami.ubuntu.id
  instance_type               = var.instance_type
  subnet_id                   = var.subnet_id
  vpc_security_group_ids      = [aws_security_group.app.id]
  key_name                    = var.key_name
  iam_instance_profile        = "LabInstanceProfile"
  associate_public_ip_address = true

  user_data = templatefile("${path.module}/userdata_db.sh", {
    s3_bucket = var.s3_bucket
  })

  credit_specification {
    cpu_credits = "unlimited"
  }

  root_block_device {
    volume_size = 20
  }

  tags = {
    Name = "album-store-1-db"
  }
}

# EC2-2: App only
resource "aws_instance" "album_store_2" {
  ami                         = data.aws_ami.ubuntu.id
  instance_type               = var.instance_type
  subnet_id                   = var.subnet_id_2
  vpc_security_group_ids      = [aws_security_group.app.id]
  key_name                    = var.key_name
  iam_instance_profile        = "LabInstanceProfile"
  associate_public_ip_address = true

  user_data = templatefile("${path.module}/userdata_app.sh", {
    s3_bucket = var.s3_bucket
  })

  credit_specification {
    cpu_credits = "unlimited"
  }

  root_block_device {
    volume_size = 20
  }

  tags = {
    Name = "album-store-2-app"
  }
}

# EC2-3: App only (third instance for more throughput)
resource "aws_instance" "album_store_3" {
  ami                         = data.aws_ami.ubuntu.id
  instance_type               = var.instance_type
  subnet_id                   = var.subnet_id
  vpc_security_group_ids      = [aws_security_group.app.id]
  key_name                    = var.key_name
  iam_instance_profile        = "LabInstanceProfile"
  associate_public_ip_address = true

  user_data = templatefile("${path.module}/userdata_app.sh", {
    s3_bucket = var.s3_bucket
  })

  credit_specification {
    cpu_credits = "unlimited"
  }

  root_block_device {
    volume_size = 20
  }

  tags = {
    Name = "album-store-3-app"
  }
}

# ─── Target Group Attachments ───

resource "aws_lb_target_group_attachment" "ec2_1" {
  target_group_arn = aws_lb_target_group.app.arn
  target_id        = aws_instance.album_store_1.id
  port             = 8080
}

resource "aws_lb_target_group_attachment" "ec2_2" {
  target_group_arn = aws_lb_target_group.app.arn
  target_id        = aws_instance.album_store_2.id
  port             = 8080
}

resource "aws_lb_target_group_attachment" "ec2_3" {
  target_group_arn = aws_lb_target_group.app.arn
  target_id        = aws_instance.album_store_3.id
  port             = 8080
}
