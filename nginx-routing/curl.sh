#!/bin/bash

url="http://localhost:8080/ping"
output_directory="responses-new"
mkdir -p "$output_directory"

# Function to send a single HTTP request and save the response
send_request() {
    local id=$1
    curl -s -o "$output_directory/response_${id}.txt" -w "\nHTTP Code: %{http_code}\nTotal Time: %{time_total}\n" $url
}

# Number of requests to send
num_requests=1000
# Number of concurrent requests
concurrency=100

for ((i=1; i<=num_requests; i++)); do
    ((current_jobs=$(jobs -r -p | wc -l)))
    while ((current_jobs >= concurrency)); do
        sleep 0.1
        ((current_jobs=$(jobs -r -p | wc -l)))
    done
    send_request $i &
done

wait
echo "All requests completed."
