#!/bin/bash
set -euo pipefail

export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

# Install k3s — disable traefik so nginx ingress can take port 80
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="server --disable=traefik" sh -

# Wait for k3s node to be ready
until kubectl get node 2>/dev/null | grep -q ' Ready'; do sleep 3; done

# Install nginx ingress controller (LoadBalancer type — klipper-lb binds to host port 80)
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.11.2/deploy/static/provider/cloud/deploy.yaml

# Install ArgoCD
kubectl create namespace argocd
kubectl apply --server-side -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for nginx ingress controller to be ready
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=180s

# Wait for ArgoCD to be ready
kubectl wait --for=condition=available deployment --all -n argocd --timeout=300s

# Apply the nll-scheduler ArgoCD Application
# ArgoCD will pull manifests from k8s/prod/ at the specified git ref
# and override image tags via kustomize
cat <<'APPEOF' | kubectl apply -f -
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: nll-scheduler
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/mak3r/nll-scheduler
    targetRevision: ${app_version}
    path: k8s/prod
  destination:
    server: https://kubernetes.default.svc
    namespace: nll-scheduler
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
APPEOF
