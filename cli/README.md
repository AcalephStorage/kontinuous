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

## Notes

This is still WIP. 


