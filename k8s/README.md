# Kubernetes

This directory has example plain Kubernetes manifests for running DDNS-updater in Kubernetes.

The Manifests have additional [Kustomize](https://kustomize.io/) overlays, which can be used to add an [ingress-route](https://kubernetes.io/docs/concepts/services-networking/ingress/) to ddns-updater.

1. Download the template files from the [`base` directory](base). For example with:

    ```sh
    curl -O https://raw.githubusercontent.com/qdm12/ddns-updater/master/k8s/base/deployment.yaml
    curl -O https://raw.githubusercontent.com/qdm12/ddns-updater/master/k8s/base/secret-config.yaml
    curl -O https://raw.githubusercontent.com/qdm12/ddns-updater/master/k8s/base/service.yaml
    curl -O https://raw.githubusercontent.com/qdm12/ddns-updater/master/k8s/base/kustomization.yaml
    ```

1. Modify `secret-config.yaml` as described in the [project readme](../README.md#configuration)
1. Use [kubectl](https://kubernetes.io/docs/reference/kubectl/) to apply the manifest:

    ```sh
    kubectl apply -k .
    ```

1. Connect the the web UI with a [kubectl port-forward](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/)

    ```sh
    kubectl port-forward svc/ddns-updater 8080:80
    ```

The web UI should now be available at [http://localhost:8080](http://localhost:8080).

## Advanced usage

Kustomize overlays can extend the installation:

* [overlay/with-ingress](overlay/with-ingress/) - Basic **HTTP** Ingress ressource
* [overlay/with-ingress-tls-cert-manager](overlay/with-ingress-tls-cert-manager/) - Basic **HTTPS** Ingress ressource which uses [cert-manager](https://github.com/cert-manager/cert-manager) to create certificates.

To install with the overlay **just change dirctory in the overlay folder you want to install** and hit `kubectl apply -k .` .
