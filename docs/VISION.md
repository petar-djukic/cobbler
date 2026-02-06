# Cobbler Vision

## Executive Summary

Cobbler orchestrates AI coding agents through four capabilities drawn from autogenic systems: self-orienting, self-programming, self-reflecting, and self-architecting. We provide a CLI tool that connects to a crumbs cupboard for work item storage and coordinates agents to plan and execute software development tasks. Cobbler replaces prototype bash scripts with a proper Go implementation that has direct cupboard access via the crumbs module.

We are not a task tracker, IDE plugin, or CI system. We coordinate agents to do work; we do not store the work itself or integrate with editors.

## Introduction

### Research Context

Autogenic systems can orient themselves to their environment, program their own behavior, reflect on their outputs, and architect their own structure. These four self-* capabilities map naturally to software development: assess the project state, execute work, evaluate results, and propose design changes. We apply this framing to AI agent coordination.

### The Problem

Manual orchestration of AI coding agents is slow and error-prone. A human must read the project state, formulate a prompt, invoke the agent, review output, and decide what to do next. Prototype bash scripts (make-work.sh and do-work.sh) automated parts of this loop but reached their limits: they lack direct access to work item storage, they build prompts through string concatenation, and they have no structured agent loop.

These scripts query beads (bd) for issues, invoke Claude via CLI, and parse output. They work for simple cases but cannot grow. make-work.sh creates work by prompting Claude to analyze the project and output JSON for new issues. do-work.sh picks a task, creates a git worktree, runs Claude, merges the branch, and closes the task. Both scripts treat the cupboard as an external service called through shell commands rather than as a directly accessible interface.

### What Cobbler Does

Cobbler graduates these prototypes into a real tool. We import the crumbs Go module and call cupboard methods directly: Cupboard.GetTable("crumbs").Get(id), Cupboard.GetTable("crumbs").Set(id, data). We use structured prompt templates rather than heredocs. We implement a proper agent loop that can measure, stitch, inspect, mend, and pattern.

The command set uses shoemaking metaphors mapped to the four self-* capabilities.

| Table 1 Command Mapping |
|-------------------------|

| Self-capability | Command | Shoemaking metaphor | What it does |
|-----------------|---------|---------------------|--------------|
| Self-orienting | cobbler measure | Measure the foot | Assess project state, propose tasks |
| Self-programming | cobbler stitch | Elves sew the shoes | Execute work via AI agents |
| Self-reflecting | cobbler inspect | Check the seams | Evaluate output quality |
| Self-reflecting | cobbler mend | Repair the sole | Fix issues found by inspect |
| Self-architecting | cobbler pattern | Draft the template | Propose design and structural changes |

We measure the work to understand what needs doing. We stitch to execute tasks through agents. We inspect and mend to evaluate and fix. We pattern to propose architectural changes. Each command maps to a self-* capability that autogenic systems exhibit.

## Why This Project

Cobbler fills the gap between task storage (crumbs) and agent execution. The crumbs cupboard stores work items as first-class entities with properties, trails, and stashes. Agents need a coordinator that can read those items, formulate prompts, invoke execution, and update state. Cobbler is that coordinator.

| Table 2 Relationship to Other Components |
|------------------------------------------|

| Component | Role | Cobbler's relationship |
|-----------|------|------------------------|
| Crumbs | Work item storage (cupboard) | Imports crumbs module; calls cupboard interface directly |
| Beads (bd) | CLI for crumbs | Cobbler replaces bd usage with direct Go calls |
| make-work.sh | Prototype work creation | Cobbler measure replaces this script |
| do-work.sh | Prototype work execution | Cobbler stitch replaces this script |
| Claude Code | AI agent runtime | Cobbler invokes agents; does not replace the runtime |

We leverage the existing cupboard interfaces defined in crumbs. The Cupboard interface has GetTable(name) returning a table reference, and each table has Get and Set methods. The attach/detach methods connect to backends with optional JSON configuration. Cobbler calls these interfaces rather than shelling out to bd commands.

