# Get Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

## Prerequisites

- Read [Set up Cluster Federation with Kubefed](https://kubernetes.io/docs/tasks/federation/set-up-cluster-federation-kubefed/) and understand how to set up a Federation.
- This document assumes that you have a federation set up with one or more managed clusters. Note that when you start the federation controller plane you need to disable the default Federated Ingress Controller that comes with it (see more details in the "Start Federation Controller Plane" section below). You can manually create managed clusters and join them to the Federation, or alternatively you can use the [cluster-manager](https://github.com/oracle/cluster-manager) to automatically provision and join the managed clusters to the Federation.
- The managed clusters must have an ingress controller installed, for example, you can install Nginx Ingress Controller. The ingress controller is required at the cluster level because it will be used to set the Status.LoadBalancer.Ingress field on the ingress object with its external load balancer IP. 
- Set up a hosted zone for the domain where the DNS records will be published. You can set up a DNS server of your choice such as aws-route53, coredns, dyndns, and google-clouddns.


## Deploy

### Start Federation Controller Plane

When starting the Federation Controller Plane, disable the default Federated Ingress Controller that comes with it. In the arguments passed to the `kubefed init` command, pass --controllermanager-arg-overrides="--controllers=ingresses=false". For example:

```
 --controllermanager-arg-overrides="--v=9,--cluster-monitor-period=5s,--controllers=ingresses=false"
```

Make sure that the mananged clusters of the Federation has an ingress controller installed. For example,

```
helm init
sleep 30 # wait for tiller pod to be active
helm install stable/nginx-ingress --set controller.publishService.enabled=true
```

### Install the Federated Ingress Controller

1. The Federated Ingress Controller is installed on the Federation host, so set your *kubectl* config context to the Federation host context.
    ```
    kubectl config use-context <your federation host cluster>
    ```
2. Override the defaults in helm/federated-ingress-controller/values.yaml in the helm chart.
    * `dnsProvider`: set to the DNS provider name that you want to use for publishing ingress endpoints (the default is aws-route53). If this DNS provider requires a configuration file, set the values in the corresponding struct. For example, for coredns, you would need to set the `etcdEndpoints` and `zones` of the `coredns` struct.
	*  `federationEndpoint`: set to your Federation API server endpoint.
	* `federationContext`: set to your Federation name.
	* `federationHostNamespace`: set to the namespace where Federation Controller Plane was started via `kubefed init`.
	* `domain`: this is where the ingress endpoints are published. Set this to the zone that the DNS provider is hosting.
	* `image`: set `respository` to where the Federated Ingress Controller image is published, and `tag` to the desired tag/version of the image.

3. Install the Federated Ingress Controller.
    ```
    helm init
    sleep 30 # wait for tiller pod to be active
    helm install --name federated-ingress-controller helm/federated-ingress-controller/
    ```

## Verify your deployment

1. Verify if the Federated Ingress Controller is installed: 
    ```
    helm ls
    ```

2. Verify if the federated ingress controller pod is running and check the log:
    ```
    kubectl --context=<your federation host cluster> get pods -n <federation control plane namespace>
    kubectl --context=<your federation host cluster> logs <pod name ie federated-ingress-controler-xxxxxx> -n <federation control plane namespace>
    ```

## Example to use the Federated Ingress Controller

These instructions assumes that you have a federation set up with one or more managed clusters and Federated Ingress Controller installed. With this example, you can create a Federated Ingress resource with the Kubenetes cluster of your choice (AWS, GKE, Minikube etc) to route requests to your application.

1. Set the kubectl context to the federation context.
	```
	kubectl config use-context <your federation>
	```

2. Create a simple application deployment and service. For example, you can refer to these file samples in the [example](../example) folder.
	```
	kubectl create namespace akube-app
	kubectl create -f example/helloworld-deployment.yaml -n akube-app
	kubectl create -f example/helloworld-service.yaml -n akube-app
	```

3. Create an ingress resource for this service.
	```
	kubectl create -f example/helloworld-ingress.yaml -n akube-app
	```

	Wait for the Federated Ingress Controller to set the aggregated value on the *Status.LoadBalancer.Ingress* field of the Federated Ingress. The value of the field are external load balancer IP addresses or host names, which will be used to access the application.
	```
	kubectl get ingress -n akube-app
	```

	Check the Federated Ingress Controller log to see if the DNS records are created at the cluster and federation levels.
	```
	kubectl --context=<your federation host cluster> logs --tail 1000 -f <pod name ie federated-ingress-controler-xxxxxx> -n <federation control plane namespace>
	```
	You can also go to your DNS provider console to check the DNS records. For example,
	```
	*helloworld.akube-app.akube-system.ing.example.com.*
	```
	Where *helloworld* is the service name, *akube-app* is the namespace, *akube-system* is the Federation system namespace, *ing* is the defualt ingress DNS suffix, and *example.com* is the domain name.
	
4. Access the application through DNS records.
    
	You can use the *dig* command to show the IP address for the DNS record name
	```
	dig +short helloworld.akube-app.akube-system.ing.example.com.
	```
	
	You can use the *links* command or browser to access the application 
	```
	links helloworld.akube-app.akube-system.ing.example.com.
	```
5. (Optional) Delete the ingress and the application when you are done with the example.
	```
	kubectl delete -f helloworld-ingress.yaml -n akube-app
	kubectl delete -f helloworld-service.yaml -n akube-app
	kubectl delete namespace akube-app
	```
6. (Optional) Uninstall the Federated Ingress Controller. 
    ```
    kubectl config use-context <your federation host cluster>
    helm delete --purge federated-ingress-controller
    ```
