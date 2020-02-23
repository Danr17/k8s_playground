## Install

### Follow the steps:
https://kind.sigs.k8s.io/docs/user/ingress/  
https://github.com/jaegertracing/jaeger-operator  

### Create a kind cluster with extraPortMappings and node-labels.

* extraPortMappings allow the local host to make requests to the Ingress controller over ports 80/443
* node-labels only allow the ingress controller to run on a specific node(s) matching the label selector

```
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
        authorization-mode: "AlwaysAllow"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
EOF
```

### Install Contour

Install Contour and Apply kind specific patches to forward the hostPorts to the ingress controller, set taint tolerations and schedule it to the custom labelled node.

```
kubectl apply -f https://projectcontour.io/quickstart/contour.yaml
kubectl patch daemonsets -n projectcontour envoy -p '{"spec":{"template":{"spec":{"nodeSelector":{"ingress-ready":"true"},"tolerations":[{"key":"node-role.kubernetes.io/master","operator":"Equal","effect":"NoSchedule"}]}}}}' 
```

### Install Jaeger 

```
kubectl create namespace observability
kubectl create -f install_jaeger/jaegertracing.io_jaegers_crd.yaml
kubectl create -f install_jaeger/service_account.yaml
kubectl create -f install_jaeger/role.yaml
kubectl create -f install_jaeger/role_binding.yaml
kubectl create -f install_jaeger/operator.yaml
```

Once the jaeger-operator deployment in the namespace observability is ready, create a Jaeger instance, like:

```
kubectl apply -f - <<EOF
apiVersion: jaegertracing.io/v1
kind: Jaeger
metadata:
  name: simplest
EOF
```

At this point you should be abel to access Jager UI by localhost

```
$ kubectl get ingress
NAME             HOSTS   ADDRESS   PORTS   AGE
simplest-query   *                 80      11m
```

### Create example-hotrod deployment  

Using https://hub.docker.com/r/jaegertracing/example-hotrod

```
$ kubectl apply -f hotrod-deployment.yaml
```

The deployment looks like:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/component: hotrod
    app.kubernetes.io/instance: jaeger
  name: jaeger-hotrod
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/component: hotrod
      app.kubernetes.io/instance: jaeger
      app.kubernetes.io/name: jaeger
  template:
    metadata:
      labels:
        app.kubernetes.io/component: hotrod
        app.kubernetes.io/instance: jaeger
        app.kubernetes.io/name: jaeger
    spec:
      containers:
      - env:
        - name: JAEGER_AGENT_HOST
          value: simplest-agent.default.svc.cluster.local
        - name: JAEGER_AGENT_PORT
          value: "6831"
        image: jaegertracing/example-hotrod:latest
        imagePullPolicy: Always
        livenessProbe:
          httpGet:
            path: /
            port: 8080
        name: jaeger-hotrod
        ports:
        - containerPort: 8080
        readinessProbe:
          httpGet:
            path: /
            port: 8080
```

At this stage you should see:

```
$ kubectl get pods --all-namespaces
NAMESPACE            NAME                                         READY   STATUS      RESTARTS   AGE
default              jaeger-hotrod-85df697fc9-7jzz8               1/1     Running     0          23m
default              simplest-59875cd85-48nzn                     1/1     Running     0          65m
kube-system          coredns-6955765f44-5k7dz                     1/1     Running     0          66m
kube-system          coredns-6955765f44-vkrfg                     1/1     Running     0          66m
kube-system          etcd-kind-control-plane                      1/1     Running     0          67m
kube-system          kindnet-8vzb7                                1/1     Running     0          66m
kube-system          kube-apiserver-kind-control-plane            1/1     Running     0          67m
kube-system          kube-controller-manager-kind-control-plane   1/1     Running     0          67m
kube-system          kube-proxy-b9rg4                             1/1     Running     0          66m
kube-system          kube-scheduler-kind-control-plane            1/1     Running     0          67m
local-path-storage   local-path-provisioner-7745554f7f-92djr      1/1     Running     0          66m
observability        jaeger-operator-5cc9697959-h6sqr             1/1     Running     0          66m
projectcontour       contour-6c7b6bbbc4-hsbzr                     1/1     Running     0          66m
projectcontour       contour-6c7b6bbbc4-qxr8z                     1/1     Running     0          66m
projectcontour       contour-certgen-6c6vv                        0/1     Completed   0          66m
projectcontour       envoy-rt4sg                                  2/2     Running     0          66m
```

Forward port to container in order to access the application:

```
$ kubectl port-forward jaeger-hotrod-85df697fc9-7jzz8 8080 8080
Forwarding from 127.0.0.1:8080 -> 8080
Forwarding from [::1]:8080 -> 8080
Unable to listen on port 8080: Listeners failed to create with the following errors: [unable to create listener: Error listen tcp4 127.0.0.1:8080: bind: address already in use unable to create listener: Error listen tcp6 [::1]:8080: bind: address already in use]
Handling connection for 8080
Handling connection for 8080
Handling connection for 8080
Handling connection for 8080
```

Now you can make requests and see the traces within Jaeger. 

![Hot Rod App](hotRod.png "Hot Rod")  
![Jager](jaeger.png "Jaeger traces")