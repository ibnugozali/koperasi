#!/bin/bash
# Script to import koperasi.sql database dump into PostgreSQL

# Variables - update these as needed
DB_USER="your_db_user"
DB_NAME="your_db_name"
SQL_FILE="database/koperasi.sql"
PSQL_BIN=$(which psql)

if [ -z "$PSQL_BIN" ]; then
  echo "psql command not found. Please install PostgreSQL client tools."
  exit 1
fi

if [ ! -f "$SQL_FILE" ]; then
  echo "SQL file not found at path: $SQL_FILE"
  exit 1
fi

echo "Importing database from $SQL_FILE into database $DB_NAME as user $DB_USER ..."

$PSQL_BIN -U "$DB_USER" -d "$DB_NAME" -f "$SQL_FILE"

if [ $? -eq 0 ]; then
  echo "Database import completed successfully."
else
  echo "Database import failed."
fi
