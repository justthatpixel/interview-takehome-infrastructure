#!/bin/sh
# Creates one read-only login for each environment. The passwords come from
# the environment Secrets, so the Secrets stay the only copy.
psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -v dev_password="$DEVELOPMENT_POSTGRES_PASSWORD" \
  -v prod_password="$PRODUCTION_POSTGRES_PASSWORD" <<'EOSQL'
CREATE ROLE api_development LOGIN PASSWORD :'dev_password';
CREATE ROLE api_production LOGIN PASSWORD :'prod_password';
GRANT CONNECT ON DATABASE ping_residential TO api_development, api_production;
GRANT SELECT ON residential_proxy TO api_development, api_production;
EOSQL
