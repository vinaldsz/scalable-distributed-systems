resource "aws_ecr_repository" "search_service" {
  name                 = var.repository_name
  image_tag_mutability = "MUTABLE"
  force_delete         = var.force_delete

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = merge(
    var.common_tags,
    {
      Name = var.repository_name
    }
  )
}
