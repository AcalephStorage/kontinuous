kontinuous-cli
==============

A cli client for kontinuous.

## Configuration

Create configuration file. By default, `kontinuous-cli` reads the config file named `config` from the current directory.

```
Host: http://kontinuous-url
Token: github-token
Secret: base64-encoded-secret
```

## Running kontinuous-cli

Get Help. 

```
$ kontinuous-cli --help
```

Example: getting a list of pipelines.

```
$ kontinuous-cli get-pipelines
```

Deploy Kontinuous to the cluster

```
$ kontinuous-cli deploy --namespace {namespace} \
    --auth-secret {base64 encoded secret} \
    --github-client-id {github client id} \
    --github-client-secret {github client secret}
    --expose
```


Remove recently deployed Kontinuous from the cluster

```
$ kontinuous-cli deploy remove
```

## Notes

Kontinuous internal registry uses Cluster IP. Should there be any changes on the IP address, please execute the following cli command:

```
$ kontinuous-cli deploy --namespace {namespace} \
    --auth-secret {base64 encoded secret} \
    --github-client-id {github client id} \
    --github-client-secret {github client secret}
```

This is still WIP. 


