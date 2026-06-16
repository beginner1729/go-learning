# Module 0 — How to Build a Small Go Project

> Before the language depth (M1+), get fluent with the **mechanics**: what a
> module is, how packages and imports fit together, and the handful of `go`
> commands you'll type hundreds of times. Everything later assumes this.

This module is **guided, not write-from-scratch**. Three small projects are
already built and runnable in `m00-getting-started/`. You'll run them and walk
through *every intermediate step* so the toolchain stops being magic.

---

## 0. Setup & run

```sh
go version          # expect go1.24.x (1.22+ is fine for this module)
cd m00-getting-started
```

That's it — no Docker, no dependencies. Everything here is standard library.

---

## 1. Learning objectives

By the end you can, from a blank directory:

- [ ] Explain what `go.mod`, a **module path**, and an **import path** are.
- [ ] Tell a **library package** from a **`main` (executable) package**.
- [ ] Run, build, and test code with `go run`, `go build`, `go test`.
- [ ] Split a program into packages and import across them within one module.
- [ ] Use the `cmd/<name>/` convention for executables.
- [ ] Read `go` tool output: `[no test files]`, `ok`, exit codes, `go vet`.

---

## 2. Concepts

### 2.1 Module vs package vs command

- A **module** is a unit of versioned code with a `go.mod` at its root. `go.mod`
  declares the **module path** (e.g. `cxm/m00/greeter`) and the Go version.
- A **package** is a directory of `.go` files that share `package <name>` on
  their first line. One directory = one package.
- A package named `main` with a `func main()` is a **command** (it compiles to
  an executable). Any other package name is a **library** (imported, never run).

### 2.2 Import paths are module path + directory

There is no central registry of names. An import path is literally the module
path joined with the directory inside the module:

```
module:  cxm/m00/greeter            (from go.mod)
dir:     greeter/                    (the package directory)
import:  "cxm/m00/greeter/greeter"   (what you write in `import (...)`)
```

So `import "cxm/m00/greeter/greeter"` then call `greeter.Greet(...)` — the
identifier you use is the **package name** (`greeter`), not the full path.

### 2.3 Exported vs unexported

A capitalized identifier (`Greet`, `Summary`, `ErrEmpty`) is **exported** — visible
to other packages. A lowercase one is private to its package. That single rule
*is* Go's access control.

### 2.4 The commands you'll actually use

| Command | What it does |
|---|---|
| `go run .` / `go run ./cmd/x` | Compile + run, no binary left behind. |
| `go build ./...` | Compile everything; report errors. `./...` = this dir and below. |
| `go build -o bin/x ./cmd/x` | Compile to a named binary you can ship/run. |
| `go test ./...` | Find and run every `TestXxx` in every package. |
| `go vet ./...` | Static checks for likely bugs (printf args, etc.). |
| `gofmt -l .` / `go fmt ./...` | Format code; `-l` just lists unformatted files. |
| `go mod init <path>` | Create a new module (`go.mod`). |
| `go mod tidy` | Add missing / drop unused dependencies. |

---

## 3. Guided exercise 0.1 — `hello`: one module, one file

**Code:** `m00-getting-started/01-hello/`. The whole project is two files:

```
01-hello/
├── go.mod      module cxm/m00/hello
└── main.go     package main + func main
```

Walk through it:

**Step 1 — look at the module file.**
```sh
cd 01-hello
cat go.mod
```
```
module cxm/m00/hello

go 1.24
```
This is what `go mod init cxm/m00/hello` would have generated. It marks this
directory as a module root.

**Step 2 — run it (no binary produced).**
```sh
go run .
```
```
hello, Pulse
```
`.` means "the main package in the current directory." `go run` compiles to a
temp location, runs it, throws the binary away.

**Step 3 — build a real binary, then run it.**
```sh
go build -o hello .
./hello
```
```
hello, Pulse
```
Now there's a `hello` executable in the directory. `go build` without `-o`
would name it after the directory.

**Step 4 — clean up.**
```sh
rm hello
```

> **What you just saw:** `package main` + `func main()` is the entry point;
> `go run` is for iterating, `go build` is for shipping.

---

## 4. Guided exercise 0.2 — `greeter`: packages, imports, a flag, a test

**Code:** `m00-getting-started/02-greeter/`.

```
02-greeter/
├── go.mod                  module cxm/m00/greeter
├── greeter/
│   ├── greeter.go          package greeter  (library: func Greet)
│   └── greeter_test.go     package greeter  (the test)
└── cmd/
    └── greet/
        └── main.go         package main     (imports greeter, reads -name)
```

Two packages now: a **library** (`greeter`) and a **command** (`cmd/greet`)
that imports it.

**Step 1 — try to run the library. It can't run.**
```sh
cd 02-greeter
go run ./greeter
```
```
go run: cannot run non-main package in directory ./greeter
```
Expected — `greeter` is `package greeter`, not `main`. Libraries are imported,
not executed.

**Step 2 — run the command instead.**
```sh
go run ./cmd/greet
```
```
hello, world
```
`./cmd/greet` points `go run` at the directory holding `package main`.

**Step 3 — pass the flag.**
```sh
go run ./cmd/greet -name Ada
```
```
hello, Ada
```
`flag.String("name", "", ...)` defined the `-name` flag; `flag.Parse()` filled
it from the command line.

