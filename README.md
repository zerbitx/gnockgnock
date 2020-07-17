# Gnock Gnock
## Who's there? 
### Whoever you'd like.

[![Go Report](https://goreportcard.com/badge/github.com/zerbitx/gnockgnock)](https://goreportcard.com/report/github.com/zerbitx/gnockgnock)
  
Live customizable server for e2e testing

### TODOs:
    1) Delays for responses to test things like timeouts 
    2) Sequences of responses? Repeated calls to the same endpoint can run through a list of responses?
    3) Cookies
    4) ETags ?
    5) Proxy to record responses and build configs
    6) Different ports
    7) HTTP/S
    
# The Idea

A server you can send a config to on the fly that interacts thusly...

```
Your test : Hey buddy if I hit this endpoint, will you send me a 409?

GnockGnock: Sure thing.

Your test: "this endpoint"

GnockGnock: 409

Your test: ✅
```

```
Your test: Hey if I hit this endpoint will you send a 200 and "You're the best!"

GnockGnock: Cool.

Your test: "this endpoint"

GnockGnock: 200: "You're the best!"

Your test: ✅
```

```
Your test: Hey will you respond to all these requests in these ways for the next 1 minute and 42 seconds

GnockGnock: K

Your test: "these endpoints in tests for 1:41"

GnockGnock: Yep, yeah, uh huh, you got it

Your test (1:43): "one of these endpoints"

GnockGnock: New config who dis?
```

# Installation

### Assuming you've cloned and you're in the repo
`go install`

### Otherwise

`go get github.com/zerbitx/gnockgnock`

# Usage 

```bash
shell1> gnockgnock
DEBU[0000]/home/your-username/go/pkg/mod/github.com/zerbitx/gnockgnock@v0.0.0-20200717014037-d4e912c66d96/gnocker/gnocker.go:296 github.com/zerbitx/gnockgnock/gnocker.(*gnocker).initConfigEndpoints() config endpoints                              GET=/gnockconfig POST=/gnockconfig gnock=gnock
INFO[0000]/home/your-username/go/pkg/mod/github.com/zerbitx/gnockgnock@v0.0.0-20200717014037-d4e912c66d96/gnocker/gnocker.go:103 github.com/zerbitx/gnockgnock/gnocker.(*gnocker).Start.func1() main                                          gnock=gnock host=127.0.0.1 port=8080
```
```bash
shell2> curl localhost:8080/gnockConfig --data-binary '@examples/example.yaml'

shell2> curl -H 'X-GNOCK-CONFIG: login401' http://gnockgnock/v1/login/dave -X POST
dave is not in the sudoers file.   This incident will be reported. 

```

# Usage with Kubernetes & kind

Add gnockgnock to your `/etc/hosts` for the ingress, then run
```bash
cd scripts
./kind-demo.sh
```

You can then run it as in the [Usage](#usage) section replacing localhost:8080 with gnockgnock.

If you set the `GNOCK_CONFIG` environment variable in your kubernetes deployment and mount a [configMap](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#add-configmap-data-to-a-volume) there, that config will be automatically loaded at container startup.