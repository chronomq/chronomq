# Yaad

Yaad is a high-throughput scheduleable-job queue. Jobs with different trigger times go in - jobs ordered by trigger by time come out.

[![Go Report Card](https://goreportcard.com/badge/github.com/urjitbhatia/goyaad)](https://goreportcard.com/report/github.com/urjitbhatia/goyaad)
[![Build Status](https://travis-ci.com/urjitbhatia/goyaad.svg?branch=master)](https://travis-ci.com/urjitbhatia/goyaad)
[![GoDoc](https://godoc.org/github.com/urjitbhatia/goyaad/pkg?status.svg)](https://godoc.org/github.com/urjitbhatia/goyaad/pkg)

- [What is Yaad?](#what-is-yaad)
- [Quickstart](#quickstart)
- [Configuration](#configuration)
- [Related work and inspiration](#Related-work-and-inspiration)

## What is Yaad

In it's simplest form, Yaad is an in-memory queue.
Producers can enqueue jobs onto Yaad with a `trigger` time and will internally order them by their `trigger` time. Jobs with `trigger` time closest to `now` are dequeued before jobs with later `trigger` times.
In other words, Yaad orders a stream of jobs by their `trigger` time.

## Quickstart

1. Install Yaad

   1. Via docker: `docker pull urjitbhatia/goyaad:latest`
   1. Using pre-built releases : `https://github.com/urjitbhatia/goyaad/releases`
   1. Installing via the `go get` : `go get github.com/urjitbhatia/yaad`

1. Start an instance

   1. Using Docker: `docker run --rm --name goyaad -p 11301:11301 urjitbhatia/goyaad:latest -L server`
   1. Using a binary Docker: `goyaad -L server`

1. Run a load test

   1. Using Docker

      ```bash
      docker run --rm --name loadtest urjitbhatia/goyaad:latest \
          -L loadtest -e -d \
          --raddr $(docker inspect goyaad -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}'):11301
      ```

   1. Using a binary: `goyaad -L loadtest -e -d`

## Configuration

### Common configuration options

1. StatsD server address `--statsAddr string Remote StatsD listener (host:port) (default ":8125")`
1. Enable human-friendly coloured logs `-l, --log-level string Set log level: INFO, DEBUG (default "INFO")`
1. Set log level `-L, --friendly-log Use a human-friendly logging style`
1. RPC listen address `--raddr string Bind RPC listener to (host:port) (default ":11301")`
   1. For `Server` mode, it the address the server should advertize on
   1. For other modes, it is the address of the target server

### Operation Mode: Server

Runs Yaad queue server. If `restore` is used, the server first attempts to restore jobs from the give snapshot location and then starts serving the queue. If the server receives a `SIGUSR1`, it exits gracefully by creating a new snapshot with any jobs that were held in memory at shutdown time.
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
