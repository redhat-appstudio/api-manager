# api-manager `WIP`

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
    kubectl get apiexports api-manager-appstudio.redhat.com -o yaml
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
3. find the controller pod in the k8s cluster:
   ```shell
   KUBECONFIG=~/.kube/config kubectl get po -A -l control-plane=controller-manager 
   ```
4. check the logs of the above pod, you should see the following content:
    ```shell
   1.666207217489971e+09	INFO	Getting APIBinding ...	{"controller": "apibinding", "controllerGroup": "apis.kcp.dev", "controllerKind": "APIBinding", "aPIBinding": {"name":"apiresource.kcp.dev-crsud"}, "namespace": "", "name": "apiresource.kcp.dev-crsud", "reconcileID": "95f343bd-8afc-4c85-b997-5c81d31e2c3d"}
   1.6662072174914813e+09	INFO	apibinding workspace: 	{"controller": "apibinding", "controllerGroup": "apis.kcp.dev", "controllerKind": "APIBinding", "aPIBinding": {"name":"apiresource.kcp.dev-crsud"}, "namespace": "", "name": "apiresource.kcp.dev-crsud", "reconcileID": "95f343bd-8afc-4c85-b997-5c81d31e2c3d", "workspace path": "root"}
   1.6662072174922872e+09	INFO	Getting APIBinding ...	{"controller": "apibinding", "controllerGroup": "apis.kcp.dev", "controllerKind": "APIBinding", "aPIBinding": {"name":"appstudio.redhat.com"}, "namespace": "", "name": "appstudio.redhat.com", "reconcileID": "228ff4dc-3015-488c-9a90-a3188a84e314"}
   1.6662072174924483e+09	INFO	apibinding workspace: 	{"controller": "apibinding", "controllerGroup": "apis.kcp.dev", "controllerKind": "APIBinding", "aPIBinding": {"name":"appstudio.redhat.com"}, "namespace": "", "name": "appstudio.redhat.com", "reconcileID": "228ff4dc-3015-488c-9a90-a3188a84e314", "workspace path": "root:my-org:api-manager-ws"}
   1.666207217492498e+09	INFO	apibinding matches APIExport name	{"controller": "apibinding", "controllerGroup": "apis.kcp.dev", "controllerKind": "APIBinding", "aPIBinding": {"name":"appstudio.redhat.com"}, "namespace": "", "name": "appstudio.redhat.com", "reconcileID": "228ff4dc-3015-488c-9a90-a3188a84e314"}
   1.666207217492536e+09	INFO	deploying apibidings chart	{"controller": "apibinding", "controllerGroup": "apis.kcp.dev", "controllerKind": "APIBinding", "aPIBinding": {"name":"appstudio.redhat.com"}, "namespace": "", "name": "appstudio.redhat.com", "reconcileID": "228ff4dc-3015-488c-9a90-a3188a84e314", "chartPath": "/workspace/chart"}
   1.6662072174926054e+09	INFO	going to deploy apibindings	{"controller": "apibinding", "controllerGroup": "apis.kcp.dev", "controllerKind": "APIBinding", "aPIBinding": {"name":"appstudio.redhat.com"}, "namespace": "", "name": "appstudio.redhat.com", "reconcileID": "228ff4dc-3015-488c-9a90-a3188a84e314", "workspace path": "root:my-org:api-manager-ws", "chart path": "/workspace/chart"}
    ```

## Roadmap

- [x] Project scaffolding
- [ ] Implement GitHub Actions based CI
- [ ] Implement proper reconcile logic
- [ ] Implement unit tests
- [ ] Implement integration tests that can run locally

