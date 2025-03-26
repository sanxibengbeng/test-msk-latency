#!/bin/bash

# Wait for Kafka to be ready
echo "Waiting 20 seconds for Kafka to be ready..."
sleep 20

# Check if topic exists
TOPIC_EXISTS=$(docker exec  kafka1 kafka-topics --bootstrap-server kafka1:29092 --list | grep "^test-topic$")

if [ -n "$TOPIC_EXISTS" ]; then
  echo "Topic test-topic exists, checking partition count..."
  
  # Get current partition count
  PARTITION_COUNT=$( docker exec  kafka1 kafka-topics --bootstrap-server kafka1:29092 --describe --topic test-topic | grep "PartitionCount:" | awk '{print $3}')
  
  if [ "$PARTITION_COUNT" -ne 50 ]; then
    echo "Topic has $PARTITION_COUNT partitions, altering to 50 partitions..."
    docker exec kafka1 kafka-topics --bootstrap-server kafka1:29092 --alter --topic test-topic --partitions 50
    echo "Topic altered to 50 partitions."
  else
    echo "Topic already has 50 partitions, no action needed."
  fi
else
  echo "Creating topic test-topic with 50 partitions..."
  docker exec  kafka1 kafka-topics --bootstrap-server kafka1:29092 --create --topic test-topic --partitions 50 --replication-factor 1
  echo "Topic created successfully."
fi

# Verify the topic configuration
echo "Verifying topic configuration:"
docker exec  kafka1 kafka-topics --bootstrap-server kafka1:29092 --describe --topic test-topic

echo "Topic initialization complete."
