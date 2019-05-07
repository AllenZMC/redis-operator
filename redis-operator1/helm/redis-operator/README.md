# redis-operator

This chart installs redis-operator in your cluster.

## Prerequisites

* Kubernetes 1.12+
* Tiller v2.8+

## Chart details

This chart:
* Installs redis-operator.
* Adds a resource definition for redis servers.
* Adds a default redis server resource

## Configuration

The following tables lists the configurable parameters of the redis-operator chart and the default values.

| Parameter                  | Description                        | Default                 |
| -----------------------    | ---------------------------------- | ----------------------- |
| **Redis** |
| `redis.sentinels.replicas`     | Redis sentinel replica count | `3` |
| `redis.sentinels.quorum`       | The number of Sentinels that need to agree about the fact the master is not reachable | `2` |
| `redis.slaves.replicas`  | Redis slave replica count | `3` |
| **Image** |
| `image.repository` | Image repository used for redis operator | `allenczm/redis-operator` |
| `image.tag` | Tag used for redis operator | `0.1.5` |
| `image.pullPolicy` | Image pull policy | `Always` |

## Debug the chart
```
helm install --dry-run --debug .
```

## Installing the chart

```
helm install -n redis --namespace redis-operator .
```

## Edit the CR Redis yaml
```
kubectl edit Redis redis-default -n redis-operator
```

## Removing the chart

1. Delete all the redis resources, this will ensure that the operator will delete all the resources it created
    ```
    kubectl delete redis -n redis-operator --all
    kubectl delete crd redises.operator.joelws.com
    ```
2. Remove the chart.
    ```
    helm delete --purge redis
    ```