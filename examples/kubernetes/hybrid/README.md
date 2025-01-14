# Hybrid Kubernetes Example

In this example, we will deploy both Ollama and the Inference Gateway onto a local Kubernetes cluster. The Inference Gateway will facilitate sending requests to Ollama as well as to various cloud provider's Large Language Models (LLMs).

1. Create the local cluster:

```bash
task cluster-create
```

2. Deploy Ollama onto Kubernetes:

```bash
task deploy-ollama
```

3. Wait for the Ollama deployment to be completely rolled out and ready(could take a while due to 2GB download, roughly 3min on a standard internet bandwidth):

```bash
kubectl -n ollama rollout status deployment/ollama
```

4. Deploy the Inference Gateway onto Kubernetes:

```bash
task deploy-inference-gateway
```

5. Wait for the Inference Gateway deployment to be completely rolled out and ready:

```bash
kubectl -n inference-gateway rollout status deployment/inference-gateway
```

6. Proxy the Inference Gateway, to access it locally:

```bash
task proxy
```

7. Check the available Ollama local LLMs:

```bash
curl -X GET http://localhost:8080/llms/ollama/v1/models
```

8. Send a request to the Inference Gateway:

```bash
curl -X POST http://localhost:8080/llms/ollama/api/generate -d '{"model": "phi3:3.8b", "prompt": "Why is the sky blue? keep it short and concise."}'
```

\*\* You can refer to the [Taskfile.yaml](./Taskfile.yaml) at any point for detailed information about the tasks used in this example.

\*\* If the response is cut off mid-stream while the token is still being transmitted, it may be caused by the inference gateway's read timeout. Consider increasing or adjusting the timeout value and redeploying the gateway.
