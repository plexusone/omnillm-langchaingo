# Release Notes - v0.2.0

**Release Date:** 2026-05-10

## Breaking Change: Module Renamed

This release renames the module to follow LangChain's `langchain-<provider>` naming convention.

### Migration

Update your imports:

```go
// Before
import "github.com/plexusone/omnillm-langchaingo"

// After
import "github.com/plexusone/langchaingo-omnillm"
```

Update go.mod:

```bash
go get github.com/plexusone/langchaingo-omnillm@v0.2.0
```

## Why the Rename?

Aligns with LangChain's established `langchain-<provider>` convention used across the ecosystem:

- `langchain-openai`
- `langchain-anthropic`
- `langchain-google-genai`

This makes the package more discoverable and consistent with user expectations.

## No Functional Changes

All functionality remains identical to v0.1.0. Only the module path has changed.
