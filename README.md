# Echo Operator

Author: Brian Tomlinson <btomlins@redhat.com>


## Description

Echo Operator manages the Echo application.


## Development Steps

1. `operator-sdk new echo-operator`
2. `operator-sdk add api --api-version=echo.rhte2019.io/v1alpha1 --kind=Echo`
3. `operator-sdk add controller --api-version=echo.rhte2019.io/v1alpha1 --kind=Echo`
4. Edit `apis/echo/v1alpha1/echo_types.go` to include fields required to instantiate an Echo application
5. `operator-sdk generate k8s`
6. `operator-sdk generate openapi`
7. Edit `controller/echo/echo_controller.go` to include the logic for managing an Echo application
8. Test the operator with `operator-sdk up local --namespace $NAMESPACE`
9. Build the `echo-operator` image with `operator-sdk build --image-builder buildah $REGISTRY/echo-operator:$TAG`
10. Push the operator image to $REGISTRY with `buildah push $REGISTRY/echo-operator:$TAG`
11. Add the image to `deploy/operator.yaml`


## Installation

TBD


## Usage

TBD


## License

TBD
