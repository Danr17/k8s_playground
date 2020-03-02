# Contour Canary Deployments

I'm testing out the process to automate canary releases, following the process described at https://docs.flagger.app/tutorials/contour-progressive-delivery. [Flagger](https://docs.flagger.app/) and [Contour](https://projectcontour.io/) are used.

Unfortunatelly it failed for me, maybe I've done something wrong. I'll revisit it after a while.

## What is Flagger and Contour

**Flagger** is a Kubernetes operator that automates the promotion of canary deployments using Istio, Linkerd, App Mesh, NGINX, Contour or Gloo routing for traffic shifting and Prometheus metrics for canary analysis.

Most of the Service Meshes allows us to perform canary deployments, let's see how Flagger is helping in this process.

**Contour** is an open source Kubernetes ingress controller providing the control plane for the Envoy edge and service proxy.​ Contour supports dynamic configuration updates and multi-team ingress delegation out of the box while maintaining a lightweight profile.

Like Istio, Coutour uses Envoy as a service proxy. **Envoy** is a high performance C++ distributed proxy designed for single services and applications, as well as a communication bus and “universal data plane” designed for large microservice “service mesh” architectures.

I guess, what it makes **Flagger** interesting is that it implements a control loop that gradually shifts traffic to the canary while measuring key performance indicators like *HTTP requests success rate*, *requests average duration* and *pods health*. **Based on analysis of the KPIs a canary is promoted or aborted**.

Let's see how it works.

### Create a kind cluster with extraPortMappings and node-labels.

* extraPortMappings allow the local host to make requests to the Ingress controller over ports 80/443
* node-labels only allow the ingress controller to run on a specific node(s) matching the label selector

```yaml
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

result in:

```bash
Creating cluster "kind" ...
 ✓ Ensuring node image (kindest/node:v1.17.0) 🖼
 ✓ Preparing nodes 📦  
 ✓ Writing configuration 📜 
 ✓ Starting control-plane 🕹️ 
 ✓ Installing CNI 🔌 
 ✓ Installing StorageClass 💾 
Set kubectl context to "kind-kind"

$ kubectl cluster-info --context kind-kind
Kubernetes master is running at https://127.0.0.1:32769
KubeDNS is running at https://127.0.0.1:32769/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy

To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.
```

## Install Contour on the cluster

```bash
$ kubectl apply -f https://projectcontour.io/quickstart/contour.yaml
namespace/projectcontour created
serviceaccount/contour created
configmap/contour created
customresourcedefinition.apiextensions.k8s.io/ingressroutes.contour.heptio.com created
customresourcedefinition.apiextensions.k8s.io/tlscertificatedelegations.contour.heptio.com created
customresourcedefinition.apiextensions.k8s.io/httpproxies.projectcontour.io created
customresourcedefinition.apiextensions.k8s.io/tlscertificatedelegations.projectcontour.io created
serviceaccount/contour-certgen created
rolebinding.rbac.authorization.k8s.io/contour created
role.rbac.authorization.k8s.io/contour-certgen created
job.batch/contour-certgen created
clusterrolebinding.rbac.authorization.k8s.io/contour created
clusterrole.rbac.authorization.k8s.io/contour created
role.rbac.authorization.k8s.io/contour-leaderelection created
rolebinding.rbac.authorization.k8s.io/contour-leaderelection created
service/contour created
service/envoy created
deployment.apps/contour created
daemonset.apps/envoy created
```

That deploys Contour and an Envoy daemonset in the projectcontour namespace:

```bash
$ kubectl get all -n projectcontour
NAME                           READY   STATUS      RESTARTS   AGE
pod/contour-6fdb8f5445-6zskn   1/1     Running     0          89s
pod/contour-6fdb8f5445-zt4fq   1/1     Running     0          89s
pod/contour-certgen-2tstl      0/1     Completed   0          89s
pod/envoy-zfd6f                2/2     Running     0          89s

NAME              TYPE           CLUSTER-IP    EXTERNAL-IP   PORT(S)                      AGE
service/contour   ClusterIP      10.96.52.65   <none>        8001/TCP                     89s
service/envoy     LoadBalancer   10.96.25.63   <pending>     80:32379/TCP,443:31806/TCP   89s

