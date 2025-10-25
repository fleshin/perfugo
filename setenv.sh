#!/usr/bin/env bash
# Source this script to export the environment variables required by Perfugo.
#
# Usage:
#   source ./setenv.sh

# HTTP server configuration
export SERVER_ADDR=":8080"

# Database connection configuration
export DATABASE_URL="postgres://perfugo:Perfugo101@192.168.0.139:5432/perfugo?sslmode=disable&password=Perfugo101"
export DATABASE_MAX_IDLE_CONNS="5"
export DATABASE_MAX_OPEN_CONNS="25"
export DATABASE_CONN_MAX_LIFETIME="30m"
export DATABASE_CONN_MAX_IDLE_TIME="5m"

# Logging configuration
export LOG_LEVEL="debug"

# Session configuration
export SESSION_LIFETIME="12h"
export SESSION_COOKIE_NAME="perfugo_session"
export SESSION_COOKIE_DOMAIN="flecha.cloud"
export SESSION_COOKIE_SECURE="true"
