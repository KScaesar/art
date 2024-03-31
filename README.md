# Artifex

The key features of Artifex are:
- Routes grouping
- Extendable: middleware support
- Universal: message-driven architecture, stream processing ...etc, use it for whatever you need.
- Lifecycle management

## Installation go package

```shell
go get -u github.com/KScaesar/Artifex
```

## generate code cli

```shell
go install github.com/KScaesar/Artifex/cmd/artifex@latest
```

```
artifex gen

or

artifex gen -dir ./example -pkg main -f sse -s Stream
```

```
artifex -h

help: 
    artifex gen -dir {Path} -pkg {Package} -f {File} -s {Subject}

-dir  Generate code to dir
-f    File prefix name
-pkg  Package name
-s    Subject name
```

## Running Artifex

First you need to generate code for using Artifex,  
one simplest example likes the follow `example.go`:  

```go

```