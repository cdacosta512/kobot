![alt text](assests/kobot_nb_black.png)
# Kobot - Kubernetes Cluster Health Analyzer

> This project is under development so if you choose to use this tool, understand things are subject to change.

Kobot is a Go-native, read-only CLI that analyzes Kubernetes cluster health and gives operators and release engineers a concise, high-level overview of the cluster and its resources.

Here are the core features and design goals at a glance:

- Go-native single binary — minimal dependencies.  
- Read-only diagnostics — non-destructive queries only.  
- Connects directly to your cluster using kubeconfig and the same context selection workflow as kubectl.  
- Human-friendly summaries and machine-friendly outputs (JSON/YAML) for automation.  
- Lightweight and fast — ideal for troubleshooting and release verification.

### Note...
I built Kobot after countless frustrating moments where my teammates and I spent hours trying to get our clusters back to a healthy state after releases (more on that below). But beyond solving that pain point, Kobot is also my way of building something real in Go — a project I hope will grow through community feedback, scrutiny, and constructive criticism.

## The Need for Kobot
As a platform engineering team, we’re responsible for maintaining both the underlying infrastructure and the workloads running within our Kubernetes clusters. Despite having several verification checks built into our deployment pipelines, we often discovered too late that certain cluster resources had become unhealthy after updates were applied.

At first glance, everything looked fine. But hours—or sometimes days—after a release, engineers would find that critical components such as Flux HelmReleases were suspended, or that pods running essential services (like our internal container registry) had been stuck in a crash loop far longer than expected.

That’s where Kobot comes in.
Kobot was built to close this visibility gap: Is the cluster—and everything running inside it—healthy after updates? If not, Kobot provides a clear, actionable health report so teams can immediately identify what’s unhealthy and begin investigating the root cause.

## License
Kobot is licensed under the [Apache License 2.0](LICENSE).

You may freely use, modify, and distribute this software under the terms of the license.