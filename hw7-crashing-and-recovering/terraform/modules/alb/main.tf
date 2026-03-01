resource "aws_lb" "main" {
  name               = var.alb_name
  internal           = false
  load_balancer_type = "application"
  security_groups    = [var.alb_security_group_id]
  subnets            = var.subnet_ids

  enable_deletion_protection = false
  enable_http2               = true

  tags = merge(
    var.common_tags,
    {
      Name = var.alb_name
    }
  )
}

resource "aws_lb_target_group" "search_service" {
  name        = var.target_group_name
  port        = var.container_port
  protocol    = "HTTP"
  vpc_id      = var.vpc_id
  target_type = "ip"

  health_check {
    healthy_threshold   = 2
    unhealthy_threshold = 3
    timeout             = 5
    interval            = 10
    path                = var.health_check_path
    matcher             = "200"
  }

  tags = merge(
    var.common_tags,
    {
      Name = var.target_group_name
    }
  )
}

resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.main.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.search_service.arn
  }
}
