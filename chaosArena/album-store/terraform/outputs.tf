output "alb_dns_name" {
  description = "ALB DNS name — use this as the ChaosArena base_url"
  value       = "http://${aws_lb.album_store.dns_name}"
}

output "ec2_1_public_ip" {
  description = "Public IP of EC2-1 (DB + App)"
  value       = aws_instance.album_store_1.public_ip
}

output "ec2_2_public_ip" {
  description = "Public IP of EC2-2 (App only)"
  value       = aws_instance.album_store_2.public_ip
}

output "ec2_1_private_ip" {
  description = "Private IP of EC2-1 (for EC2-2 DATABASE_URL)"
  value       = aws_instance.album_store_1.private_ip
}

output "ssh_ec2_1" {
  description = "SSH command for EC2-1"
  value       = "ssh -i ~/.ssh/${var.key_name}.pem ubuntu@${aws_instance.album_store_1.public_ip}"
}

output "ssh_ec2_2" {
  description = "SSH command for EC2-2"
  value       = "ssh -i ~/.ssh/${var.key_name}.pem ubuntu@${aws_instance.album_store_2.public_ip}"
}

output "ec2_3_public_ip" {
  description = "Public IP of EC2-3 (App only)"
  value       = aws_instance.album_store_3.public_ip
}

output "ssh_ec2_3" {
  description = "SSH command for EC2-3"
  value       = "ssh -i ~/.ssh/${var.key_name}.pem ubuntu@${aws_instance.album_store_3.public_ip}"
}
