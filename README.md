# daemonjob-operator

![CI-status](https://github.com/mcbenjemaa/daemonjob-operator/actions/workflows/tests.yml/badge.svg)

Daemonjob-operator runs K8s Jobs on every nodes.

DaemonJob is the concept to run ephemeral K8s jobs on every node. (I'm using it personally, to patch node config after bootstraping a cluster)

### Getting started

You can install daemonjob using helm.

First add helm repo

```
helm repo add daemonjob-repo https://mcbenjemaa.github.io/daemonjob-operator
```

Then, install the chart.

```
helm install daemonjob daemonjob-repo/daemonjob-operator
```


you're ready, you can start creating `DaemonJob`


```
kubectl apply -f config/samples/daemonjob.yaml
```


######  check out [Design](DESIGN.md)



#### Develop

**Requirements**


* K8s cluster
* Helm3


#### Using Tilt

If you don't have tilt, go get it [here](https://tilt.dev)

To start everthing, just 

```
tilt up
```

tilt now will deploy crd/conroller/sample and ensure fast feedback loop.
When you change something, it will be detected by tilt and deployed.


to clean up 

```
tilt down
```

#### Manual 

###### Deploy the crd

```
make install
```

###### Start the controller locally

```
make run
```

###### Run tests

```
make test
```


###### Copy crd to helm/chart

```
make helm
```

###### For more details

```
make help
```