NAME                   DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
daemonset.apps/envoy   1         1         1       1            1           <none>          89s

NAME                      READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/contour   2/2     2            2           89s

NAME                                 DESIRED   CURRENT   READY   AGE
replicaset.apps/contour-6fdb8f5445   2         2         2       89s

NAME                        COMPLETIONS   DURATION   AGE
job.batch/contour-certgen   1/1           6s         89s
```

## Install Flagger in the projectcontour namespace:

Below I'll deploy Flagger and Prometheus configured to scrape the Contour's Envoy instances.

```bash
$ kubectl apply -k github.com/weaveworks/flagger//kustomize/contour
namespace/projectcontour unchanged
customresourcedefinition.apiextensions.k8s.io/alertproviders.flagger.app created
customresourcedefinition.apiextensions.k8s.io/canaries.flagger.app created
customresourcedefinition.apiextensions.k8s.io/metrictemplates.flagger.app created
serviceaccount/flagger-prometheus created
serviceaccount/flagger created
clusterrole.rbac.authorization.k8s.io/flagger-prometheus created
clusterrole.rbac.authorization.k8s.io/flagger created
clusterrolebinding.rbac.authorization.k8s.io/flagger-prometheus created
clusterrolebinding.rbac.authorization.k8s.io/flagger created
configmap/flagger-prometheus-5hdhmkhck9 created
service/flagger-prometheus created
deployment.apps/flagger-prometheus created
deployment.apps/flagger created
```

Notice the resource difference, inspect the resource age:

```bash
$ kubectl get all -n projectcontour
NAME                                      READY   STATUS      RESTARTS   AGE
pod/contour-6fdb8f5445-6zskn              1/1     Running     0          7m4s
pod/contour-6fdb8f5445-zt4fq              1/1     Running     0          7m4s
pod/contour-certgen-2tstl                 0/1     Completed   0          7m4s
pod/envoy-zfd6f                           2/2     Running     0          7m4s
pod/flagger-679d8d69b9-w8mx8              1/1     Running     0          2m45s
pod/flagger-prometheus-7668bb9c97-28dgb   1/1     Running     0          2m45s

NAME                         TYPE           CLUSTER-IP      EXTERNAL-IP   PORT(S)                      AGE
service/contour              ClusterIP      10.96.52.65     <none>        8001/TCP                     7m4s
service/envoy                LoadBalancer   10.96.25.63     <pending>     80:32379/TCP,443:31806/TCP   7m4s
service/flagger-prometheus   ClusterIP      10.96.191.166   <none>        9090/TCP                     2m45s

NAME                   DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
daemonset.apps/envoy   1         1         1       1            1           <none>          7m4s

NAME                                 READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/contour              2/2     2            2           7m4s
deployment.apps/flagger              1/1     1            1           2m45s
deployment.apps/flagger-prometheus   1/1     1            1           2m45s

NAME                                            DESIRED   CURRENT   READY   AGE
replicaset.apps/contour-6fdb8f5445              2         2         2       7m4s
replicaset.apps/flagger-679d8d69b9              1         1         1       2m45s
replicaset.apps/flagger-prometheus-7668bb9c97   1         1         1       2m45s

