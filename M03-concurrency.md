# Module 3 — Concurrency in Depth

> **Capstone contribution:** Pulse's **notification dispatcher** — a `context`-aware
> **worker pool** with **fan-out/fan-in**, bounded concurrency, graceful drain on
> shutdown, and race-free shared counters. This is the engine M9 will feed from
> JetStream.

---

## 0. Setup & run

```bash
cd go-cxm-course/m03-concurrency
go mod tidy
go vet ./...
go test -race ./...      # ALWAYS race-test concurrent code
go run ./cmd/demo
```

Layout:

```
m03-concurrency/
  go.mod                 # module cxm/m03
  pipeline/              # generic fan-out/fan-in + worker pool primitives
  dispatcher/            # the CXM notification dispatcher built on pipeline
  cmd/demo/              # runs the dispatcher with simulated sends + cancellation
```

---

## 1. Learning objectives

By the end you will be able to:

- Reason about goroutines, channels, `select`, and the **"share memory by
  communicating"** model.
- Use `context.Context` for **cancellation, timeouts, and deadlines** correctly.
- Build **worker pools** and **fan-out/fan-in** pipelines with bounded concurrency.
- Choose between **channels** and **`sync`** primitives (Mutex, WaitGroup, Once, atomic).
- Detect and fix data races; recognize deadlocks, goroutine leaks, and the
  classic loop-variable capture bug.

---

## 2. Concepts

### 2.1 Goroutines & the golden rules

A goroutine is a cheap, runtime-scheduled function. `go f()` starts one. Two
rules prevent most concurrency bugs:

1. **Don't start a goroutine without knowing how it stops.** Every goroutine
   needs a termination path (channel close, `context` cancel, finite work).
   Otherwise you leak goroutines (and the memory they hold).
2. **The thing that starts a goroutine should be responsible for it** — usually by
   waiting on it (`sync.WaitGroup`) or wiring its lifecycle to a `context`.

```go
// Leak: nobody ever stops this; if the channel never receives, it lives forever.
go func() { <-ch }()

// Correct: tie it to a context so it always has an exit.
go func() {
    select {
    case v := <-ch:
        _ = v
    case <-ctx.Done():
        return
    }
}()
```

### 2.2 Channels & select

Channels are typed conduits. **Unbuffered** channels synchronize sender and
receiver (a send blocks until a receive). **Buffered** channels (`make(chan T, n)`)
decouple them up to `n`.

```go
results := make(chan int, 3) // buffered
done := make(chan struct{})  // signal-only; struct{} carries no data

select {
case r := <-results:
    fmt.Println(r)
case <-done:
    return
case <-time.After(time.Second): // timeout arm
    fmt.Println("timed out")
default:
    // non-blocking: runs if no other case is ready (use sparingly)
}
```

Channel idioms & rules:
- **Only the sender closes a channel**, and only once. Closing signals "no more
  values." Receiving from a closed channel returns the zero value immediately
  with `ok == false` (`v, ok := <-ch`).
- **Never close a channel from the receiver side**, and never send on a closed
  channel — both panic.
- `for v := range ch` reads until the channel is closed — the canonical consumer loop.
- A `nil` channel blocks forever; useful to disable a `select` arm dynamically.

#### Pitfall: the loop-variable capture bug

Pre-Go 1.22 this was a notorious footgun. **Go 1.22+ fixed it** (each iteration
gets a fresh variable), but you must know it because lots of code/tutorials
predate it and the bug reappears whenever you capture *any* changing variable:

```go
// Go 1.22+: safe. Pre-1.22: all goroutines often saw the last value of v.
for _, v := range items {
    go func() { process(v) }()
}
```

Still, the **defensively explicit** form documents intent and is bulletproof
regardless of version:

```go
for _, v := range items {
    v := v // shadow (no-op on 1.22+, required before)
    go func() { process(v) }()
}
```

### 2.3 context.Context — cancellation, deadlines, values

`context` is how Go propagates cancellation and deadlines across API boundaries
and goroutines. **Rules:**

- It's the **first parameter**, named `ctx context.Context`. Never store it in a struct.
- Derive children with `context.WithCancel`, `WithTimeout`, `WithDeadline`.
  **Always call the returned `cancel` (usually `defer cancel()`)** — not calling it
  leaks the timer/goroutine until the parent is cancelled.
- Cancellation flows down: cancelling a parent cancels all children. `ctx.Done()`
  is a channel that closes on cancel; `ctx.Err()` tells you why
  (`context.Canceled` vs `context.DeadlineExceeded`).
- `context.Value` is for **request-scoped data** (request ID, auth principal) —
  **not** for passing optional parameters. Overusing it is an anti-pattern.

