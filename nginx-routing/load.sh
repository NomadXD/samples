#!/bin/bash

url="http://localhost:8080/ping"
output_directory="responses"
mkdir -p "$output_directory"

for ((i=1; i<=10; i++)); do
    hey -n 1000 -c 10 $url | tee "$output_directory/response_$i.txt" > /dev/null
done
