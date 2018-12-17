# A simple beanstalkd-like job queue implementation in go

[![Go Report Card](https://goreportcard.com/badge/github.com/urjitbhatia/goyaad)](https://goreportcard.com/report/github.com/urjitbhatia/goyaad)
[![Build Status](https://travis-ci.com/urjitbhatia/goyaad.svg?branch=master)](https://travis-ci.com/urjitbhatia/goyaad)

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
