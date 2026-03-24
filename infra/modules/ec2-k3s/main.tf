data "aws_vpc" "default" {
  default = true
}

resource "aws_security_group" "nll_scheduler" {
  name        = "nll-scheduler-${var.tester_name}"
  description = "NLL Scheduler tester environment for ${var.tester_name}"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "HTTP - nll-scheduler app"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "SSH - admin access"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    description = "All outbound - required for image pulls and k3s install"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name    = "nll-scheduler-${var.tester_name}"
    Project = "nll-scheduler"
    Tester  = var.tester_name
  }
}

resource "aws_instance" "nll_scheduler" {
  ami                    = var.ami_id
  instance_type          = var.instance_type
  vpc_security_group_ids = [aws_security_group.nll_scheduler.id]
  key_name               = var.key_name != "" ? var.key_name : null

  user_data = templatefile("${path.module}/cloud-init.sh.tpl", {
    app_version = var.app_version
  })

  root_block_device {
    volume_size = 20
    volume_type = "gp3"
    tags = {
      Name    = "nll-scheduler-${var.tester_name}-root"
      Project = "nll-scheduler"
      Tester  = var.tester_name
    }
  }

  tags = {
    Name    = "nll-scheduler-${var.tester_name}"
    Project = "nll-scheduler"
    Tester  = var.tester_name
  }
}