NAME                        COMPLETIONS   DURATION   AGE
job.batch/contour-certgen   1/1           6s         7m4s
```

## Install the load tester

Install the load testing service to generate traffic during the canary analysis:

```bash
$ kubectl create ns test
$ kubectl apply -k github.com/weaveworks/flagger//kustomize/tester
service/flagger-loadtester created
deployment.apps/flagger-loadtester created
```

## Create the deployment

```bash
$ kubectl apply -k github.com/weaveworks/flagger//kustomize/podinfo
deployment.apps/podinfo created
horizontalpodautoscaler.autoscaling/podinfo created
```

**The Horizontal Pod Autoscaler** automatically scales the number of pods in a replication controller, deployment, replica set or stateful set based on observed CPU utilization (or, with custom metrics support, on some other application-provided metrics). The Horizontal Pod Autoscaler is implemented as a Kubernetes API resource and a controller. The resource determines the behavior of the controller. The controller periodically adjusts the number of replicas in a replication controller or deployment to match the observed average CPU utilization to the target specified by user.

the result:

```yaml
$ kubectl get deployment podinfo -o yaml -n test
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "1"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{},"labels":{"app":"podinfo"},"name":"podinfo","namespace":"test"},"spec":{"minReadySeconds":5,"progressDeadlineSeconds":60,"revisionHistoryLimit":5,"selector":{"matchLabels":{"app":"podinfo"}},"strategy":{"rollingUpdate":{"maxUnavailable":1},"type":"RollingUpdate"},"template":{"metadata":{"annotations":{"prometheus.io/port":"9797","prometheus.io/scrape":"true"},"labels":{"app":"podinfo"}},"spec":{"containers":[{"command":["./podinfo","--port=9898","--port-metrics=9797","--grpc-port=9999","--grpc-service-name=podinfo","--level=info","--random-delay=false","--random-error=false"],"env":[{"name":"PODINFO_UI_COLOR","value":"#34577c"}],"image":"stefanprodan/podinfo:3.1.0","imagePullPolicy":"IfNotPresent","livenessProbe":{"exec":{"command":["podcli","check","http","localhost:9898/healthz"]},"initialDelaySeconds":5,"timeoutSeconds":5},"name":"podinfod","ports":[{"containerPort":9898,"name":"http","protocol":"TCP"},{"containerPort":9797,"name":"http-metrics","protocol":"TCP"},{"containerPort":9999,"name":"grpc","protocol":"TCP"}],"readinessProbe":{"exec":{"command":["podcli","check","http","localhost:9898/readyz"]},"initialDelaySeconds":5,"timeoutSeconds":5},"resources":{"limits":{"cpu":"2000m","memory":"512Mi"},"requests":{"cpu":"100m","memory":"64Mi"}}}]}}}}
  creationTimestamp: "2020-03-02T09:22:59Z"
  generation: 2
  labels:
    app: podinfo
  name: podinfo
  namespace: test
  resourceVersion: "4148"
  selfLink: /apis/apps/v1/namespaces/test/deployments/podinfo
  uid: a819717a-1919-4c47-9595-9a55c1fe1438
spec:
  minReadySeconds: 5
  progressDeadlineSeconds: 60
  replicas: 2
  revisionHistoryLimit: 5
  selector:
    matchLabels:
      app: podinfo
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      annotations:
        prometheus.io/port: "9797"
        prometheus.io/scrape: "true"
      creationTimestamp: null
      labels:
        app: podinfo
    spec:
      containers:
      - command:
        - ./podinfo
        - --port=9898
        - --port-metrics=9797
        - --grpc-port=9999
        - --grpc-service-name=podinfo
        - --level=info
        - --random-delay=false
        - --random-error=false
        env:
        - name: PODINFO_UI_COLOR
          value: '#34577c'
        image: stefanprodan/podinfo:3.1.0
        imagePullPolicy: IfNotPresent
        livenessProbe:
          exec:
            command:
            - podcli
            - check
            - http
            - localhost:9898/healthz
          failureThreshold: 3
          initialDelaySeconds: 5
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 5
        name: podinfod
        ports:
        - containerPort: 9898
          name: http
          protocol: TCP
        - containerPort: 9797
          name: http-metrics
          protocol: TCP
        - containerPort: 9999
          name: grpc
          protocol: TCP
        readinessProbe:
          exec:
            command:
            - podcli
            - check
            - http
            - localhost:9898/readyz
          failureThreshold: 3
          initialDelaySeconds: 5
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 5
        resources:
          limits:
            cpu: "2"
            memory: 512Mi
          requests:
            cpu: 100m
            memory: 64Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
status:
  availableReplicas: 2
  conditions:
  - lastTransitionTime: "2020-03-02T09:23:18Z"
    lastUpdateTime: "2020-03-02T09:23:18Z"
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  - lastTransitionTime: "2020-03-02T09:22:59Z"
    lastUpdateTime: "2020-03-02T09:23:29Z"
    message: ReplicaSet "podinfo-7c84d8c94d" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  observedGeneration: 2
  readyReplicas: 2
  replicas: 2
  updatedReplicas: 2
  ```

  and the horizontal pod autoscaler:

  ```yaml
  $ kubectl get horizontalpodautoscaler podinfo -o yaml -n test
