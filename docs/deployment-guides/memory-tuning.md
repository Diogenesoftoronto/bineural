# Memory Tuning (Railway / single-instance deployment)

This deployment runs the gateway as one container behind Railway. The upstream
defaults are tuned for a large, high-throughput cluster and pre-allocate far more
memory than a personal/small deployment needs. The settings below cut idle RAM
substantially with no meaningful loss for low-to-moderate traffic.

## What was changed

### 1. Go runtime is auto-capped to the container memory limit
`transports/docker-entrypoint.sh` now sets, at startup:

- **`GOMEMLIMIT`** to ~90% of the container's cgroup memory limit (read from
  `/sys/fs/cgroup/memory.max` on cgroup v2, falling back to v1). This is the single
  biggest lever: without it, Go's garbage collector lets the live heap grow toward
  ~2× before collecting. With it, the GC reclaims aggressively as the heap nears the
  limit, so resident memory tracks the container size instead of overshooting it.
- **`GOGC=75`** (default 100). Slightly more frequent GC for a smaller resident heap;
  costs a little CPU.

Both respect any value you set yourself — set `GOMEMLIMIT` or `GOGC` as a Railway
service variable to override the auto-tuning.

### 2. Smaller request/response object pool
`config.json` → `client.initial_pool_size`: **5000 → 100**. Upstream pre-allocates
5000 pooled request/response objects at boot. The pool still grows on demand under
load; 100 just sets a much smaller floor.

### 3. Smaller per-provider worker pools and buffers
`config.json` → each provider's `concurrency_and_buffer_size`: **{1000, 5000} → {50, 100}**.

Each configured provider spins up `concurrency` worker goroutines and a channel
buffer of `buffer_size` in-flight request slots. With 7 providers, the upstream
defaults meant ~7,000 idle goroutines and buffers for ~35,000 queued requests. The
reduced values give ~350 goroutines and ~700 slots — still ample for personal use.

## How config.json changes take effect

`config_store` (SQLite) is enabled, but **`config.json` wins on boot**: the loader
hashes the file config and, on a mismatch with the stored config, syncs the file in
(`"file config takes precedence"`). So editing `config.json` and redeploying is the
correct, durable way to change these — no need to edit the DB or the UI.

> Note: values you change later in the web UI are persisted in the SQLite store and
> kept as long as `config.json` is unchanged. If you edit `config.json` again, the
> file values overwrite the UI changes for the keys present in the file.

## Tuning further

| Symptom | Lever | Direction |
|---|---|---|
| Still over budget at idle | `GOGC` | Lower (e.g. 50). Trades CPU for RAM. |
| OOM / restarts under load | container size, then `GOMEMLIMIT` | Raise the Railway plan; auto-tune follows it. |
| Requests queueing/blocked | provider `concurrency` / `buffer_size` | Raise for hot providers only. |
| High allocation churn in logs | `client.initial_pool_size` | Raise toward 500–1000. |

### Other knobs worth a look if RAM is still tight
- **Drop unused providers** from `config.json` — each one you don't call still
  allocates its worker pool and buffer.
- **`client.drop_excess_requests`** (currently `false`): set `true` to shed load
  instead of buffering when a provider queue is full — bounds memory under bursts.
- **Logs retention**: `logs_store` is SQLite on the `/app/data` volume; it grows with
  traffic. It's disk, not RAM, but prune it if the volume fills.

## Verifying after deploy

Check the startup logs for the auto-tune line:

```
Auto-set GOMEMLIMIT=<n>B (90% of cgroup limit <n> bytes)
```

If it's absent, the container had no enforced cgroup memory limit (so nothing to cap
to) — set `GOMEMLIMIT` explicitly as a Railway variable in that case.