**Step 4 — run the test.**
```sh
go test ./...
```
```
?   cxm/m00/greeter/cmd/greet   [no test files]
ok  cxm/m00/greeter/greeter     0.4s
```
Read this: the command package has no tests (`?`), the library passed (`ok`).
`./...` tested both packages in one go.

**Step 5 — see a failing test (then undo).** Temporarily change `want` in
`greeter/greeter_test.go` (e.g. `"hello, Ada"` → `"HELLO"`), rerun `go test ./...`:
```
--- FAIL: TestGreet (0.00s)
    greeter_test.go: Greet("Ada") = "hello, Ada", want "HELLO"
FAIL
```
That's what a red test looks like. Revert your change.

> **What you just saw:** import path = module + dir; a library can't `go run`;
> `cmd/<name>/` holds executables; `go test ./...` reports per package.

---

## 5. Guided exercise 0.3 — `stats`: logic, errors, a table test, a CLI

**Code:** `m00-getting-started/03-stats/`. Same shape, with a value+error API
and a CLI that parses arguments.

```
03-stats/
├── go.mod
├── stats/
│   ├── stats.go        Summarize([]int) (Summary, error); sentinel ErrEmpty
│   └── stats_test.go   table-driven test + an error-case test
└── cmd/stats/main.go   parses ints from os.Args, prints the summary
```

**Step 1 — run the tests (note the table-driven subtests).**
```sh
cd 03-stats
go test -v ./stats
```
```
=== RUN   TestSummarize
=== RUN   TestSummarize/single
=== RUN   TestSummarize/many
=== RUN   TestSummarize/negatives
--- PASS: TestSummarize (0.00s)
    --- PASS: TestSummarize/single (0.00s)
    ...
=== RUN   TestSummarize_Empty
--- PASS: TestSummarize_Empty (0.00s)
PASS
ok  cxm/m00/stats/stats
```
`-v` shows each subtest by name (`t.Run` created them). One slice of cases, one
loop — the idiom you'll use everywhere.

**Step 2 — run the CLI on good input.**
```sh
go run ./cmd/stats 3 1 4 1 5
```
```
count=5 sum=14 min=1 max=5
```

**Step 3 — watch error handling + exit code.**
```sh
go run ./cmd/stats x
echo "exit: $?"
```
```
skipping "x": not an integer
error: stats: no values
exit: 1
```
The bad arg is skipped (printed to **stderr**), no valid numbers remain, so
`Summarize` returns `ErrEmpty`, `main` prints it and calls `os.Exit(1)`. A
non-zero exit code is how a CLI tells scripts/CI it failed.

**Step 4 — build a shippable binary.**
```sh
go build -o bin/stats ./cmd/stats
./bin/stats 10 20 30
rm -rf bin
```
```
count=3 sum=60 min=10 max=30
```

**Step 5 — run the housekeeping tools.**
```sh
go vet ./...
gofmt -l .
```
Both should print **nothing** — `go vet` found no suspicious code, `gofmt -l`
listed no unformatted files. Silence is success.

> **What you just saw:** the Go `(value, error)` pattern, a sentinel error
> (`errors.Is`), table-driven tests with subtests, reading args, exit codes, and
> the `vet`/`fmt` quality tools.

---

## 6. Command cheat sheet

```sh
go mod init cxm/m00/myproj   # start a module in the current dir
go run .                     # run main in this dir
go run ./cmd/app             # run main in ./cmd/app
go build ./...               # compile everything, report errors
go build -o bin/app ./cmd/app
go test ./...                # run all tests
go test -v -run TestGreet ./greeter   # one test, verbose
go vet ./...                 # static analysis
go fmt ./...                 # format in place
go mod tidy                  # sync dependencies with imports
```

---

## 7. Now you try (small, safe tweaks)

These keep everything compiling — make the change, then re-run the command.

1. **greeter:** add a `-shout` bool flag to `cmd/greet` that upper-cases the
   greeting (`strings.ToUpper`). Add a test case to `greeter_test.go` for a new
   `GreetShout` or pass-through behavior. Verify with
   `go test ./... && go run ./cmd/greet -name ada -shout`.
2. **stats:** add a `Mean float64` field to `Summary` (sum/count). Update the
   `want` values in the table test, run `go test -v ./stats`, then
   `go run ./cmd/stats 2 4 6` and confirm the mean prints.
3. **from scratch:** in a fresh temp dir run `go mod init demo/scratch`, write a
   `main.go` that prints today's date (`time.Now()`), and `go run .`. You've now
   created a module end to end.

If a build breaks, read the **first** error only (later ones are usually
fallout), fix it, re-run. That loop — change → `go test ./...` → read first
error → fix — is the whole job.

---

## 8. Self-check — you should now be able to…

- [ ] Say what `go.mod` declares and what a module path is.
- [ ] Explain why `go run ./greeter` fails but `go run ./cmd/greet` works.
- [ ] Predict the import path for a package given the module path and its dir.
- [ ] Read `go test ./...` output (`?` vs `ok` vs `FAIL`) and a non-zero exit.
- [ ] Produce a binary with `go build -o` and run it.
- [ ] Reach for `go vet` and `gofmt` and know that silence means clean.

Next: **[M01 — Interfaces, Composition & Idiomatic Errors](M01-interfaces-and-errors.md)**,
where you stop running pre-built code and start writing it.
