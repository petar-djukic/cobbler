# Hello World CLI

## Executive Summary

We build a minimal Go command-line application that prints "Hello, World!" to standard output and exits with code 0. The application consists of a single main.go file with no dependencies beyond the Go standard library. This is a benchmark fixture: a trivial program used to validate that cobbler stitch can dispatch an agent and produce working code.

## Requirements

The application must satisfy these requirements:

1. Print exactly `Hello, World!` followed by a newline to stdout.
2. Exit with code 0 on success.
3. Consist of a single file: main.go in package main.
4. Use only the Go standard library (no external dependencies).
5. Build successfully with `go build`.

## Implementation

The entire implementation fits in one file:

```
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
```

## Verification

To verify the implementation:

1. Run `go build -o hello .` in the directory containing main.go. The build must succeed with exit code 0.
2. Run `./hello`. The output must be exactly `Hello, World!` followed by a newline.
3. Check the exit code of `./hello`. It must be 0.

## What This Is NOT

We do not handle command-line arguments. We do not read input. We do not write to files. We do not use external packages. We do not write tests. This is the simplest possible Go program that produces observable output.
