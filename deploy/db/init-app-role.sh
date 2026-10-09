#!/bin/sh
# Runs once on first init: the app gets its own non-superuser role that owns
# only its database. PUBLIC loses connect/create rights.
set -eu
psql -v ON_ERROR_STOP=1 --username postgres <<SQL
CREATE ROLE webrdp LOGIN PASSWORD '${APP_DB_PASSWORD}' NOSUPERUSER NOCREATEDB NOCREATEROLE;
CREATE DATABASE webrdp OWNER webrdp;
REVOKE ALL ON DATABASE webrdp FROM PUBLIC;
\c webrdp
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
ALTER SCHEMA public OWNER TO webrdp;
SQL
