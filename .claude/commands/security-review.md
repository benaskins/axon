---
description: Review code for security vulnerabilities
allowed-tools: Read, Grep, Glob, Bash(git diff *)
model: sonnet
---

# Security Review

Scan for security vulnerabilities. Two modes:

1. **Changed files** (default): Review files changed since the base branch.
2. **Full repo**: When `$ARGUMENTS` contains "all" or "full", review all `.go` files in the repo.

## Determine scope

If `$ARGUMENTS` contains "all" or "full":
- Use `Glob` to find all `.go` files (exclude `*_test.go` and `vendor/`)
else:
- Use `git diff --name-only origin/main...HEAD` to get changed files
- If that fails (e.g. no remote tracking), fall back to scanning all `.go` files

Read each file in scope.

## Check for

**Critical:**
- SQL injection (string concatenation in queries, fmt.Sprintf in SQL)
- Command injection (os/exec with unsanitized input)
- Path traversal (user input in file paths)
- Hardcoded secrets, credentials, or API keys
- InsecureSkipVerify or disabled TLS validation

**High:**
- Authentication/authorization bypasses
- Missing input validation at system boundaries
- Unsafe deserialization
- Race conditions on shared state without synchronization

**Medium:**
- Sensitive data in logs or error messages
- Missing timeouts on HTTP clients or contexts
- Unbounded allocations from user input (DoS)
- Weak cryptographic choices

**Low:**
- Exported types that should be unexported
- Missing error handling
- Overly broad file permissions

## Output format

For each finding:
- **File:line** — exact location
- **Severity** — Critical / High / Medium / Low
- **Issue** — what's wrong
- **Fix** — concrete recommendation

Group by severity (Critical first). If no issues found, say so.
