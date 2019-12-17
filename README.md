# A simple beanstalkd-like job queue implementation in go

[![Go Report Card](https://goreportcard.com/badge/github.com/urjitbhatia/yaad)](https://goreportcard.com/report/github.com/urjitbhatia/yaad)
[![Build Status](https://travis-ci.com/urjitbhatia/yaad.svg?branch=master)](https://travis-ci.com/urjitbhatia/yaad)
[![GoDoc](https://godoc.org/github.com/urjitbhatia/yaad/pkg?status.svg)](https://godoc.org/github.com/urjitbhatia/yaad/pkg)

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

- `yaad -addr localhost:11300 -s localhost:8125` starts the yaad server listening at 11300 on localhost and sends statsd metrics to 8125.
- Run `yaad -help` for more information
- `SIGUSR1` will trigger a graceful shutdown by persisting current jobs to disk. To bootstrap with the jobs from disk, run with the `-r or --restore` flag (With the appropriate data dir set `-d or --dataDir`)
