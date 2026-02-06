# Use Case: Hello World Benchmark

## Summary

A developer invokes cobbler stitch against a benchmark fixture directory containing a spec document for a Hello World Go CLI. Cobbler reads the spec, creates a task in the cupboard, builds a prompt from templates, dispatches an agent, and the agent produces a working main.go that prints "Hello, World!" to stdout.

## Actor and Trigger

**Actor**: Developer running the benchmark suite or manually testing stitch.

**Trigger**: Developer invokes `cobbler stitch` against a benchmark fixture directory that contains a specification document (e.g., `spec.md`) describing a Hello World Go CLI.

## Flow

1. Developer runs `cobbler stitch --fixture benchmarks/hello-world` (or equivalent invocation pointing to the benchmark directory).
2. Cobbler reads the spec document from the fixture directory. The spec describes the required output: a Go CLI that prints "Hello, World!" to stdout.
3. Cobbler creates a crumb in the cupboard representing the code task. The crumb includes the spec content, task type (coding), and acceptance criteria (go build passes, output matches "Hello, World!").
4. Cobbler claims the crumb by setting its state to `taken`.
5. The prompt builder constructs a prompt from templates. The prompt includes the spec, project context, and instructions for producing a single main.go file.
6. Cobbler dispatches the prompt to the agent via the Agent interface. The agent receives the prompt and produces main.go content.
7. The agent writes main.go to the fixture output directory (or worktree).
8. Cobbler runs quality gates: `go build` on main.go and verifies the binary produces the expected output.
9. On success, cobbler updates the crumb state to `completed`. On failure, the crumb state becomes `failed` with error details.
10. The benchmark runner reports pass or fail based on the crumb state and gate results.

## Architecture Touchpoints

| Component | Interface/Protocol | Role in this use case |
|-----------|-------------------|----------------------|
| Cupboard | GetTable("crumbs").Set(), Get() | Stores the code task crumb; tracks state transitions |
| Stitch executor | Executor.Execute() | Claims crumb, orchestrates prompt-agent-gate workflow |
| Prompt builder | internal/prompt templates | Constructs prompt from spec and templates |
| Agent interface | Agent.Run() | Sends prompt to LLM, receives generated code |
| Quality gates | go build, output verification | Validates generated code compiles and runs correctly |

## Success / Demo Criteria

1. Run the benchmark command: `cobbler stitch --fixture benchmarks/hello-world`
2. Observe: cobbler reads spec.md, creates and claims a crumb
3. Observe: agent produces main.go in the output directory
4. Run `go build -o hello main.go` in the output directory; build succeeds with exit code 0
5. Run `./hello`; stdout contains exactly "Hello, World!" (or "Hello, World!\n")
6. The benchmark runner reports: PASS
7. The crumb in the cupboard has state `completed`

Checkable outcomes:

- `go build` exit code is 0
- Binary output matches expected string
- Crumb state is `completed`
- Benchmark report shows pass

## Out of Scope

We do not cover:

- Multiple-file applications (main.go only)
- Test file generation (no main_test.go)
- Refactoring loops or iterative fixes
- Agent retry on failure
- Git worktree management (uses simple output directory)
- Complex spec parsing or multi-step prompts

## Dependencies

- rel01.0-uc001-cupboard-connection: Cupboard must be connectable via Go module
- rel01.0-uc003-agent-loop: Agent interface and prompt templates must be functional

## Risks / Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Agent produces incorrect code | Benchmark fails | Quality gates catch failure; spec is simple enough to succeed reliably |
| go build not available in environment | Benchmark cannot validate | Document prerequisite: Go toolchain installed |
| Output format varies ("Hello World" vs "Hello, World!") | False failures | Spec explicitly states exact output string; agent prompt includes verbatim requirement |
