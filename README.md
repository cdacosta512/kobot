![alt text](assests/kobot_nb_black.png)
# Kobot - Kubernetes Cluster Health Analyzer
Kobot is a Go-native, read-only CLI that analyzes Kubernetes cluster health and gives operators and release engineers a concise, high-level overview of the cluster and its resources.

Here are the core features and design goals at a glance:

- Go-native single binary — minimal dependencies.  
- Read-only diagnostics — non-destructive queries only.  
- Connects directly to your cluster using kubeconfig and the same context selection workflow as kubectl.  
- Human-friendly summaries and machine-friendly outputs (JSON/YAML) for automation.  
- Lightweight and fast — ideal for troubleshooting and release verification.







- keycloak verification -- check the redirect
- pods healthy
- HR are ready
- All apps on the correct versions
- blackbox