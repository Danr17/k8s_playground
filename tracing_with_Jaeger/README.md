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