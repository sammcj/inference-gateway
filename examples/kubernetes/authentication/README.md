# Basic Kubernetes Example

In this basic example, we will deploy the Inference Gateway.

Feel free to explore the [ConfigMap](inference-gateway/configmap.yaml) and [Secret](inference-gateway/secret.yaml) configurations of the Inference Gateway to set up your desired providers.

We'll be using keycloak as the authentication provider in this example. You can refer to the [keycloak](keycloak) directory for the keycloak configuration.

1. Create the local cluster:

```bash
task cluster-create
```

2. Deploy keycloak onto Kubernetes:

```bash
task deploy-keycloak
```

3. Unfortunately there is a bug and I couldn't activate the client in the keycloak realm via JSON configuration. So you have to do it manually. You can do it by following these steps:

- task proxy-keycloak
- Go to the keycloak admin console: http://localhost:8080/
- Login with the admin credentials: `admin` and `admin`
- Go to the `inference-gateway-realm` realm
- Go to the `Clients` tab
- Click on the `inference-gateway` client
- In the `Settings` tab scroll down to `Capability config` and set `Service Accounts Enabled` to `On` and click on `save`(there is a bug that when you try to activate it via JSON the container crash, but luckily it works with a bit of ClickOps).

4. We also need to add a mapper to the client, to include the `audience` claim in the token. This is required for the Inference Gateway to validate the token. You can do it by following these steps:

   - Go to the `inference-gateway` client.
   - Go to the `Client scopes` tab.
   - Click on `inference-gateway-client-dedicated`.
   - Click on `Add Mapper`.
   - Click `Add by Configuration`.
   - Keep everything default, the mapper type should be `Audience`.
   - Give the name `inference-gateway-client`, this should be equal to the client id.
   - Click on `Save`.

5. Now that we have this part completed, let's request for an ID token, using the client credentials flow(feel free to experiment with other flows as well, this is just an example):

```bash
ACCESS_TOKEN=$(curl -s -X POST -H "Host: keycloak.keycloak:8080" "http://localhost:8080/realms/inference-gateway-realm/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "client_id=inference-gateway-client" \
  -d "client_secret=your-client-secret" \
  -d "grant_type=client_credentials" | jq -r '.access_token')
```

6. Enable the authentication in the Inference Gateway, by setting the `ENABLE_AUTH` environment variable to `true` in the [inference-gateway/configmap.yaml](inference-gateway/configmap.yaml) file:

```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: inference-gateway
  namespace: inference-gateway
  labels:
    app: inference-gateway
data:
  ...
  ENABLE_AUTH: "true" # Change this to true
  OIDC_ISSUER_URL: "http://keycloak:8080/realms/inference-gateway-realm"
  ...
```

7. We also need to obtain the `OIDC_CLIENT_ID` and `OIDC_CLIENT_SECRET` from the keycloak configuration and configure it on the inference Gateway [secret](inference-gateway/secret.yaml):

```yaml
---
apiVersion: v1
kind: Secret
...
stringData:
  OIDC_CLIENT_ID: "inference-gateway-client"
  OIDC_CLIENT_SECRET: "your-client-secret"
  ...
```

8. Deploy the Inference Gateway onto Kubernetes:

```bash
task deploy-inference-gateway
```

9. Proxy the Inference Gateway, to access it locally:

```bash
task proxy-inference-gateway
```

10. First let's try to access an endpoint without the token:

```bash
curl -X GET http://localhost:8080/llms
```

You should see the response `Authorization header missing`.

10. Let's set the token in the header and try again:

```bash
curl -X GET -H "Authorization: Bearer $ACCESS_TOKEN" http://localhost:8080/llms
```

You should be granted and see the expected response.

11. Interact with the Inference Gateway using the specific provider API(note the prefix is `/llms/{provider}/*`):

```bash
curl -X POST -H "Authorization: Bearer $ACCESS_TOKEN" http://localhost:8080/llms/groq/openai/v1/chat/completions -d '{"model": "llama-3.2-3b-preview", "messages": [{"role": "user", "content": "Explain the importance of fast language models. Keep it short and concise."}]}' | jq .
```

## Notes

- The client secret and other sensitive information should not be hardcoded in production.
- Use a proper Keycloak setup for production.
- Adjust the token lifespan in the Keycloak realm settings if needed.
- Consider using a production-grade authentication provider if Keycloak does not meet your needs.
- There is no endpoint for getting the access token for security reasons. Passing a client secret in the frontend is not secure. The client should request the token from the IdP and use it to authenticate with the Inference Gateway. The inference Gateway doesn't store the token, it only validates it with the IdP.
