# Bifrost Gateway

Bifrost Gateway is a blazing-fast HTTP API that unifies access to 15+ AI providers (OpenAI, Anthropic, AWS Bedrock, Google Vertex, and more) through a single OpenAI-compatible interface. Deploy in seconds with zero configuration and get automatic fallbacks, semantic caching, tool calling, and enterprise-grade features.

**Complete Documentation**: [https://github.com/maximhq/bifrost/wiki

---

## Quick Start

### Installation

Choose your preferred method:

#### NPX (Recommended)

```bash
# Install and run locally
npx -y @maximhq/bifrost

# Open web interface at http://localhost:8080
```

#### Docker

```bash
# Pull and run Bifrost Gateway
docker pull maximhq/bifrost
docker run -p 8080:8080 maximhq/bifrost

# For persistent configuration
docker run -p 8080:8080 -v $(pwd)/data:/app/data maximhq/bifrost
```

### Configuration

Bifrost starts with zero configuration needed. Configure providers through the **built-in web UI** at `http://localhost:8080` or via API:

```bash
# Add OpenAI provider via API
curl -X POST http://localhost:8080/api/providers \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "openai",
    "keys": [{"value": "sk-your-openai-key", "models": ["gpt-4o-mini"], "weight": 1.0}]
  }'
```

For file-based configuration, create `config.json` in your app directory:

```json
{
  "providers": {
    "openai": {
      "keys": [{"value": "env.OPENAI_API_KEY", "models": ["gpt-4o-mini"], "weight": 1.0}]
    }
  }
}
```

### Your First API Call

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "openai/gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello, Bifrost!"}]
  }'
```

**That's it!** You now have a unified AI gateway running locally.

---

## Key Features

Bifrost Gateway provides enterprise-grade AI infrastructure with these core capabilities:

### Core Features

- **[Unified Interface](https://github.com/maximhq/bifrost/wiki
- **[Multi-Provider Support](https://github.com/maximhq/bifrost/wiki
- **[Drop-in Replacement](https://github.com/maximhq/bifrost/wiki
- **[Automatic Fallbacks](https://github.com/maximhq/bifrost/wiki
- **[Streaming Support](https://github.com/maximhq/bifrost/wiki

### Advanced Features

- **[Model Context Protocol (MCP)](https://github.com/maximhq/bifrost/wiki
- **[Semantic Caching](https://github.com/maximhq/bifrost/wiki
- **[Load Balancing](https://github.com/maximhq/bifrost/wiki
- **[Governance & Budget Management](https://github.com/maximhq/bifrost/wiki
- **[Custom Plugins](https://github.com/maximhq/bifrost/wiki

### Enterprise Features

- **[Clustering](https://github.com/maximhq/bifrost/wiki
- **[SSO Integration](https://github.com/maximhq/bifrost/wiki
- **[Vault Support](https://github.com/maximhq/bifrost/wiki
- **[Custom Analytics](https://github.com/maximhq/bifrost/wiki
- **[In-VPC Deployments](https://github.com/maximhq/bifrost/wiki

**Learn More**: [Complete Feature Documentation](https://github.com/maximhq/bifrost/wiki

---

## SDK Integrations

Replace your existing SDK base URLs to unlock Bifrost's features instantly:

### OpenAI SDK

```python
import openai
client = openai.OpenAI(
    base_url="http://localhost:8080/openai",
    api_key="dummy"  # Handled by Bifrost
)
```

### Anthropic SDK

```python
import anthropic
client = anthropic.Anthropic(
    base_url="http://localhost:8080/anthropic",
    api_key="dummy"  # Handled by Bifrost
)
```

### Google GenAI SDK

```python
import google.generativeai as genai
genai.configure(
    transport="rest",
    api_endpoint="http://localhost:8080/genai",
    api_key="dummy"  # Handled by Bifrost
)
```

**Complete Integration Guides**: [SDK Integrations](https://github.com/maximhq/bifrost/wiki

---

## Documentation

### Getting Started

- [Quick Setup Guide](https://github.com/maximhq/bifrost/wiki
- [Provider Configuration](https://github.com/maximhq/bifrost/wiki
- [Integration Guide](https://github.com/maximhq/bifrost/wiki

### Advanced Topics

- [MCP Tool Calling](https://github.com/maximhq/bifrost/wiki
- [Semantic Caching](https://github.com/maximhq/bifrost/wiki
- [Fallbacks & Load Balancing](https://github.com/maximhq/bifrost/wiki
- [Budget Management](https://github.com/maximhq/bifrost/wiki

**Browse All Documentation**: [https://github.com/maximhq/bifrost/wiki

---

*Built with ❤️ by [Maxim](https://github.com/maximhq/bifrost)*
