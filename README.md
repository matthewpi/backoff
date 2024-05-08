# Backoff

[![Godoc Reference][pkg.go.dev_img]][pkg.go.dev]
[![Pipeline Status][pipeline_img  ]][pipeline  ]

Go package that provides a context-aware exponential backoff.

[pkg.go.dev]:     https://pkg.go.dev/github.com/matthewpi/backoff
[pkg.go.dev_img]: https://img.shields.io/badge/%E2%80%8B-reference-007d9c?logo=go&logoColor=white&style=flat-square

[pipeline]:     https://github.com/matthewpi/backoff/actions/workflows/test.yml
[pipeline_img]: https://img.shields.io/github/actions/workflow/status/matthewpi/backoff/ci.yaml?style=flat-square&label=tests

## Usage

```go
b := backoff.New(3, 2, 1*time.Second, 5*time.Second)

// Avoid generating a new context every time Next is called.
ctx := context.Background()
for b.Next(ctx) {
  // Do something.
  //
  // break if successful, continue on failure
}
```

ref; [`example_test.go`](./example_test.go)

## Installation

```bash
go get github.com/matthewpi/backoff
```
