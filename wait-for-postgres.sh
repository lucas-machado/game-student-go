#!/bin/bash
# wait-for-postgres.sh

set -e

host="$1"
port=$2

until PGPASSWORD=ps_password psql -h "$host" -p "$port" -U "ps_user" -d "backend" -c '\q'; do
  >&2 echo "Postgres is unavailable - sleeping"
  sleep 1
done

>&2 echo "Postgres is up - executing command"