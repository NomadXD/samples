#!/bin/sh
# Wait for Toxiproxy to be ready
sleep 5

# Create the Redis proxy
toxiproxy-cli create redis -l 0.0.0.0:26379 -u redis:6379

# Add latency toxic
toxiproxy-cli toxic add redis -t latency -a latency=8000
