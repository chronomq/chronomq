###A simple beanstalkd-like job queue implementation in go

##### Goals

- Start with a in-memory data structure
- Support beanstalkd wire protocol?
- Make all Job operations zero-copy
- Allow a disk mapped mode (fully mapped, lazily mapped, full in-mem)

##### Architecture

The most fitting architectural analogy for this is to imagine a bicycle wheel.
This wheel spins only in the forward direction (arrow-of-time). At any instant, a `Spoke`
of the wheel is active. The central `Hub` connects all `Spokes` and is responsible for
`rotating` through them.

An `active Spoke` is a `Spoke` whose:

- `Start Time` is `<= SystemTime::now`  `&&`
- `SystemTime::now <=` the `Start Time + Active Duration`

No two spokes can have overlapping intervals of time responsibility. [TODO: Add unit test]

Each `Spoke` points to a list of `Jobs` that should trigger sometime during the spoke's
time bounds - asking a `Spoke` for ready jobs is called `Spoke walking`.

A Special spoke called the `Past Spoke` is maintained by the `Hub` to handle incoming jobs
whose `trigger time` is in the past. The `Hub` walks this spoke before walking any spoke at
the start of each rotation. This way, we maintain a total order on `trigger_at` times for all
Jobs that we accept responsibility for.

- `ulimit -Sv 500000` 500mb mem limit for load testing mem leaks locally