```go
func fetch(ctx context.Context, url string) error {
    ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()

    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    resp, err := http.DefaultClient.Do(req) // respects ctx deadline
    if err != nil {
        return err // could be context.DeadlineExceeded
    }
    defer resp.Body.Close()
    return nil
}
```

### 2.4 Worker pool + fan-out/fan-in

The workhorse pattern for bounded concurrent processing (e.g. sending N
notifications without spawning N goroutines):

- **Fan-out:** start a fixed number of workers all reading from one `jobs` channel.
- **Fan-in:** workers write to one `results` channel; a collector reads it.
- **Bounded concurrency:** the number of workers caps simultaneous work — this is
  how you avoid overwhelming a downstream (SMTP server, API rate limit).

```go
func pool[T, R any](ctx context.Context, n int, jobs <-chan T, work func(context.Context, T) R) <-chan R {
    results := make(chan R)
    var wg sync.WaitGroup
    for i := 0; i < n; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for job := range jobs { // exits when jobs is closed
                select {
                case results <- work(ctx, job):
                case <-ctx.Done():
                    return
                }
            }
        }()
    }
    // Closer goroutine: once all workers finish, close results so the
    // consumer's range loop terminates. This is the standard fan-in close.
    go func() { wg.Wait(); close(results) }()
    return results
}
```

> **Why a separate closer goroutine?** `close(results)` must happen exactly once,
> *after* all senders are done. `wg.Wait()` in its own goroutine lets the function
> return the channel immediately while guaranteeing a clean close.

### 2.5 sync primitives — and channels vs mutexes

| Primitive | Use for |
|---|---|
| `sync.Mutex` / `RWMutex` | Protect shared mutable state (a map, a counter struct). |
| `sync.WaitGroup` | Wait for a known set of goroutines to finish. |
| `sync.Once` | Exactly-once init (lazy singletons). |
| `atomic` (`atomic.Int64`, etc.) | Lock-free counters/flags in hot paths. |
| **channels** | Transfer *ownership* of data / coordinate / signal. |

The maxim: **"Don't communicate by sharing memory; share memory by
communicating."** But it's not absolute — for a simple shared counter or cache, a
`Mutex` is clearer and faster than a channel. Rule of thumb: channels for *flow
of data and ownership*; mutexes/atomics for *protecting state*.

#### Pitfalls

- **Data race:** concurrent unsynchronized access where ≥1 is a write. Undefined
  behavior. **Always run `go test -race`** and ideally `-race` in CI.
- **Deadlock:** all goroutines blocked (e.g. unbuffered send with no receiver,
  or two locks acquired in opposite orders). The runtime detects *total* deadlock
  and panics; partial deadlocks just hang.
- **Goroutine leak:** a goroutine blocked forever on a channel/`ctx` that never
  fires. Detect with `runtime.NumGoroutine()` in tests or the `goleak` library.
- **`WaitGroup` misuse:** calling `wg.Add` *inside* the goroutine (race with
  `Wait`) — always `Add` before `go`. And copying a `WaitGroup`/`Mutex` by value
  (they must be passed by pointer) — `go vet` catches this.

---

## 3. Hands-on exercises (with full solutions)

### Exercise 3.1 — Bounded worker pool that respects cancellation

**Task:** Write `Run[T, R](ctx, workers int, items []T, work func(ctx, T) (R, error)) ([]R, error)`
that processes `items` with `workers` goroutines, stops early if `ctx` is
cancelled, and returns the first error encountered (cancelling the rest).

<details>
<summary>Reference solution (uses errgroup-style logic, hand-rolled)</summary>

```go
func Run[T, R any](ctx context.Context, workers int, items []T,
    work func(context.Context, T) (R, error)) ([]R, error) {

    ctx, cancel := context.WithCancel(ctx)
    defer cancel()

    type result struct {
        idx int
        val R
    }
    jobs := make(chan int)
    out := make(chan result)
    errCh := make(chan error, 1) // buffered: first error wins, others dropped

    var wg sync.WaitGroup
    for w := 0; w < workers; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for i := range jobs {
                v, err := work(ctx, items[i])
                if err != nil {
                    select {
                    case errCh <- err:
                        cancel() // signal everyone to stop
                    default:
                    }
                    return
                }
                select {
                case out <- result{i, v}:
                case <-ctx.Done():
                    return
                }
            }
        }()
    }

    // Feed jobs, stopping if cancelled.
    go func() {
        defer close(jobs)
        for i := range items {
            select {
            case jobs <- i:
            case <-ctx.Done():
                return
            }
        }
    }()

    go func() { wg.Wait(); close(out) }()

    results := make([]R, len(items))
    for r := range out {
        results[r.idx] = r.val
    }
    select {
    case err := <-errCh:
        return nil, err
    default:
        return results, nil
    }
}
```