## Planning and Implementation

### Success Criteria

We measure success along three dimensions.

Autonomous execution: agents complete documentation and code tasks via cobbler stitch without human intervention beyond approving the initial measure output.

Cupboard integration: cobbler accesses crumbs directly through Go module import, never through shell commands to bd.

Graduated prototypes: make-work.sh functionality moves to cobbler measure; do-work.sh functionality moves to cobbler stitch. The scripts become obsolete.

### What Done Looks Like

A developer runs cobbler measure to assess project state. Cobbler reads the cupboard, analyzes existing work, and proposes new tasks. The developer reviews and approves. Cobbler writes approved tasks back to the cupboard.

The developer runs cobbler stitch. Cobbler picks a task, formulates a prompt from templates, invokes an AI agent, captures output, and updates the cupboard. For documentation tasks, the agent writes markdown. For code tasks, the agent writes implementation. Cobbler can run multiple tasks in sequence.

When issues arise, the developer runs cobbler inspect to evaluate recent work. If defects exist, cobbler mend attempts fixes. For larger structural changes, cobbler pattern proposes architectural updates.

The prototype scripts (make-work.sh, do-work.sh) are no longer used. All agent orchestration flows through cobbler.

### Implementation Phases

| Table 3 Implementation Phases |
|-------------------------------|

| Phase | Focus | Deliverables |
|-------|-------|--------------|
| 01.0 | Stitch for documentation | cobbler stitch executes documentation tasks; cupboard read/write via crumbs module; prompt templates for doc work |
| 02.0 | Stitch for code | cobbler stitch executes code tasks; git worktree management; test execution hooks |
| 03.0 | Measure | cobbler measure assesses project state and proposes tasks; replaces make-work.sh |
| 04.0 | Inspect, mend, pattern | cobbler inspect evaluates output; cobbler mend fixes issues; cobbler pattern proposes design changes |

### Risks and Mitigations

| Table 4 Risks |
|---------------|

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Crumbs module API changes | High | Medium | Pin crumbs version; update cobbler when crumbs releases |
| Agent output quality varies | Medium | High | Inspect and mend commands provide feedback loop; human approval gates |
| Prompt template complexity | Medium | Medium | Start with simple templates; iterate based on agent performance |
| Git worktree edge cases | Low | Medium | Test worktree lifecycle thoroughly; graceful cleanup on failure |

## What This Is NOT

We are not a task tracker. Use crumbs (or bd CLI) to manage work items. Cobbler reads and writes to the cupboard but does not replace it.

We are not an IDE plugin. Cobbler runs from the terminal. We do not integrate with VS Code, JetBrains, or other editors.

We are not a CI system. We do not run on push, trigger builds, or gate merges. Cobbler runs when a developer invokes it.

We are not fully autonomous. Humans approve measure output before stitch executes. Humans review inspect findings before mend runs. Cobbler assists; it does not replace human judgment.

We are not a workflow engine. We do not define DAGs, manage dependencies between jobs, or orchestrate multi-service deployments. We coordinate a single agent to do a single task at a time.

We are not an LLM wrapper or prompt library. We call AI agents through their existing interfaces (Claude Code). We do not abstract over model providers or manage API keys.

| Table 5 Comparison to Related Concepts |
|----------------------------------------|

| Concept | What it does | How cobbler differs |
|---------|--------------|---------------------|
| Task tracker (Jira, Linear) | Stores and manages work items | Cobbler reads from crumbs; does not store tasks |
| CI system (GitHub Actions, Jenkins) | Runs jobs on events | Cobbler runs on developer command, not events |
| Agentic framework (LangChain, AutoGPT) | Provides agent abstractions | Cobbler orchestrates existing agents; does not provide new abstractions |
| Workflow engine (Temporal, Airflow) | Manages DAGs of jobs | Cobbler runs single tasks sequentially |

## References

See ARCHITECTURE.md for component design and interfaces. See PRDs in docs/product-requirements/ for detailed requirements.
