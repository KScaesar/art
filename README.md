# Artifex

The key features of Artifex are:
- Message Routes Group
- Extendable: middleware support
- Universal: message-driven architecture, stream processing ...etc, use it for whatever you need.
- Adapter Lifecycle Management

## Installation go package

```shell
go get -u github.com/KScaesar/Artifex
```

## Generate code cli

```shell
go install github.com/KScaesar/Artifex/cmd/artifex@latest
```

```
artifex gen

or

artifex gen -dir {Path} -pkg {Package} -f {File} -s {Subject}
```

```
artifex -h

help: 
    artifex gen -dir  ./    -pkg  infra    -f  kafka -s  Topic
    artifex gen -dir {Path} -pkg {Package} -f {File} -s {Subject}

-dir  Generate code to dir
-f    File prefix name
-pkg  Package name
-s    Subject name
```

## Running Artifex

One example like the following:

[Example](./example/main.go)

[Go Playground
](https://go.dev/play/p/8YJ2zMCWDy6)

```go
package main

type MyMessage struct {
    Subject string
    Bytes   []byte
}

func main() {
	routeDelimiter := "/"
	getSubject := func(msg *MyMessage) string { return msg.Subject }
	mux := Artifex.NewMux[MyMessage](routeDelimiter, getSubject)

	builtInMiddleware := Artifex.MW[MyMessage]{Artifex.NewLogger(false, Artifex.LogLevelDebug)}
	mux.SetHandleError(builtInMiddleware.PrintError(getSubject))

	// When a subject cannot be found, execute the 'Default'
	mux.SetDefaultHandler(DefaultHandler)

	v1 := mux.Group("v1/").
		PreMiddleware(HandleAuth())
	v1.Handler("Hello/{user}", Hello)

	db := make(map[string]any)
	v1.Handler("UpdatedProductPrice/{brand}", UpdatedProductPrice(db))

	// Endpoints:
	// [ ".*"                             , "main.DefaultHandler"]
	// [ "v1/Hello/{user}"                , "main.Hello"]
	// [ "v1/UpdatedProductPrice/{brand}" , "main.main.UpdatedProductPrice.func4"]
	fmt.Println("Endpoints:", mux.Endpoints())

	intervalSecond := 2
	Listen(mux, intervalSecond)
}

func Listen(mux *Artifex.Mux[MyMessage], second int) {
	adapter := NewAdapter(second)
	fmt.Printf("wait %v seconds\n\n", second)
	for {
		message, err := adapter.Recv()
		if err != nil {
			return
		}

		mux.HandleMessage(message, nil)
		fmt.Println()
	}
}
```