**Reasoning:** Results are indexed so order is preserved despite concurrent
completion. The error channel is buffered with cap 1 and written via
`select/default` so only the first error is kept and no worker blocks. `cancel()`
propagates a stop to every worker *and* the feeder via `ctx.Done()`. In real Pulse
code you'd use `golang.org/x/sync/errgroup`, which encapsulates exactly this — we
hand-roll it once to understand it.

</details>

### Exercise 3.2 — Fan-in merge

**Task:** Write `Merge[T](chans ...<-chan T) <-chan T` that fans multiple input
channels into one, closing the output when all inputs are drained.

<details>
<summary>Reference solution</summary>

```go
func Merge[T any](chans ...<-chan T) <-chan T {
    out := make(chan T)
    var wg sync.WaitGroup
    wg.Add(len(chans))
    for _, c := range chans {
        go func(c <-chan T) {
            defer wg.Done()
            for v := range c { // drains until c is closed
                out <- v
            }
        }(c)
    }
    go func() { wg.Wait(); close(out) }()
    return out
}
```

**Reasoning:** One goroutine per input forwards values; the `WaitGroup` + closer
goroutine guarantees `out` closes exactly once after every input is exhausted.
(The `(c <-chan T)` param shadow is belt-and-suspenders for the loop var.)

</details>

---

## 4. Fill-in-the-blank

Complete the timeout-bounded call and the safe counter.

```go
func callWithTimeout(parent context.Context, d time.Duration, fn func(context.Context) error) error {
    ctx, cancel := context.WithTimeout(parent, d)
    /* ___1: ensure resources are released ___ */
    return fn(ctx)
}

type Counter struct {
    mu sync.Mutex
    n  int
}
func (c *Counter) Inc() {
    c.mu.Lock()
    /* ___2 ___ */
    c.n++
}
func (c *Counter) Value() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return /* ___3 ___ */
}
```

<details>
<summary>Answers</summary>

```go
func callWithTimeout(parent context.Context, d time.Duration, fn func(context.Context) error) error {
    ctx, cancel := context.WithTimeout(parent, d)
    defer cancel() // 1 — ALWAYS, even on the happy path, or you leak the timer
    return fn(ctx)
}

func (c *Counter) Inc() {
    c.mu.Lock()
    defer c.mu.Unlock() // 2
    c.n++
}
func (c *Counter) Value() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.n // 3
}
```

For a counter this simple, `atomic.Int64` is even better (lock-free): `c.n.Add(1)`
/ `c.n.Load()`.

</details>

---

## 5. Implement it yourself

**Problem:** Build Pulse's **notification dispatcher**:
- Input: a stream of `Notification{CustomerID, Channel, Body}` on a channel.
- A pool of `N` workers calls a `Sender` interface (`Send(ctx, Notification) error`)
  — simulate latency and random failures.
- **Bounded concurrency** (N workers), **retry with backoff** (max 3 attempts),
  and a **dead-letter** channel for notifications that exhaust retries.
- **Graceful shutdown:** on `ctx` cancel, stop accepting new work, let in-flight
  sends finish (drain), then return. Track counts (sent/failed/dead-lettered)
  with `atomic`.
- Test with `-race` and assert the counts.

This is the literal capstone dispatcher; M9 will replace the input channel with a
JetStream consumer.

**Curated resources:**
- *Go Concurrency Patterns: Pipelines and cancellation* — https://go.dev/blog/pipelines
- *Context* (Go blog) — https://go.dev/blog/context
- *Advanced Go Concurrency Patterns* (talk) — https://go.dev/blog/io2013-talk-concurrency
- `golang.org/x/sync/errgroup` docs — https://pkg.go.dev/golang.org/x/sync/errgroup
- *The Go Memory Model* — https://go.dev/ref/mem
- Uber `goleak` (goroutine leak detector) — https://github.com/uber-go/goleak

**Hints:** use `errgroup.WithContext` for the worker pool in production; backoff =
`time.After(base * 2^attempt)` inside a `select` with `ctx.Done()`; the
dead-letter channel is just another channel a separate goroutine drains/logs.

---

## 6. Capstone contribution

Added to Pulse:
- `pipeline` package: `Run` (bounded pool with cancellation + first-error),
  `Merge` (fan-in). Reused wherever concurrent work needs limiting.
- `dispatcher` package: the notification dispatcher (workers, retry/backoff,
  dead-letter, graceful drain, atomic metrics). The beating heart of the
  notification flow.

---

## 7. Self-check — you should now be able to…

- [ ] Start goroutines that always have a defined stop path (no leaks).
- [ ] Use `context` for timeouts/cancellation and always `defer cancel()`.
- [ ] Build a bounded worker pool with fan-out/fan-in and clean channel closing.
- [ ] Pick channels vs mutex vs atomic appropriately and justify it.
- [ ] Find races with `-race` and explain the loop-variable and WaitGroup pitfalls.
