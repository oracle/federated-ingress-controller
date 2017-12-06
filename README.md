[![Go Report Card](https://goreportcard.com/badge/github.com/oracle/kubernetes-incubator/federated-ingress-controller)](https://goreportcard.com/report/github.com/oracle/kubernetes-incubator/federated-ingress-controller)

# Federated Ingress Round Robin DNS Controller

The Federated Ingress Round Robin DNS Controller allows you to create federated ingress resource with the Kubernetes cluster of your choice (Amazon Web Services (AWS), Google Kubernetes Engine (GKE), Minikube, etc), and route requests to your application based on the defined ingress rules.

## Before You Begin

The default Federated Ingress Controller, which is part of the [Federation project](https://github.com/kubernetes/federation) has a dependency on GKE cluster and global load balancer. So the Federated Ingress resource can work only with GKE cluster in the Federation project.

The Federated Ingress Round Robin DNS Controller removes this restriction and allows the federated ingress to work with GKE cluster as well as other types of Kubernetes clusters such as AWS and Minikube. It creates DNS records for ingress endpoints and uses DNS server to do the routing. You can choose different DNS provider such as aws-route53, coredns, dyndns, and google-clouddns as the DNS server.

<img src="./federated_ingress_controller.png" alt="Federated Ingress Controller" width="60%" height="60%"/>
<figcaption> Figure:1  Federated Ingress Round Robin DNS Controller Architecture</figcaption>
<br>
The Federated Ingress Round Robin DNS Controller architecture diagram explains how the controller distributes ingress resource to managed clusters. The cluster level ingress controller sets the *Status.LoadBalancer.Ingress* field on the cluster level ingress object with its external load balancer IP. The Federated Ingress Round Robin DNS Controller aggregates the value of the *Status.LoadBalancer.Ingress* field from managed clusters and sets the same value on the Federated Ingress resource. The DNS controller uses that value from Federated Ingress resource to create Round Robin A-records.  

## Build

Requirements:

- Go 1.8.x 

To build the Federated Ingress Controller:

get src:
- `export K8S_INCUBATOR_DIR="$GOPATH/src/github.com/kubernetes-incubator"`
- `mkdir -p $K8S_INCUBATOR_DIR`
- `cd $K8S_INCUBATOR_DIR`
- `git clone https://github.com/oracle/kubernetes-incubator/federated-ingress-controller`

build:
- `export DOCKER_REGISTRY=docker.io/changeme`
- `make` will cleanup, vet, run unit tests, build binary and image
- `make bin` will build just the controller binary so it can run locally

docker image:
- `make image` will produce a docker image containing the artifacts suitable for deploying to Kubernetes.
- `make push_image` will produce a docker image containing the artifacts suitable for deploying to Kubernetes.

Contributions to this code are welcome!  The code in this repository can be built and tested using the Makefile.


## Deploy

Follow [deploy instructions](deploy/README.md) to deploy the Federated Ingress Controller.


# Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- Slack: #sig-multicluster
- Mailing List: https://groups.google.com/forum/#!forum/sig-multicluster

## Kubernetes Incubator

This is a pending proposal for [Kubernetes Incubator project](https://github.com/kubernetes/community/blob/master/incubator.md). The incubator team for the project is:

- Sponsor: tbd
- Champion: tbd
- SIG: ~~sig-multicluster~~ tbd

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

