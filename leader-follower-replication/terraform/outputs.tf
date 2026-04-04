output "instance_public_ip" {
  value       = aws_instance.kv_host.public_ip
  description = "SSH: ssh -i <key>.pem ec2-user@<ip>"
}

output "instance_public_dns" {
  value = aws_instance.kv_host.public_dns
}

output "ecr_repository_url" {
  value       = aws_ecr_repository.kv_node.repository_url
  description = "Use this in scripts/deploy.sh as ECR_REPO"
}

output "lf1_leader_url" {
  value = "http://${aws_instance.kv_host.public_ip}:8010"
}

output "lf2_leader_url" {
  value = "http://${aws_instance.kv_host.public_ip}:8020"
}

output "lf3_leader_url" {
  value = "http://${aws_instance.kv_host.public_ip}:8030"
}

output "leaderless_node0_url" {
  value = "http://${aws_instance.kv_host.public_ip}:8040"
}
