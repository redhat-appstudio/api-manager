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

build the docker image
```shell
make docker-build REGISTRY=quay.io ORG=<organization-name-here>
```

push the docker image
```shell
make docker-push REGISTRY=quay.io ORG=<organization-name-here>
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
    make deploy
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
3. check that the bindings are ok:
   ```shell
   kubectl get apibindings appstudio.redhat.com -o yaml
   ```
4. deploy an ApiManager object:
    ```shell
    kubectl apply -f config/samples/appstudio_v1alpha1_apimanager.yaml
    ```
5. find the controller pod in the k8s cluster:
   ```shell
   KUBECONFIG=~/.kube/config kubectl get po -A -l control-plane=controller-manager 
   ```
6. check the logs of the above pod, you should see the following content:
    ```shell
    00 OK in 7 milliseconds
    I1013 21:44:24.667372       1 leaderelection.go:278] successfully renewed lease api-manager-system/751dd497.redhat.com
    I1013 21:44:26.685268       1 round_trippers.go:553] GET https://192.168.1.110:6443/apis/coordination.k8s.io/v1/namespaces/api-manager-system/leases/751dd497.redhat.com 200 OK in 11 milliseconds
    I1013 21:44:26.702299       1 round_trippers.go:553] PUT https://192.168.1.110:6443/apis/coordination.k8s.io/v1/namespaces/api-manager-system/leases/751dd497.redhat.com 200 OK in 7 milliseconds
    I1013 21:44:26.703503       1 leaderelection.go:278] successfully renewed lease api-manager-system/751dd497.redhat.com
    1.6656974669204428e+09	INFO	Getting apimanager ...	{"controller": "apimanager", "controllerGroup": "appstudio.redhat.com", "controllerKind": "ApiManager", "apiManager": {"name":"apimanager-sample","namespace":"default"}, "namespace": "default", "name": "apimanager-sample", "reconcileID": "a4cf7904-ae96-4c2c-885e-2a424ef67440"}
    1.6656974669205887e+09	INFO	apimanger found: 	{"controller": "apimanager", "controllerGroup": "appstudio.redhat.com", "controllerKind": "ApiManager", "apiManager": {"name":"apimanager-sample","namespace":"default"}, "namespace": "default", "name": "apimanager-sample", "reconcileID": "a4cf7904-ae96-4c2c-885e-2a424ef67440", "object": {"kind":"ApiManager","apiVersion":"appstudio.redhat.com/v1beta1","metadata":{"name":"apimanager-sample","namespace":"default","uid":"6a154bbe-6284-4eaf-be9e-062576a7b136","resourceVersion":"3336","generation":1,"creationTimestamp":"2022-10-13T21:44:26Z","annotations":{"kcp.dev/cluster":"root:my-org:api-manager-consumer-ws","kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"appstudio.redhat.com/v1beta1\",\"kind\":\"ApiManager\",\"metadata\":{\"annotations\":{},\"name\":\"apimanager-sample\",\"namespace\":\"default\"},\"spec\":null}\n"},"managedFields":[{"manager":"kubectl-client-side-apply","operation":"Update","apiVersion":"appstudio.redhat.com/v1beta1","time":"2022-10-13T21:44:26Z","fieldsType":"FieldsV1","fieldsV1":{"f:metadata":{"f:annotations":{".":{},"f:kubectl.kubernetes.io/last-applied-configuration":{}}}}}]},"spec":{},"status":{}}}
    I1013 21:44:28.712838       1 round_trippers.go:553] GET https://192.168.1.110:6443/apis/coordination.k8s.io/v1/namespaces/api-manager-system/leases/751dd497.redhat.com 200 OK in 7 milliseconds
    I1013 21:44:28.718440       1 round_trippers.go:553] PUT https://192.168.1.110:6443/apis/coordination.k8s.io/v1/namespaces/api-manager-system/leases/751dd497.redhat.com 200 OK in 4 milliseconds
    I1013 21:44:28.719352       1 leaderelection.go:278] successfully renewed lease api-manager-system/751dd497.redhat.com
    ```

## Roadmap

- [x] Project scaffolding
- [ ] Implement GitHub Actions based CI
- [ ] Implement proper reconcile logic
- [ ] Implement unit tests
- [ ] Implement integration tests that can run locally