apiVersion: autoscaling/v1
kind: HorizontalPodAutoscaler
metadata:
  annotations:
    autoscaling.alpha.kubernetes.io/conditions: '[{"type":"AbleToScale","status":"True","lastTransitionTime":"2020-03-02T09:23:15Z","reason":"SucceededGetScale","message":"the
      HPA controller was able to get the target''s current scale"},{"type":"ScalingActive","status":"False","lastTransitionTime":"2020-03-02T09:23:30Z","reason":"FailedGetResourceMetric","message":"the
      HPA was unable to compute the replica count: unable to get metrics for resource
      cpu: unable to fetch metrics from resource metrics API: the server could not
      find the requested resource (get pods.metrics.k8s.io)"}]'
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"autoscaling/v2beta1","kind":"HorizontalPodAutoscaler","metadata":{"annotations":{},"name":"podinfo","namespace":"test"},"spec":{"maxReplicas":4,"metrics":[{"resource":{"name":"cpu","targetAverageUtilization":99},"type":"Resource"}],"minReplicas":2,"scaleTargetRef":{"apiVersion":"apps/v1","kind":"Deployment","name":"podinfo"}}}
  creationTimestamp: "2020-03-02T09:22:59Z"
  name: podinfo
  namespace: test
  resourceVersion: "4153"
  selfLink: /apis/autoscaling/v1/namespaces/test/horizontalpodautoscalers/podinfo
  uid: 898c3d7b-e799-4242-870b-01647a24a71f
spec:
  maxReplicas: 4
  minReplicas: 2
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: podinfo
  targetCPUUtilizationPercentage: 99
status:
  currentReplicas: 2
  desiredReplicas: 2
  lastScaleTime: "2020-03-02T09:23:15Z"
  ```

  ## Create a Canary custom resource 

```yaml
apiVersion: flagger.app/v1alpha3
kind: Canary
metadata:
  name: podinfo
  namespace: test
spec:
  # deployment reference
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: podinfo
  # HPA reference
  autoscalerRef:
    apiVersion: autoscaling/v2beta1
    kind: HorizontalPodAutoscaler
    name: podinfo
  service:
    # service port
    port: 80
    # container port
    targetPort: 9898
    # Contour request timeout
    timeout: 15s
    # Contour retry policy
    retries:
      attempts: 3
      perTryTimeout: 5s
  # define the canary analysis timing and KPIs
  analysis:
    # schedule interval (default 60s)
    interval: 30s
    # max number of failed metric checks before rollback
    threshold: 5
    # max traffic percentage routed to canary
    # percentage (0-100)
    maxWeight: 50
    # canary increment step
    # percentage (0-100)
    stepWeight: 5
    # Contour Prometheus checks
    metrics:
      - name: request-success-rate
        # minimum req success rate (non 5xx responses)
        # percentage (0-100)
        threshold: 99
        interval: 1m
      - name: request-duration
        # maximum req duration P99 in milliseconds
        threshold: 500
        interval: 30s
    # testing
    webhooks:
      - name: acceptance-test
        type: pre-rollout
        url: http://flagger-loadtester.test/
        timeout: 30s
        metadata:
          type: bash
          cmd: "curl -sd 'test' http://podinfo-canary.test/token | grep token"
      - name: load-test
        url: http://flagger-loadtester.test/
        type: rollout
        timeout: 5s
        metadata:
          cmd: "hey -z 1m -q 10 -c 2 -host app.example.com http://envoy.projectcontour"
```

Apply the config:

```bash
$ kubectl apply -f ./podinfo-canary.yaml
canary.flagger.app/podinfo created
```

The canary analysis will run for five minutes while validating the HTTP metrics and rollout hooks every half a minute.  
Checkout the canary objects created:

```bash
$ kubectl get all -n test
NAME                                      READY   STATUS    RESTARTS   AGE
pod/flagger-loadtester-57f545d677-jnf7r   1/1     Running   0          67m
pod/podinfo-primary-69fd97bd48-4bnv9      1/1     Running   0          2m13s
pod/podinfo-primary-69fd97bd48-vpkrt      1/1     Running   0          2m13s

