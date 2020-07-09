#!/bin/sh

set -e

echo "Creating Kind cluster..."
echo
cat <<EOF | kind create cluster --name gnockgnock --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
EOF
echo

echo "Adding NGINX ingress..."
echo
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/provider/kind/deploy.yaml
echo

echo "Waiting on the ingress..."
echo
sleep 25
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s
echo

echo "Adding gnockgnock service and ingress..."
kubectl apply -f kind-demo.yaml
echo

echo "Now add gnockgnock to your hosts file e.g."
echo "127.0.0.1 gnockgnock"
