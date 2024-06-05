#!/bin/bash

# Ensure the script exits if any command fails
set -e

# Check if a name argument is provided
if [ -z "$1" ]; then
    echo "Usage: $0 <name>"
    exit 1
fi

# Get the name from the command line argument
names=$1
rounds=$2

# Define the endpoint URL
ENDPOINT="http://localhost:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/hardfork?name=$names"

# Send a GET request to the endpoint and store the response
RESPONSE=$(curl -s -w "%{http_code}" -o response_body.txt "$ENDPOINT")

# Extract the HTTP status code from the response
BODY=$(echo "$RESPONSE" | sed '$d')
HTTP_STATUS=$(echo "$RESPONSE" | tail -n1)
echo $HTTP_STATUS

# Check if the HTTP status code is 200 (OK)
if [ "$HTTP_STATUS" -eq 200 ]; then
    echo $BODY
    exit 0
    fi
else
    echo "Failed to call endpoint. HTTP status: $HTTP_STATUS"
    exit 1
fi
