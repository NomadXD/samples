#!/bin/bash

# Define the output file
output_file="status_output.csv"

# Write the header to the CSV file
echo "Timestamp,Active connections,Server accepts,Server handled,Server requests,Reading,Writing,Waiting" > $output_file

# Function to process the curl output and append to the CSV file
process_output() {
  local timestamp=$(date +"%Y-%m-%d %H:%M:%S")
  local output=$1
  
  # Parse the output
  local active_connections=$(echo "$output" | grep "Active connections" | awk '{print $3}')
  local server_stats=$(echo "$output" | awk 'NR==3 {print $1 "," $2 "," $3}')
  local reading=$(echo "$output" | grep "Reading" | awk '{print $2}')
  local writing=$(echo "$output" | grep "Writing" | awk '{print $4}')
  local waiting=$(echo "$output" | grep "Waiting" | awk '{print $6}')
  
  # Append the parsed data to the CSV file
  echo "$timestamp,$active_connections,$server_stats,$reading,$writing,$waiting" >> $output_file
}

# Infinite loop to call curl every 10 seconds
while true; do
  output=$(curl -s localhost:8080/basic_status)
  process_output "$output"
  sleep 10
done
