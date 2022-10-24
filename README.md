# api-manager `WIP`
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=api-manager&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=api-manager)
![Security Scan](https://github.com/redhat-appstudio/api-manager/actions/workflows/security.yaml/badge.svg
)
[![Go Report Card](https://goreportcard.com/badge/github.com/redhat-appstudio/api-manager)](https://goreportcard.com/report/github.com/redhat-appstudio/api-manager)
[![GoDoc](https://godoc.org/github.com/redhat-appstudio/api-manager?status.png)](https://godoc.org/github.com/redhat-appstudio/api-manager)


A [kcp](https://github.com/kcp-dev/kcp) specific controller that creates `apibindings` in _AppStudio user workspaces_.

The controller reconciles every time an _AppStudio user workspace_ is created, and applies a list of _apibindings_ in order to enable all the _service providers_ in this newly created workspace.


_TODO: add details and references regarding the terminology around AppStudio things._


## Development

Following section will list prerequisites and workflow for contributors.

### Prerequisites

* Golang 1.18
* kcp environment
* docker/podman

### Build

build the docker images
```shell
make docker-build 
```
Note: this will build 2 docker images:
- apimanager:latest-appstudio (containing the `./helm/appstudio` Chart in the `/workspace/chart` directory of the container image)
- apimanager:latest-hacbs (containing the `./helm/hacbs` Chart in the `/workspace/chart` directory of the container image)

push the docker images above
```shell
make docker-push
```

### Deploy the controller

Following procedure assumes you have a running [kcp](https://github.com/kcp-dev/kcp) and k8s cluster.

1. create the organization workspace:
    ```shell
    k ws create my-org --enter
    ```
2. create the workload workspace where the controller will be deployed:
    ```shell
    k ws create api-manager-ws --enter
    ```
3. create the sync deployment objects to deploy into your k8s cluster:
    ```shell
    kubectl kcp workload sync docker-desktop --syncer-image ghcr.io/kcp-dev/kcp/syncer:v0.9.0 -o syncer-docker-desktop-main.yaml
    ```
4. edit the `syncer-docker-desktop-main.yaml` above to  enable also `services` resources. For a complete example see: [syncer example](./test/syncer-docker-desktop-main.yaml)
5. deploy the syncher and all the sync objects to your k8s cluster (assuming the kubeconfig for your cluster is under ~/.kube/config):
   ```shell
    KUBECONFIG=~/.kube/config kubectl apply -f syncer-docker-desktop-main.yaml
   ```
6. deploy the controller (assuming you already built and pushed the docker images to the docker registry):
    ```shell
    make deploy_appstudio (or deploy_hacbs for the hacbs docker image)
    ```
7. check the `apiexport` was correctly configured:
    ```shell
    kubectl get apiexports api-manager-export -o yaml
    ```
8. deploy Service Providers APIExports for testing APIBindings:
    ```shell
    make spapiexports
    ```

### Test the deployment

1. create a _consumer workspace_ in the kcp cluster:
    ```shell
    kubectl ws root:my-org && kubectl ws create api-manager-consumer-ws --enter 
    ```
2. create the apibindings:
   ```shell
   make apibinding
   ```
3. check that all APIBindings were "correctly deployed": 
   ```shell
   kubectl get apibindings
   
   api-manager-binding         20s
   apiresource.kcp.dev-crsud   42s
   application-api             19s
   application-service         19s
   build-service               19s
   gitops-appstudio-service    19s
   gitops-core-service         18s
   pipeline-service            18s
   scheduling.kcp.dev-1emov    42s
   spi                         17s
   tenancy.kcp.dev-8so3l       42s
   workload.kcp.dev-4a61h      42s
    ```
NOTE: you should see all the APIBindings created, some bound errors may be present if CRD's are missing.
But all permission claims should show up as accepted.
   
### Check dependency tree using goda

In case you need to look after vulnerable direct/transitive dependencies, or you just want to understand where indirect dependencies are coming from, use [goda](https://pkg.go.dev/github.com/loov/goda#section-readme).

To list all dependency tree run the following from the root folder of the project:
```shell
goda tree ./...:all
```



## Roadmap

- [x] Project scaffolding
- [x] Implement GitHub Actions based CI
- [x] Implement proper reconcile logic
- [ ] Implement unit tests
- [ ] Implement integration tests that can run locally

