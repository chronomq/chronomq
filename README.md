# Chronomq

Chronomq is a high-throughput scheduleable-job queue. Jobs with different trigger times go in - jobs ordered by trigger by time come out.

[![Go Report Card](https://goreportcard.com/badge/github.com/chronomq/chronomq)](https://goreportcard.com/report/github.com/chronomq/chronomq)
[![Build Status](https://travis-ci.com/chronomq/chronomq.svg?branch=master)](https://travis-ci.com/chronomq/chronomq)
[![GoDoc](https://godoc.org/github.com/chronomq/chronomq/pkg?status.svg)](https://godoc.org/github.com/chronomq/chronomq/pkg)

- [What is Chronomq?](#what-is-chronomq)
- [Quickstart](#quickstart)
  - [Docker](#docker)
  - [Pre-built Binary](#pre-built-binary)
  - [Running built-in load test](#running-built-in-load-test)
- [Configuration](#configuration)
- [Related work and inspiration](#Related-work-and-inspiration)

## What is Chronomq

Chronomq is a job queue (like rabbitMQ) but for jobs that trigger at some time. You add jobs to it with a specific date/time trigger and they are made available to come off the queue at that time.

In it's simplest form, Chronomq is an in-memory queue.
Producers can enqueue jobs onto Chronomq with a `trigger` time and Chronomq will internally order them by their `trigger` time. Jobs with `trigger` time closest to `now` are dequeued before jobs with later `trigger` times.
In other words, Chronomq orders a stream of jobs by their `trigger` time.

### Use cases

- Scheduled Emails/SMSs - send messages some time after an event occurs
- Abandoned shopping cart reminders
- Deadman switches - perform pre-set action if job isn't canceled before expiry
- Streaming Time-sorted messages - enqueue jobs in a window and consume a sorted stream
- Asynchronous work queue - run time-consuming tasks asynchronously

## Quickstart

### Docker

1. Pull latest image docker: `docker pull chronomq/chronomq:latest`
1. Run server: `docker run --rm --name chronomq -p 11301:11301 chronomq/chronomq:latest -L server`
1. Interact using the cli:

   ```bash
   docker run --rm chronomq/chronomq:latest put --id "j1" --body "I should be ready second" --delay 45s
   docker run --rm chronomq/chronomq:latest put --id "j2" --body "I should be ready first" --delay 2s
   docker run --rm chronomq/chronomq:latest put --id "j3" --body "Cancel me" --delay 20m
   docker run --rm chronomq/chronomq:latest next --timeout 20s
   docker run --rm chronomq/chronomq:latest next --timeout 50s
   docker run --rm chronomq/chronomq:latest cancel --id "j3"
   ```

### Pre-built binary

1. Fetch binary
   1. Github releases `https://github.com/chronomq/chronomq/releases`
   1. Or via `go get`: `go get github.com/chronomq/chronomq`
1. Run server: `chronomq -L server`
1. Interact using the cli:

   ```bash
   chronomq put --id "j1" --body "I should be ready second" --delay 45s
   chronomq put --id "j2" --body "I should be ready first" --delay 2s
   chronomq put --id "j3" --body "Cancel me" --delay 20m
   chronomq next --timeout 20s
   chronomq next --timeout 50s
   chronomq cancel --id "j3"
   ```

#### Running built-in load test

1. Docker:

   ```bash
      docker run --rm --name loadtest chronomq/chronomq:latest \
         -L loadtest -e -d \
         --raddr \$(docker inspect chronomq -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}'):11301
   ```

1. Run a load test: `chronomq -L loadtest -e -d`

## Configuration

### Common configuration options

1. StatsD server address `--statsAddr string Remote StatsD listener (host:port) (default ":8125")`
1. Enable human-friendly coloured logs `-l, --log-level string Set log level: INFO, DEBUG (default "INFO")`
1. Set log level `-L, --friendly-log Use a human-friendly logging style`
1. RPC listen address `--raddr string Bind RPC listener to (host:port) (default ":11301")`
   1. For `Server` mode, it the address the server should advertize on
   1. For other modes, it is the address of the target server

### Operation Mode: Server

Runs Chronomq queue server. If `restore` is used, the server first attempts to restore jobs from the give snapshot location and then starts serving the queue. If the server receives a `SIGUSR1`, it exits gracefully by creating a new snapshot with any jobs that were held in memory at shutdown time.
The snapshots can be copied to different machines and supplied to new server instances.

1. Restore jobs from a snapshot `--restore`
   1. Loads snapshots from `--store-url`
      1. Filesystem dir (default: PWD) or S3-style url.
         Examples: filesystemdir/subdir or {file|s3|gs|azblob}://bucket (default "/usr/local/bin")
      1. An optional `--store-prefix` can also be provided for S3 compatible addressing scheme
1. SpokeSpan is an advanced tuning parameter. It sets the `bucket` size for job ordering.
   `-S, --spokeSpan duration Spoke span (golang duration string format) (default 10s)`
   It configures the spread of job `trigger` times.

### Operation Mode: Loadtest

Runs a loadtest that generated jobs with random payloads with their `trigger` times between the `delayMin` and the `delayMax` values (relative to when they're generated).

1. Set max concurrent producer connections `-c, --con int Number of connections to use (default 5)`
1. Max `trigger` time for test jobs. A random trigger time between `delayMin` and `delayMax` is assigned to test jobs `-M, --delayMax int Max delay in seconds (Delay is random over delayMin, delayMax) (default 60)`
1. Min `trigger` time for test jobs `-N, --delayMin int Min delay in seconds (Delay is random over delayMin, delayMax)`
1. Set dequeue mode `-d, --dequeueMode Dequeue jobs`
1. Set enqueue mode `-e, --enqueue Enqueue jobs`
1. Number of jobs to generated `-n, --num int Number of total jobs (default 1000)`
1. Fixed job payload size `-z, --size int Job size in bytes (default 1000)`

### Operation Mode: Inspect

1. Number of Jobs to fetch `-n, --num int Max Number of jobs to inspect (default 1)`
   - If the target server has fewer than `num` jobs, it will return less than `num` jobs.
1. Inspect output file location `-o, --out string Write output to outfile (default: stdout)`

## Related work and inspiration

- [Beanstalkd](https://github.com/beanstalkd/beanstalkd)
- [RabbitMQ](https://www.rabbitmq.com)
