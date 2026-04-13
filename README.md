# omnillm-langchaingo

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/plexusone/omnillm-langchaingo/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/omnillm-langchaingo/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/omnillm-langchaingo/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/omnillm-langchaingo/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/plexusone/omnillm-langchaingo/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/plexusone/omnillm-langchaingo/actions/workflows/go-sast-codeql.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/plexusone/omnillm-langchaingo
 [goreport-url]: https://goreportcard.com/report/github.com/plexusone/omnillm-langchaingo
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/omnillm-langchaingo
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/omnillm-langchaingo
 [viz-svg]: https://img.shields.io/badge/visualizaton-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=plexusone%2Fomnillm-langchaingo
 [loc-svg]: https://tokei.rs/b1/github/plexusone/omnillm-langchaingo
 [repo-url]: https://github.com/plexusone/omnillm-langchaingo
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/omnillm-langchaingo/blob/master/LICENSE

LangChainGo adapter for [omnillm](https://github.com/plexusone/omnillm).

This package implements the `github.com/tmc/langchaingo/llms.Model` interface using omnillm as the LLM backend, allowing you to use omnillm's unified provider abstraction with LangChainGo's chains, agents, and RAG pipelines.

## Installation

```bash
go get github.com/plexusone/omnillm-langchaingo
```

## Usage

### With omnillm (batteries-included)

```go
import (
    "context"
    "fmt"
    "os"

    "github.com/plexusone/omnillm"
    "github.com/plexusone/omnillm-langchaingo"
    "github.com/tmc/langchaingo/llms"
)

func main() {
    // Create omnillm client with multiple providers
    client := omnillm.NewClient(omnillm.ClientConfig{
        Providers: []omnillm.ProviderConfig{
            {Provider: omnillm.ProviderNameAnthropic, APIKey: os.Getenv("ANTHROPIC_API_KEY")},
            {Provider: omnillm.ProviderNameOpenAI, APIKey: os.Getenv("OPENAI_API_KEY")},
        },
    })

    // Create LangChainGo model
    llm := langchaingo.New(client, "claude-sonnet-4-20250514")

    // Use with LangChainGo
    response, err := llms.GenerateFromSinglePrompt(context.Background(), llm,
        "What is the capital of France?")
    if err != nil {
        panic(err)
    }
    fmt.Println(response)
}
```

### With omnillm-core (minimal dependencies)

```go
import (
    omnillm "github.com/plexusone/omnillm-core"
    _ "github.com/plexusone/omnillm-anthropic"  // Import only the provider you need
    "github.com/plexusone/omnillm-langchaingo"
)

func main() {
    client := omnillm.NewClient(omnillm.ClientConfig{
        Provider: omnillm.ProviderNameAnthropic,
        APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
    })

    llm := langchaingo.New(client, "claude-sonnet-4-20250514")
    // ...
}
```

### With LangChainGo Chains

```go
import (
    "github.com/tmc/langchaingo/chains"
    "github.com/tmc/langchaingo/prompts"
)

prompt := prompts.NewPromptTemplate(
    "Analyze this market and estimate probability: {{.question}}",
    []string{"question"},
)

chain := chains.NewLLMChain(llm, prompt)

result, err := chain.Call(context.Background(), map[string]any{
    "question": "Will Bitcoin reach $150k by end of 2025?",
})
```

### With Tool Calling

```go
tools := []llms.Tool{
    {
        Type: "function",
        Function: &llms.FunctionDefinition{
            Name:        "get_market_price",
            Description: "Get current price for a prediction market",
            Parameters: map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "market_id": map[string]any{"type": "string"},
                },
                "required": []string{"market_id"},
            },
        },
    },
}

response, err := llm.GenerateContent(ctx,
    []llms.MessageContent{{
        Role:  llms.ChatMessageTypeHuman,
        Parts: []llms.ContentPart{llms.TextContent{Text: "What's the current price for market xyz?"}},
    }},
    llms.WithTools(tools),
)
```

### With Streaming

```go
response, err := llm.GenerateContent(ctx,
    []llms.MessageContent{{
        Role:  llms.ChatMessageTypeHuman,
        Parts: []llms.ContentPart{llms.TextContent{Text: "Tell me a story"}},
    }},
    llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
        fmt.Print(string(chunk))
        return nil
    }),
)
```

## Features

- Full `llms.Model` interface implementation
- Streaming support with callback
- Tool/function calling support
- All LangChainGo call options (temperature, max tokens, stop words, etc.)
- Token usage tracking in generation info
- Works with both omnillm and omnillm-core

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
│  ┌─────────┬─────────┬─────────┬─────────┬─────────┐  │
│  │ OpenAI  │Anthropic│ Gemini  │ Bedrock │ Ollama  │  │
│  └─────────┴─────────┴─────────┴─────────┴─────────┘  │
└───────────────────────────────────────────────────────┘
```

## License

MIT License - see [LICENSE](LICENSE) for details.
