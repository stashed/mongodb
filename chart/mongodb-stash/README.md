# MongoDB-stash

[mongodb-stash](https://github.com/stashed/mongodb-stash) - MongoDB database backup/restore plugin for [Stash by AppsCode](https://appscode.com/products/stash/).

## TL;DR;

```console
helm repo add appscode https://charts.appscode.com/stable/
helm repo update
helm install appscode/mongodb-stash --name=mongodb-stash-3.6 --version=3.6
```

## Introduction

This chart installs necessary `Function` and `Task` definition to backup or restore MongoDB database 3.6 using Stash.

## Prerequisites

- Kubernetes 1.11+

## Installing the Chart

- Add AppsCode chart repository to your helm repository list,

```console
helm repo add appscode https://charts.appscode.com/stable/
```

- Update helm repositories to fetch latest charts from the remove repository,

```console
helm repo update
```

- Install the chart with the release name `mongodb-stash-3.6` run the following command,

```console
helm install appscode/mongodb-stash --name=mongodb-stash-3.6 --version=3.6
```

The above commands installs `Functions` and `Task` crds that are necessary to backup MongoDB database 3.6 using Stash.

## Uninstalling the Chart

To uninstall/delete the `mongodb-stash-3.6` run the following command,

```console
helm delete mongodb-stash-3.6
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following table lists the configurable parameters of the `postgre-stash` chart and their default values.

|        Parameter         |                                                           Description                                                            |     Default      |
| ------------------------ | -------------------------------------------------------------------------------------------------------------------------------- | ---------------- |
| `global.registry`        | Docker registry used to pull respective images                                                                                   | `appscode`       |
| `global.image`           | Docker image used to backup/restore PosegreSQL database                                                                          | `mongodb-stash` |
| `global.tag`             | Tag of the image that is used to backup/restore MongoDB database. This is usually same as the database version it can backup. | `3.6`           |
| `global.backup.mgArgs`   | Optional arguments to pass to `mgdump` command  while bakcup                                                                     |                  |
| `global.restore.mgArgs`  | Optional arguments to pass to `psql` command while restore                                                                       |                  |
| `global.metrics.enabled` | Specifies whether to send Prometheus metrics                                                                                     | `true`           |
| `global.metrics.labels`  | Optional comma separated labels to add to the Prometheus metrics                                                                 |                  |

> We have declared all the configurable parameters as global parameter so that the parent chart can overwrite them.

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`.

For example:

```console
helm install --name mongodb-stash-3.6 --set global.metrics.enabled=false appscode/mongodb-stash
```

**Tips:** Use escape character (`\`) while providing multiple comma-separated labels for `global.metrics.labels`.

```console
 helm install chart/mongodb-stash --set global.metrics.labels="k1=v1\,k2=v2"
```
