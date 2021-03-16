# Beacon

![GitHub Workflow Status](https://img.shields.io/github/workflow/status/vilsol/beacon/build)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/vilsol/beacon)
![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/vilsol/beacon)

```
kubernetes beacon that restarts deployments when image updates

Usage:
  beacon [command]

Available Commands:
  help        Help about any command
  run         Run the beacon

Flags:
  -h, --help                 help for beacon
      --interval duration    Interval between update checks (default 10m0s)
      --kubeconfig string    absolute path to the kubeconfig file (default "C:\\Users\\User\\.kube\\config")
      --label string         Label to filter deployments (default "vilsol.beacon")
      --log string           The log level to output (default "info")
      --namespaces strings   List of namespaces to monitor (defaults to all)
      --pretty               Use pretty logging output (non-json)

Use "beacon [command] --help" for more information about a command.
```

## Helm

You can install beacon using helm

```shell
helm repo add beacon https://vilsol.github.io/beacon/
helm repo update
helm install beacon beacon/beacon 
```