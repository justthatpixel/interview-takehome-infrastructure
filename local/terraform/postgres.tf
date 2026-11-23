# Postgres for local tests only. The data lives in the pod, so a new pod
# starts again from the init scripts.

locals {
  database = "ping_residential"
  owner    = "ping_dev"

  # Local stand-ins for the passwords that the External Secrets Operator
  # provides in our clusters.
  owner_password = "localdev"
  environment_passwords = {
    development = "dev-local-only"
    production  = "prod-local-only"
  }
}

resource "kubernetes_namespace_v1" "db" {
  metadata {
    name = "db"
  }
}

resource "kubernetes_secret_v1" "owner" {
  metadata {
    name      = "postgres"
    namespace = kubernetes_namespace_v1.db.metadata[0].name
  }
  data = {
    password = local.owner_password
  }
}

# The password for each environment's database user. You can change anything
# about how these secrets are structured: their names, keys and namespaces.
resource "kubernetes_secret_v1" "environment" {
  for_each = local.environment_passwords

  metadata {
    name      = "${each.key}-postgres-password"
    namespace = kubernetes_namespace_v1.db.metadata[0].name
  }
  data = {
    password = each.value
  }
}

resource "kubernetes_config_map_v1" "init" {
  metadata {
    name      = "postgres-init"
    namespace = kubernetes_namespace_v1.db.metadata[0].name
  }
  data = {
    "init.sql" = file("${path.module}/../../services/api/init.sql")
    "roles.sh" = file("${path.module}/../roles.sh")
  }
}

resource "kubernetes_deployment_v1" "postgres" {
  metadata {
    name      = "postgres"
    namespace = kubernetes_namespace_v1.db.metadata[0].name
  }

  spec {
    replicas = 1

    strategy {
      type = "Recreate"
    }

    selector {
      match_labels = {
        app = "postgres"
      }
    }

    template {
      metadata {
        labels = {
          app = "postgres"
        }
      }

      spec {
        # Runs on the server node, so a drain of the workers does not stop it.
        node_selector = {
          "node-role.kubernetes.io/control-plane" = "true"
        }
        toleration {
          key      = "node-role.kubernetes.io/control-plane"
          operator = "Exists"
          effect   = "NoSchedule"
        }

        container {
          name  = "postgres"
          image = "postgres:15"

          env {
            name  = "POSTGRES_USER"
            value = local.owner
          }
          env {
            name  = "POSTGRES_DB"
            value = local.database
          }
          env {
            name = "POSTGRES_PASSWORD"
            value_from {
              secret_key_ref {
                name = kubernetes_secret_v1.owner.metadata[0].name
                key  = "password"
              }
            }
          }

          # Read only by the init scripts, to create the environment users
          dynamic "env" {
            for_each = kubernetes_secret_v1.environment
            content {
              name = "${upper(env.key)}_POSTGRES_PASSWORD"
              value_from {
                secret_key_ref {
                  name = env.value.metadata[0].name
                  key  = "password"
                }
              }
            }
          }

          port {
            container_port = 5432
          }

          # Checks TCP, because during init Postgres listens only on its socket.
          readiness_probe {
            exec {
              command = ["pg_isready", "-h", "127.0.0.1", "-U", local.owner, "-d", local.database]
            }
            period_seconds = 2
          }

          volume_mount {
            name       = "init"
            mount_path = "/docker-entrypoint-initdb.d"
            read_only  = true
          }
        }

        volume {
          name = "init"
          config_map {
            name = kubernetes_config_map_v1.init.metadata[0].name
          }
        }
      }
    }
  }
}

resource "kubernetes_service_v1" "postgres" {
  metadata {
    name      = "postgres"
    namespace = kubernetes_namespace_v1.db.metadata[0].name
  }

  spec {
    selector = {
      app = "postgres"
    }
    port {
      port = 5432
    }
  }
}
