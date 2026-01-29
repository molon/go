---
name: go-errstack-patch
description: |
  Skill for patching Go standard library to add stack traces to errors.
  Use when: modifying Go source to add stack traces to errors.New, fmt.Errorf, errors.Join;
  implementing %+v formatting for errors with stack traces; building patched Go from source.
---

# Go Error Stack Trace Patch

Patches Go's `errors` and `fmt` packages to capture stack traces automatically.

## Reference Implementation

Full implementation: https://github.com/golang/go/compare/go1.25.6...molon:go:go1.25.6-errstack0.0.1?expand=1

**Note**: The reference implementation includes a `.opencode/skills` directory which should be ignored. Only the Go source code modifications, Dockerfile, GitHub Actions workflow, and VS Code settings are relevant.

## New Files (in commit)

These files are completely new and can be copied directly from the commit:

| File | Description |
|------|-------------|
| `.vscode/settings.json` | VS Code settings to disable gopls when editing Go source |
| `Dockerfile` | Multi-stage build for patched Go |
| `src/errors/stack.go` | Stack types (`StackFrame`, `StackTrace`, `Stack`) and capture logic |
| `src/errors/stack_test.go` | Test suite for stack trace functionality |

### api/go1.25.txt (append to existing file)

Add the following API declarations (after `debug/elf`, before `go/ast`) to pass the API check.
Note: `#0` is used as a placeholder for proposal approval number since this is a custom patch:

```
pkg errors, func CaptureStack(int) *Stack #0
pkg errors, method (*Stack) HasStack() bool #0
pkg errors, method (*Stack) RemoveStackFrames(int) #0
pkg errors, method (*Stack) StackTrace() StackTrace #0
pkg errors, method (StackFrame) PC() uintptr #0
pkg errors, method (StackTrace) Format(func(string)) #0
pkg errors, type Stack struct #0
pkg errors, type StackFrame uintptr #0
pkg errors, type StackTrace []StackFrame #0
pkg errors, type StackTracer interface { HasStack, StackTrace } #0
pkg errors, type StackTracer interface, HasStack() bool #0
pkg errors, type StackTracer interface, StackTrace() StackTrace #0
pkg errors, var FormatStackTrace func(StackTrace, func(string)) #0
pkg errors, var ShouldCaptureStack func(error) bool #0
```

## Modified Standard Library Files

These files require careful modification to existing Go source:

### src/errors/errors.go

- Add `*Stack` embedding to `errorString` struct
- Modify `New()` to call `CaptureStack(1)`

### src/errors/join.go

- Add `*Stack` embedding to `joinError` struct
- Modify `Join()` to conditionally capture stack via `ShouldCaptureStack()`

### src/fmt/errors.go

- Add `import "errors"` 
- Add `*errors.Stack` embedding to `wrapError` and `wrapErrors` structs
- Modify `Errorf()`:
  - For no `%w`: call `errors.New()` then `RemoveStackFrames(1)`
  - For single `%w`: conditionally capture stack via `errors.ShouldCaptureStack()`
  - For multiple `%w`: conditionally capture stack via `errors.ShouldCaptureStack()`

### src/fmt/print.go

- Add `import "errors"`
- In `handleMethods()`, add `%+s` error formatting branch before default error handling
- Add new functions at end of file:
  - `fmtErrorWithStack(err error)`
  - `fmtErrorWithStackImpl(err error, depth int)`
  - `fmtStackTrace(frames errors.StackTrace)`
  - `fmtStackTraceWithIndent(frames errors.StackTrace, indent string)`

### src/reflect/deepequal.go

- Add `import "errors"`
- In `deepValueEqual()` Struct case, skip **embedded** (anonymous) `*errors.Stack` fields during comparison
- Named fields of type `*errors.Stack` are still compared normally
- This ensures `reflect.DeepEqual` ignores stack trace differences in error types, so existing tests using `reflect.DeepEqual` for error comparison continue to work

