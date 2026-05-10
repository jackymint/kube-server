# kube-server

Lightweight Kubernetes cluster manager for macOS (Apple Silicon & Intel).

Faster and lighter than OrbStack — built with vfkit, k0s, and Alpine Linux.

## Install

```bash
brew tap jackymint/kube-server
brew install kube-server
```

## Usage

```bash
# Cluster management
kube-server cluster create dev
kube-server cluster start dev
kube-server cluster stop dev
kube-server cluster list

# Node management
kube-server node add dev
kube-server node remove dev dev-worker-1
kube-server node resize dev dev-worker-1 --disk 20GB

# Interactive TUI
kube-server tui dev
```

## Default Config

| | Control-plane | Worker |
|---|---|---|
| Count | 1 | 2 |
| CPU | 2 cores | 2 cores |
| RAM | 2GB | 2GB |
| Disk | 20GB | 10GB |

Config file: `~/.kube-server/config.yaml`

```yaml
cluster:
  worker_count: 2
  control_plane:
    cpu: 2
    memory: 2GB
    disk: 20GB
  worker:
    cpu: 2
    memory: 2GB
    disk: 10GB
```

## Requirements

- macOS Ventura or later
- Apple Silicon (M1/M2/M3) or Intel
