# Run ddns-updater on Kubernetes

This folder contains some example Kubernetes manifests which should simplify the DDNS-Setup on a Kubernetes-Cluster. (For example in your Home-Lab).

The Manifests are plain Kubernetes manifests with addtional Kustomize overlays, which can be used to add an [ingress-route](https://kubernetes.io/docs/concepts/services-networking/ingress/) to the ddns-updater.

What is Kustomize?

*Kustomize introduces a template-free way to customize application configuration that simplifies the use of off-the-shelf applications. Now, built into kubectl as apply -k.* - [Kustomize](https://kustomize.io/)

## Basic install on Kubernetes

Requierments:

* Kubernetes Cluster e.g. [k3s](https://k3s.io/)

How To:

1. Connect to your Kubernetes-Cluster (Get your [Kubeconfig](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/))

2. Clone the Repo

    ```sh
    git clone https://github.com/qdm12/ddns-updater.git
    ```

3. Swtich Directory

    ```sh
    cd ddns-updater/k8s/base
    ```

4. Change the config to your needs in the [*secret-config.yaml*](base/secret-config.yaml) as described in the [Documentation](/README.md#configuration).

5. Apply the Manifest with [kubectl](https://kubernetes.io/docs/reference/kubectl/)

    ```sh
    kubectl apply -k .
    ```

6. Connect the the UI using a kubectl [port-forward](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/)

    ```sh
    kubectl port-forward svc/ddns-updater 8000:8080
    ```

7. Open the ddns-updater-UI: **<http://localhost:8000>**

If this do not work feel free to open an [issue](https://github.com/qdm12/ddns-updater/issues/new/choose).

## Advanced Install

### Overlays

This folder also contains kustomize overlays which can extend the basic install.

* [with-ingress](overlay/with-ingress/) - Basic **HTTP** Ingress ressource

* [with-ingress-cert-manager](overlay/with-ingress-cert-manager/) - Basic **HTTPS** Ingress ressource which uses [cert-manager](https://github.com/cert-manager/cert-manager) to create certificates.

To install with the overlay **just change dirctory in the overlay folder you want to install** and hit `kubectl apply -k .` .

### GitOps with Argo-CD

If you want to use this with Argo-CD take a look in this [Repo](https://github.com/3deep5me/argo-cd-ready-applications).
