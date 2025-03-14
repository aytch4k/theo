#!/bin/bash

# Run database verification script
echo "Starting database verification..."
python3 /app/scripts/db-verify/verify_db.py

# Exit with the script's exit code
exit $?