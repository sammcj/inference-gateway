# Example using docker-compose

## Prerequisites

- Docker
- Docker Compose

## Quick Guide

Copy `.env.example` to `.env` and adjust the values (`.env` is added to gitignore and will not be tracked).

1. Bring the environment up:

```bash
docker compose up -d
```

2. List the available models of a specific API, for example Groq:

```bash
curl http://localhost:8080/llms/groq/openai/v1/models
```

Or the local models:

```bash
curl http://localhost:8080/llms/ollama/v1/models
```

3. Use a specific API models, for example Groq:

```bash
curl http://localhost:8080/llms/groq/openai/v1/chat/completions -s -d '{"model": "llama-3.3-70b-versatile","messages": [{"role": "user","content": "Explain the importance of fast language models"}], "stream": true}'
```
