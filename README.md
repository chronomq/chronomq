# A simple beanstalkd-like job queue implementation in go

[![Go Report Card](https://goreportcard.com/badge/github.com/urjitbhatia/goyaad)](https://goreportcard.com/report/github.com/urjitbhatia/goyaad)
[![Build Status](https://travis-ci.com/urjitbhatia/goyaad.svg?branch=master)](https://travis-ci.com/urjitbhatia/goyaad)
[![GoDoc](https://godoc.org/github.com/urjitbhatia/goyaad/pkg?status.svg)](https://godoc.org/github.com/urjitbhatia/goyaad/pkg)

## Goals

- Partial support beanstalkd wire protocol
  - Only supports a single default tube and put, reserve, cancel operations
- Allow a disk mapped mode (fully mapped, lazily mapped, full in-mem)

## Architecture

The most fitting architectural analogy for this is to imagine a bicycle wheel.
This wheel spins only in the forward direction (arrow-of-time). At any instant, a `Spoke`
of the wheel is active. The central `Hub` connects all `Spokes` and is responsible for
`rotating` through them.

An `active Spoke` is a `Spoke` whose:

- `Start Time` is `<= SystemTime::now`  `&&`
- `SystemTime::now <=` the `Start Time + Active Duration`

Each `Spoke` points to a list of `Jobs` that should trigger sometime during the spoke's
time bounds - asking a `Spoke` for ready jobs is called `Spoke walking`.

A Special spoke called the `Past Spoke` is maintained by the `Hub` to handle incoming jobs
whose `trigger time` is in the past. The `Hub` walks this spoke before walking any spoke at
the start of each rotation. This way, we maintain a total order on `trigger_at` times for all
Jobs that we accept responsibility for.

## Notes

- Use `ulimit -Sv 500000` 500mb mem limit for load testing mem leaks locally

## Running

- `goyaad -addr localhost:11300 -s localhost:8125` starts the goyaad server listening at 11300 on localhost and sends statsd metrics to 8125.
- Run `goyaad -help` for more information
- `SIGUSR1` will trigger a graceful shutdown by persisting current jobs to disk. To bootstrap with the jobs from disk, run with the `-r or --restore` flag (With the appropriate data dir set `-d or --dataDir`)

## Loadtesting & Benchmarks

It is important to make sure that the load generation logic is actually able to produce load at a rate higher can the actual workload.

For a load generation machine with the specs:

```text
(GCP) n1-highmem-8 (8 vCPUs, 52 GB memory)
Intel Broadwell
```

it was able to generate load with at a rate of about 40,000 RPS sent over the wire to a running Yaad instance over the network - this server was configured to short-circuit and drop the job withouth returning any errors.

The actual loadtest was able to enqueue jobs at a rate of 27,000 RPS with 10 concurrent client-connections and dequeue at a rate of 16,340 RPS for a machine with the following specs:

```text
(GCP) e2-standard-16 (16 vCPUs, 64 GB memory)
Intel Broadwell
```
