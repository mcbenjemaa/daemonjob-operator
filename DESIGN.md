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



