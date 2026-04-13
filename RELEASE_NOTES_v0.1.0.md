# Release Notes - v0.1.0

**Release Date:** 2026-04-13

## Highlights

LangChainGo adapter for omnillm - use any omnillm provider with LangChainGo chains, agents, and RAG pipelines.

## Overview

This initial release provides a complete implementation of the `llms.Model` interface from [LangChainGo](https://github.com/tmc/langchaingo), allowing you to use [omnillm](https://github.com/plexusone/omnillm)'s unified LLM provider abstraction with LangChainGo's ecosystem of chains, agents, memory, and RAG tools.

## Features

### Core Functionality

- **Model Adapter**: `New()` function creates a LangChainGo-compatible model from any omnillm provider
- **Full Interface**: Implements `llms.Model` with `GenerateContent()` and `Call()` methods
- **Streaming**: Native streaming support via `llms.WithStreamingFunc()` callback

### Tool Calling

- Automatic conversion between LangChainGo and omnillm tool formats
- Support for function definitions with parameters and descriptions
- Tool call response handling for agent loops

### Options Support

All standard LangChainGo call options are supported:

- `llms.WithTemperature()`
- `llms.WithMaxTokens()`
- `llms.WithTopP()`
- `llms.WithStopWords()`
- `llms.WithSeed()`

### Message Types

Full message role mapping:

- System → system
- Human → user
- AI → assistant
- Tool → tool (with tool call ID)

## Installation

```bash
go get github.com/plexusone/omnillm-langchaingo
```

## Quick Start

```go
import (
    "github.com/plexusone/omnillm"
    "github.com/plexusone/omnillm-langchaingo"
    "github.com/tmc/langchaingo/llms"
)

// Create omnillm client
client := omnillm.NewClient(omnillm.ClientConfig{
    Provider: omnillm.ProviderNameAnthropic,
    APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
})

// Create LangChainGo model
llm := langchaingo.New(client, "claude-sonnet-4-20250514")

// Use with LangChainGo
response, _ := llms.GenerateFromSinglePrompt(ctx, llm, "Hello!")
```

## Architecture

```
┌───────────────────────────────────────────────────────┐
│  LangChainGo (chains, agents, RAG, memory, tools)     │
├───────────────────────────────────────────────────────┤
│  omnillm-langchaingo (this package)                   │
│  implements llms.Model interface                      │
├───────────────────────────────────────────────────────┤
│  omnillm / omnillm-core                               │
│  unified LLM provider abstraction                     │
└───────────────────────────────────────────────────────┘
```

## Known Limitations

- `ImageURLContent` not yet supported (returns error)
- `BinaryContent` not yet supported (returns error)

These will be addressed in future releases as omnillm adds multimodal support.

## Contributors

- @grokify
