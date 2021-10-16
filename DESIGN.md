## Design

To run pods on every nodes, kubernetes provides  [daemonset](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/)
But, what if you want to run `jobs` on every nodes, which would make it easier and get better status overview of your workloads.


`As pods successfully complete, the Job tracks the successful completions. When a specified number of successful completions is reached, the task (ie, Job) is complete`

If you run `pods` using `daemonset`, that works, but it's not well designed.
Imagine you need a `Job` that runs on all your nodes, and you need to control the status (failed jobs, clenaup etc... and this is provided by the native `Job` resource out of the box.)


A typical use case could be:
- backups
- configuration
- security analysis
- get some information from your nodes
- restore 
- etc..



#### DaemonJob

DaemonJob is the concept to run ephemeral K8s jobs on every node.

A Developer could create a DaemonJob resource using K8s API.

A `DaemonJob` resource specify a `jobTemplate` to create `Jobs` from it.

Example: (pre-alpha)

```yaml
apiVersion: daemon.justk8s.com/v1alpha1
kind: DaemonJob
metadata:
  name: daemonjob-sample
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: hello
              image: busybox:stable
              imagePullPolicy: IfNotPresent
              command:
                - /bin/sh
                - -c
                - date; echo Hello from the Kubernetes cluster v2
          restartPolicy: OnFailure
```

#### Features

###### DaemonJob mutable job

When you create a `DaemonJob` it will spinup jobs on every nodes, but if you change somerthing in the template and apply it again,
the reconciler realize that the job is already created!

If the developer wants to force this mechanism, he could do so by specifying `spec.replace: true` (by default is false)

the reconciler, then attempt to delete all jobs and create new ones, form provided `jobTemplate`.

**Ideas** Create a hash of `jobTemplate` and save it in the `annotation`, if `spec.replace` is set to true,
the reconciler replace the jobs and eventually delete old jobs.   (as upsteam k8s does)

```yaml
metadata:
  ...
  annotations:
     daemon.justk8s.com/template-hash: ""

spec:
  replace: true
  jobTemplate:
    ...
```  

###### DaemonJob selector


Despite that, `DaemonJob` should run on every nodes, in some circumstances we need to run the jobs on specific set of nodes.

For instance, 

- Given that we need to run a job for only the `linux` nodes?

- Given that jobs should run on specific node pool!

- Given that jobs should not run on some nodes(maybe the nodes that runs ingress controller)!


`selector`, `ingoreSelector` Will implement this feature, and they are mutually exclusive. 

```yaml
spec:
  selector:
     os: linux
  jobTemplate:
    ...
```  


```yaml
spec:
  ingoreSelector:
     ingress: true
  jobTemplate:
    ...
```



in this case only  `selector` is taking effect.
```yaml
spec:
  selector:
     os: linux
  ingoreSelector:
     ingress: true
  jobTemplate:
    ...
```  
