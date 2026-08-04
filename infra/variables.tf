variable "location" {
  type    = string
  default = "centralindia"
}

variable "project_name" {
  type    = string
  default = "orderpulse"
}

variable "ssh_source_ip" {
  description = "YOUR laptop's public IP in CIDR form, e.g. 49.207.12.34/32"
  type        = string
}
