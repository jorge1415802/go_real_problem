## About the proyect
A high-throughput Go CronJob built with Hexagonal Architecture (Ports & Adapters) that ingests and synchronizes large datasets from an external API into a relational database within seconds using concurrent worker pools.

## Stack
- [Go](https://golang.org)
- [mysql](https://www.mysql.com/)
- [kubernetes](https://kubernetes.io/) 
- [Docker](https://www.docker.com/)

## Features
- Concurrent worker pool
- Batch synchronization
- High-throughput ingestion

## Usage
1. git clone the repository
2. Create and configure variables in .env file
3. run this command go mod download
4. run this command go run cmd/sync-job/main.go

## Author
- [Jorge Luis Rodriguez Github](https://github.com/jorge1415802)
- [Jorge Luis Rodriguez Linkedin](https://www.linkedin.com/in/jorge-luis-rodriguez-756793154/)