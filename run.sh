#!/bin/bash
# Build the Docker image
# This creates an image named scanner_server from the Dockerfile in the current directory
docker buildx build -t scanner_server .

# Stop the currently running container if it exists
# The `|| true` ensures the script continues even if the command fails because the container does not exist
docker stop scanner_server || true

# Remove the old container if it exists
# This is necessary to free up the name scanner_server for the new container
docker rm scanner_server || true

# Start a new container from the scanner_server image
# Runs in detached mode (-d) and restart if stoped, within the 'mrt' network
docker run --restart=always -d --network=mrt --name scanner_server \
-v $(pwd)/files:/app/files \
-v $(pwd)/uploads:/app/uploads \
scanner_server