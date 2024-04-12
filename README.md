# Artifex

The key features of Artifex are:
- Routes Group
- Routes Parameter: `/users/{user_id}/orders/{order_id}`
- Extendable: middleware support
- Universal: message-driven architecture, stream processing ...etc, use it for whatever you need.
- Adapter Lifecycle Management

## Why Create This Package: 

Simplifying Message Handling in Go.

I believe that most Go developers have used the Gin HTTP package, and my favorite part of it is the way Message HandleFunc are written.

This approach not only adheres to the Single Responsibility Principle (SRP) but also utilizes middleware design to make the code more extensible, meeting the criteria of the Open-Closed Principle (OCP).

In everyday work, we not only handle HTTP messages but also utilize other backend common messaging methods such as Redis, RabbitMQ, WebSocket, SSE, and Kafka.

Unfortunately, I often encounter code that is difficult to maintain, written using basic switch-case or if-else statements, in my work.

```
for {
	message, err := adapter.Recv()
	if err != nil {
		return
	}

	subject:= getSubject(message)
	switch subject {
	case "topic1":
		handleTopic1()
	case "channel2":
		handleChannel2()
	case "routingKey3":
		handleRoutingKey3()
	case "eventType4":
		handleWebsocketEventType4()
	...
	}
}
```

In Go, these foundational open-source packages typically don't offer a built-in method to achieve HandleFunc design patterns.

Therefore, I developed a message mux (multiplexing) based on generics,  
hoping to transform message handling from any adapter into a pattern similar to Gin's HandleFunc.

## Installation go package

```shell
go get -u github.com/KScaesar/Artifex
```

## Usage example

One example like the following:

[Example](./example/main.go)

[Go Playground
](https://go.dev/play/p/q-vu3_d8Ws7)

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

## Advanced usage

Generate code cli is used to generate template code for `message.go` and `adapter.go`.

Modify the template content according to the requirements,  
select PubSub, Publisher, or Subscriber as needed, and delete unused code.

- [Publisher SSE Adapter Example](https://github.com/KScaesar/Artifex-Adapter?tab=readme-ov-file#sse)


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
