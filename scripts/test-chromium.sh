#!/bin/bash

# Exit on error
set -e

echo "Starting services..."
docker compose up -d

retries=0
max_retries=30
service_url="http://localhost:8080/health"

echo "Waiting for service to be ready..."
while [ $retries -lt $max_retries ]; do
    if curl -s "$service_url" > /dev/null; then
        echo "Service is ready!"
        break
    fi
    echo -n "."
    sleep 2
    retries=$((retries + 1))
done

if [ $retries -eq $max_retries ]; then
    echo "Service failed to start after $max_retries retries"
    exit 1
fi

echo "Adding Chromium repository..."
curl -X PUT "http://localhost:8080/api/v1/repositories/chromium/chromium" \
    -H "accept: application/json"

echo "Setup complete. Check docs/api.yaml for available endpoints."
echo "To stop the services: docker compose down" 