<h1 align="center">Inference Gateway</h1>

<p align="center">
  <!-- CI Status Badge -->
  <a href="https://github.com/edenreich/inference-gateway/actions/workflows/ci.yml?query=branch%3Amain">
    <img src="https://github.com/edenreich/inference-gateway/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI Status"/>
  </a>
  <!-- Version Badge -->
  <a href="https://github.com/edenreich/inference-gateway/releases">
    <img src="https://img.shields.io/github/v/release/edenreich/inference-gateway?color=blue&style=flat-square" alt="Version"/>
  </a>
  <!-- License Badge -->
  <a href="https://github.com/edenreich/inference-gateway/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/edenreich/inference-gateway?color=blue&style=flat-square" alt="License"/>
  </a>
</p>

The Inference Gateway is a proxy server designed to facilitate access to various language model APIs. It allows users to interact with different language models through a unified interface, simplifying the configuration and the process of sending requests and receiving responses from multiple LLMs, enabling an easy use of Mixture of Experts.

- [Key Features](#key-features)
- [Supported API's](#supported-apis)
- [Configuration](#configuration)
- [Examples](#examples)
- [License](#license)

## Key Features

- 📜 **Open Source**: Available under the MIT License.
- 🚀 **Unified API Access**: Proxy requests to multiple language model APIs, including Groq, OpenAI, Ollama etc.
- ⚙️ **Environment Configuration**: Easily configure API keys and URLs through environment variables.
- 🐳 **Docker Support**: Use Docker and Docker Compose for easy setup and deployment.
- ☸️ **Kubernetes Support**: Ready for deployment in Kubernetes environments.
- 📊 **OpenTelemetry Tracing**: Enable tracing for the server to monitor and analyze performance.
- 🛡️ **Production Ready**: Built with production in mind, with configurable timeouts and TLS support.
- 🌿 **Lightweight**: Includes only essential libraries and runtime, resulting in smaller size binary of ~8.6MB.
- 📉 **Minimal Resource Consumption**: Designed to consume minimal resources and have a lower footprint.
- 📚 **Documentation**: Well documented with examples and guides.
- 🧪 **Tested**: Extensively tested with unit tests and integration tests.
- 🛠️ **Maintained**: Actively maintained and developed.
- 📈 **Scalable**: Easily scalable and can be used in a distributed environment - with <a href="https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/" target="_blank">HPA</a> in Kubernetes.

## Supported API's

- [OpenAI](https://platform.openai.com/)
- [Ollama](https://ollama.com/)
- [Groq Cloud](https://console.groq.com/)
- [Google](https://aistudio.google.com/)

## Configuration

The following environment variables could be set for the Inference Gateway:

| Variable               | Description                         | Default    |
| ---------------------- | ----------------------------------- | ---------- |
| `ENVIRONMENT`          | `production` or `development`.      | production |
| `ENABLE_TELEMETRY`     | Enable telemetry for the server.    | false      |
| `SERVER_PORT`          | The port the server will listen on. | 8080       |
| `SERVER_READ_TIMEOUT`  | The server read timeout.            | 30s        |
| `SERVER_WRITE_TIMEOUT` | The server write timeout.           | 30s        |
| `SERVER_IDLE_TIMEOUT`  | The server idle timeout.            | 120s       |
| `SERVER_TLS_CERT_PATH` | The path to the TLS certificate.    | ""         |
| `SERVER_TLS_KEY_PATH`  | The path to the TLS key.            | ""         |

The following environment variables could be set for the LLMs APIs:

| Variable                  | Description                                | Default                                   |
| ------------------------- | ------------------------------------------ | ----------------------------------------- |
| `OLLAMA_API_URL`          | The URL for Ollama API.                    | ""                                        |
| `GROQ_API_URL`            | The URL for Groq Cloud API.                | https://api.groq.com                      |
| `GROQ_API_KEY`            | The Access for Groq Cloud API.             | ""                                        |
| `OPENAI_API_URL`          | The URL for the OpenAI API.                | https://api.openai.com                    |
| `OPENAI_API_KEY`          | The Access token for OpenAI API.           | ""                                        |
| `GOOGLE_AISTUDIO_API_URL` | The URL for the Google AI Studio API.      | https://generativelanguage.googleapis.com |
| `GOOGLE_AISTUDIO_API_KEY` | The Access token for Google AI Studio API. | ""                                        |

If the API key is not set, the API will not be available.

## Examples

- Using [Docker Compose](examples/docker-compose/)
- Using [Kubernetes](examples/kubernetes/)
- Using standard [REST endpoints](examples/rest-endpoints/)

## License

This project is licensed under the MIT License.
