# Cluster Install

## K3s Install

After refactoring into using Makefile, you can use the various make targets.
The default target `make k3s-all` will install k3s on the control plane and workers, as well as install cilium and deploy argocd.
Run `make help` for information about the other targets.

reference: [K3s quick-start guide](https://docs.k3s.io/quick-start)

## Bootstrapping

### Argo-managed apps

Once ArgoCD is healthy, you can install the app-of-apps to bootstrap the rest of the platform-apps.

```sh
kubectl apply -f bootstrap/argocd-apps/app-of-apps.yaml
```

## Planet Express

TODO: refactor to get installed through workload-apps

```bash
kubectl create ns planet-express
fd deployment | xargs -I % kubectl apply -f %
```

TODO: cert-manager install with argo
TODO: ingress

# Old Notes below

Notes from before the makefile / argocd app-of-apps refactor are below (but they should probably still work)

## Cert-Manager

[Getting started guide](https://cert-manager.io/docs/installation/)

To install, run
`kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.1/cert-manager.yaml`

After installing, check that it's working with `cmctl check api` (available via brew).

:warn: Get a Cloudflare API token that has permission to update domain DNS records on your behalf. Make sure to not commit it to git :warn:

Create a secret with your Cloudflare key, e.g. `kaf cert-manager/secret.yaml`. This will allow cert-manager to use your cloudflare token on your behalf to make certificate issuing requests.

Test out the token validity, clusterissuer, and certificate files by doing this in staging first, where there are are not strict rate limits.

```bash
kubectl apply -f cert-manager/staging
```

Check status with

```bash
kubectl get certificate ...
kubectl describe certificate ...
kubectl get certificaterequest ...
kubectl describe certificaterequest ...
kubectl get order ...
kubectl describe order ...
kubectl get challenge ...
kubectl describe challenge ...
```

Once you've verified the certificates are provisioned and working correctly (it may take 5-10 minutes for the certificates to be ready), delete the staging ones and issue the prod ones.

```bash
kubectl delete -f cert-manager/staging
kubectl apply -f cert-manager/prod
```

## Traefik Ingress

```bash
helm repo add traefik https://traefik.github.io/charts
helm repo update
helm install traefik traefik/traefik -f ingress/traefik-values.yaml --wait
kubectl apply -f argocd-server-transport
kubectl apply -f IngressRoutes.yaml
kubectl apply -f ingress.yaml
```

For most services, you can now do straight up TLS with the certs that were provisioned earlier with cert-manager. ArgoCD is a bit different though because the service itself expects a certificate. Since the certificate created by cert-manager doesn't include any IP SANs of the pods (`10.42.x.x`), the TLS will fail when traefik tries to redirect to ArgoCD-Server.

To get around this, we need to create a `ServersTransport` CRD to tell Traefik to skip TLS on the backend when communicating with the ArgoCD Server.

## Vault

Vault no longer needed now that I don't need to run Caddy anymore.
