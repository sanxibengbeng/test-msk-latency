# MSK Delay Test

This project sets up a test environment to measure and analyze consumer group rebalancing behavior in Amazon MSK (Managed Streaming for Kafka).

## Architecture

The test architecture consists of:

1. **Producer**: Continuously generates 1KB messages at a configurable rate
2. **MSK Cluster**: Kafka 3.5.1 with 50 partitions
3. **Consumer Group**: 50 consumers that process messages with a 1.5-second delay

## Project Structure

```
msk-delay/
├── cmd/
│   ├── producer/       # Producer application
│   └── consumer/       # Consumer application
├── pkg/
│   ├── config/         # Configuration handling
│   ├── kafka/          # Kafka client implementations
│   └── logger/         # Logging utilities
├── conf/
│   ├── dev/            # Development environment configuration
│   └── prod/           # Production environment configuration
├── infrastructure/
│   └── cdk/            # AWS CDK deployment code
└── docker-compose.yml  # Local development environment
```

## Development Environment

The development environment uses Docker Compose to run:
- 3 Kafka brokers
- 1 Zookeeper instance
- Producer and consumer applications

### Running Locally

```bash
# Start the development environment
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the environment
docker-compose down
```

## Production Environment

The production environment is deployed using AWS CDK and consists of:
- MSK cluster with 3 brokers
- EC2 instance for the producer
- EC2 instance for the consumer
- CloudWatch logs and metrics

### Deploying to AWS

```bash
# Navigate to the CDK directory
cd infrastructure/cdk

# Install dependencies
npm install

# Deploy the stack
cdk deploy
```

## Configuration

Configuration is stored in JSON files in the `conf` directory:

- `conf/dev/kafka.json`: Development environment configuration
- `conf/prod/kafka.json`: Production environment configuration

Key configuration parameters:
- `brokers`: List of Kafka broker addresses
- `topic`: Kafka topic name
- `partitions`: Number of partitions (50)
- `consumer_group`: Consumer group ID
- `consumer_count`: Number of consumers (50)
- `message_size_kb`: Size of each message (1KB)
- `producer_interval_ms`: Interval between messages
- `consumer_processing_time_ms`: Processing time per message (1500ms)

## Monitoring

In the production environment, the following metrics are available in CloudWatch:
- MSK CPU utilization
- MSK memory usage
- MSK disk usage
- Consumer group rebalancing events per hour

A CloudWatch dashboard is automatically created to visualize these metrics.
