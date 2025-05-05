Various things for my home lab


# Cluster Install
## K3s Install
[K3s quick-start guide](https://docs.k3s.io/quick-start)

### Server
Upload this to `/etc/rancher/k3s/config.yaml`
```
write-kubeconfig-mode: "0644"
tls-san:
  - "wgraham.io"
```

and run `curl -sfL https://get.k3s.io | sh -s -`

Get the token from the server to join the clients
`cat /var/lib/rancher/k3s/server/node-token`

### Agents
Run this on each one

`curl -sfL https://get.k3s.io | K3S_TOKEN="TOKEN_FROM_SERVER" K3S_URL=https://wgraham.io:6443 sh -`

# Bootstrapping
### Pre-anything steps

Just make all the namespaces now
```
kubectl create ns vault
kubectl create ns argocd
kubectl create ns logging
kubectl create ns monitoring
kubectl create ns cert-manager
```

## Cert-Manager
[Getting started guide](https://cert-manager.io/docs/installation/)

To install, run
`kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.17.2/cert-manager.yaml`

After installing, check that it's working with `cmctl check api` (available via brew).

Then create a secret with your Cloudflare key, e.g.

```
apiVersion: v1
kind: Secret
metadata:
  name: cloudflare-api-token-secret
  namespace: cert-manager
type: Opaque
stringData:
  api-token: <API_TOKEN>
```

Now you can provision the `staging` and `prod` Cluster Issuers.

Once these have been created, you can finally provision the `ceritificates.yaml`. These take a few minutes to fully provision. Also, there are strict rate limits on the prod certificates, so only do these if you're sure the setup is good already.

Check status with
```
kubectl get certificate ...
kubectl describe certificate ...
kubectl get certificaterequest ...
kubectl describe certificaterequest ...
kubectl get order ...
kubectl describe order ...
kubectl get challenge ...
kubectl describe challenge ...
```

## Traefik Ingress

Make sure you install CRDs first
`kubectl apply -f https://raw.githubusercontent.com/traefik/traefik/v3.3/docs/content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml`

For most services, you can now do straight up TLS with the certs that were provisioned earlier with cert-manager. ArgoCD is a bit different though because the service itself expects a certificate. Since the certificate created by cert-manager doesn't include any IP SANs of the pods (`10.42.x.x`), the TLS will fail when traefik tries to redirect to ArgoCD-Server.

To get around this, we need to create a `ServersTransport` CRD to tell Traefik to skip TLS on the backend when communicating with the ArgoCD Server.

```
kubectl apply -f argocd-server-transport
kubectl apply -f IngressRoutes.yaml
kubectl apply -f ingress.yaml
```

## ElasticSearch
[Install ECK with manifests](https://www.elastic.co/docs/deploy-manage/deploy/cloud-on-k8s/install-using-yaml-manifest-quickstart)

First, install CRDS and the operator

```
kubectl create -f https://download.elastic.co/downloads/eck/3.0.0/crds.yaml
kubectl apply -f https://download.elastic.co/downloads/eck/3.0.0/operator.yaml
```

Now we can apply the local elasic and kibana manifests

```
kubectl apply -f elasticsearch.yaml
kubectl apply -f kibana.yaml
```

After doing this, make sure to update the password in the `logging/fluent-bit-values.yaml` so the pods can authenticate. Password can be obtained with `kubectl get secret -n logging elastic-es-elastic-user -o json | jq -r '.data."elastic"' | base64 -d`

## ArgoCD
[ArgoCD quick-start guide](https://argo-cd.readthedocs.io/en/stable/getting_started/)

kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

### TLS Configuration
If you have already created certificates in all namespaces,
kubectl get secret -n argocd wildcard-tls -o json | jq -r '.data."tls.key"' | base64 -d > tls.key
kubectl get secret -n argocd wildcard-tls -o json | jq -r '.data."tls.crt"' | base64 -d > tls.crt
kubectl create -n argocd secret tls argocd-server-tls --cert=tls.crt --key=tls.key

### Argo-managed apps
`kubectl apply -f <yamls-in-argocd-dir>`

For fluent-bit, make sure elasticsearch is set up already though

## Vault
Just `kaf` the `vault.yaml` bruh.

Gonna actually not expose the ingress for this one, and might eventually take it out. Not even sure if Caddy needs to be running anymore.

## Grafana
This one is pretty fully baked using the community helm charts

Add the helm repo and install
```
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install -n monitoring kube-prometheus prometheus-community/kube-prometheus-stack
```

# Security

For now, the `grafana` and `kibana` ingresses are disabled (uninstall them) because they don't have secure configs. Once this is fixed we can expose them, but for now I'm not comfortable doing that.