NAME                         TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
service/flagger-loadtester   ClusterIP   10.96.80.100    <none>        80/TCP    67m
service/podinfo              ClusterIP   10.96.200.118   <none>        80/TCP    73s
service/podinfo-canary       ClusterIP   10.96.155.118   <none>        80/TCP    2m13s
service/podinfo-primary      ClusterIP   10.96.7.228     <none>        80/TCP    2m13s

NAME                                 READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/flagger-loadtester   1/1     1            1           67m
deployment.apps/podinfo              0/0     0            0           65m
deployment.apps/podinfo-primary      2/2     2            2           2m13s

NAME                                            DESIRED   CURRENT   READY   AGE
replicaset.apps/flagger-loadtester-57f545d677   1         1         1       67m
replicaset.apps/podinfo-7c84d8c94d              0         0         0       65m
replicaset.apps/podinfo-primary-69fd97bd48      2         2         2       2m13s

NAME                                                  REFERENCE                    TARGETS         MINPODS   MAXPODS   REPLICAS   AGE
horizontalpodautoscaler.autoscaling/podinfo           Deployment/podinfo           <unknown>/99%   2         4         0          65m
horizontalpodautoscaler.autoscaling/podinfo-primary   Deployment/podinfo-primary   <unknown>/99%   2         4         2          73s

NAME                         STATUS        WEIGHT   LASTTRANSITIONTIME
canary.flagger.app/podinfo   Initialized   0        2020-03-02T10:27:38Z
```

deployment.apps/podinfo is scaled to 0 and the traffic will be routed through deployment.apps/podinfo-primary 

## Create a HTTPProxy definition and include the podinfo proxy generated by Flagger 

```bash
apiVersion: projectcontour.io/v1
kind: HTTPProxy
metadata:
  name: podinfo-ingress
  namespace: test
spec:
  virtualhost:
    fqdn: app.example.com
  includes:
    - name: podinfo
      namespace: test
      conditions:
        - prefix: /
```

Aply and check the result

```bash
kubectl apply -f ./podinfo-ingress.yaml

$ kubectl -n test get httpproxies
NAME              FQDN              TLS SECRET   STATUS   STATUS DESCRIPTION
podinfo                                          valid    valid HTTPProxy
podinfo-ingress   app.example.com                valid    valid HTTPProxy
```

## Failed rollout

Trying:

```bash
kubectl -n test set image deployment/podinfo \
podinfod=stefanprodan/podinfo:3.1.1
```

I guess I'm doing something wrong. Maybe because I'm not using a real domain and hack the app.example.com.  
However, I may have to revisit it, or try it out again, maybe with another Service Mesh. 

```bash
$ kubectl -n test describe canary/podinfo

Status:
  Canary Weight:  0
  Conditions:
    Last Transition Time:  2020-03-02T11:39:38Z
    Last Update Time:      2020-03-02T11:39:38Z
    Message:               Canary analysis failed, deployment scaled to zero.
    Reason:                Failed
    Status:                False
    Type:                  Promoted
  Failed Checks:           0
  Iterations:              0
  Last Applied Spec:       6792395483969345581
  Last Transition Time:    2020-03-02T11:39:38Z
  Phase:                   Failed
  Tracked Configs:
Events:
  Type     Reason  Age   From     Message
  ----     ------  ----  ----     -------
  Warning  Synced  4m2s  flagger  Halt advancement podinfo-primary.test waiting for rollout to finish: observed deployment generation less then desired generation
  Normal   Synced  3m2s  flagger  Initialization done! podinfo.test
  Normal   Synced  62s   flagger  New revision detected! Scaling up podinfo.test
  Warning  Synced  2s    flagger  Rolling back podinfo.test failed checks threshold reached 0
  Warning  Synced  2s    flagger  Canary failed! Scaling down podinfo.test
```

Anyway the concepts are really cool as you can customize for your own KPIs, so definitely it diserve a try.
