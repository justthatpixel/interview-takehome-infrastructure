variable "kube_context" {
  description = "kubectl context of the local k3d cluster."
  type        = string
  default     = "k3d-takehome"
}