### test/noinit.go

- Remove `import "errors"` since `errors` package now has init funcs due to `ShouldCaptureStack` and `FormatStackTrace` variables
- Replace `errors.New("mine")` with a simple local error type to avoid importing `errors` package
- This test verifies that certain packages can be initialized at link time without runtime init functions

## Dockerfile Bootstrap Version

Go requires a recent version (typically N-1) to build from source. Use the latest patch version of the bootstrap Go:

| Target Go Version | Bootstrap Image |
|-------------------|-----------------|
| Go 1.25.x | `golang:1.24.12-bookworm` (or latest 1.24.x) |
| Go 1.24.x | `golang:1.23.x-bookworm` (use latest 1.23.x) |
| Go 1.23.x | `golang:1.22.x-bookworm` (use latest 1.22.x) |

## Build

```bash
export GOROOT_BOOTSTRAP=/usr/local/go  # Go 1.24+ for building Go 1.25.x
cd src && ./make.bash
../bin/go test -v -run "Test.*Stack" errors
```

## Docker Image (GitHub Container Registry)

The repository includes a GitHub Actions workflow (`.github/workflows/docker-publish.yml`) that automatically builds and pushes the Docker image to GitHub Container Registry (ghcr.io).

### Trigger

Only triggered when pushing a tag in format `go<version>-errstack<patch>`, e.g., `go1.25.6-errstack0.0.1`

The workflow will:
1. Validate tag format matches `go<version>-errstack<patch>` pattern
2. Run comprehensive tests (all Go standard library packages)
3. Build and push the image only if tests pass

### Image Tags

For tag `go1.25.6-errstack0.0.1`, the following images are published:
- `ghcr.io/molon/go-errstack:1.25.6-0.0.1-alpine` (full version)
- `ghcr.io/molon/go-errstack:1.25.6-alpine` (latest patch for this Go version)
- `ghcr.io/molon/go-errstack:latest`

### Usage in Production

In your production Dockerfiles, simply change:
```dockerfile
# Before
FROM golang:1.25-alpine

# After  
FROM ghcr.io/molon/go-errstack:1.25.6-alpine
```

### Manual Build (optional)

```bash
docker build -t go-errstack .
docker run --rm go-errstack go version
```

## Output Format

```
error message
function.name
    /path/to/file.go:123
~~~
  wrapped error message
  function.name
      /path/to/file.go:456
  ---
  another wrapped error
  function.name
      /path/to/file.go:789
```

- `~~~` separates parent from children
- `---` separates sibling errors in `errors.Join` or multi-`%w`
- Indentation increases with depth

## Key Design Decisions

1. **Conditional capture**: Only capture stack if error tree has no existing stack
2. **Pointer embedding**: Use `*Stack` to allow nil (no stack captured)
3. **RemoveStackFrames**: Adjust stack when `fmt.Errorf` delegates to `errors.New`
4. **No strconv**: Use local `itoa` to avoid import cycle
5. **ShouldCaptureStack as var**: Defined as a variable instead of function to allow custom logic override if needed
6. **FormatStackTrace as var**: Stack trace formatter is a variable, allowing users to customize the output format
7. **%+s for stack trace**: Use `fmt.Sprintf("%+s", err)` to output error with stack trace, keeping `%+v` behavior unchanged for backward compatibility
8. **DeepEqual ignores embedded Stack**: `reflect.DeepEqual` skips **embedded** (anonymous) `*errors.Stack` fields, so existing tests using `reflect.DeepEqual` for error comparison continue to work. Named fields are still compared.

## GitHub Actions Workflow

The workflow has two modes:

### Branch Push (`go*` branches)
- Runs quick tests only (test stage)
- Does not build or push images
- Useful for development validation

### Tag Push (`go*` tags)
- Validates tag format: `go<version>-errstack<patch>`
- Runs full standard library tests
- Builds and pushes Docker images to ghcr.io
- Only proceeds if all tests pass
