# Loadtesting & Benchmarks

It is important to make sure that the load generation logic is actually able to produce load at a rate higher can the actual workload.

For a load generation machine with the specs:

```text
(GCP) n1-highmem-8 (8 vCPUs, 52 GB memory)
Intel Broadwell
```

it was able to generate load with at a rate of about 40,000 RPS sent over the wire to a running Chronomq instance over the network - this server was configured to short-circuit and drop the job withouth returning any errors.

The actual loadtest was able to enqueue jobs at a rate of 27,000 RPS with 10 concurrent client-connections and dequeue at a rate of 16,340 RPS for a machine with the following specs:

```text
(GCP) e2-standard-16 (16 vCPUs, 64 GB memory)
Intel Broadwell
```
