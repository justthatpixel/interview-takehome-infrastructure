# Connection details for each environment's database user.
output "postgres" {
  value = {
    for env, secret in kubernetes_secret_v1.environment : env => {
      host            = "${kubernetes_service_v1.postgres.metadata[0].name}.${kubernetes_namespace_v1.db.metadata[0].name}.svc.cluster.local"
      port            = 5432
      user            = "api_${env}"
      database        = local.database
      password_secret = "${secret.metadata[0].namespace}/${secret.metadata[0].name}"
    }
  }
}
