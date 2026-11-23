terraform {
  required_version = ">= 1.7"

  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 3.4"
    }
  }

  # Local state: this stack only exists on the candidate's machine.
}

provider "kubernetes" {
  config_path    = "~/.kube/config"
  config_context = var.kube_context
}
