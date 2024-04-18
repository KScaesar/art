# Artifex

- [Features](#Features)
- [Installation go package](#installation-go-package)
- [Why Create This Package](#why-create-this-package)
- [Usage example](#usage-example)
- [Advanced usage](#advanced-usage)

## Features

- Routes Group
- Routes Parameter: `/users/{user_id}/orders/{order_id}`
- Extendable: middleware support
- Universal: message-driven architecture, stream processing ...etc, use it for whatever you need.
- Adapter Lifecycle Management


## Installation go package

```shell
go get -u github.com/KScaesar/Artifex
```

## Why Create This Package

Simplifying Message Handling in Go.

I believe that most Go developers have used the Gin HTTP package, and my favorite part of it is the way Message HandleFunc are written.

This approach not only satisfies the Single Responsibility Principle (SRP) but also utilizes middleware design to enhance the code's extensibility, fulfilling the requirements of the Open-Closed Principle (OCP).

In everyday work, we not only handle HTTP messages but also utilize other backend common messaging methods such as Redis, RabbitMQ, WebSocket, SSE, and Kafka.

Unfortunately, I often encounter code that is difficult to maintain, written using basic switch-case or if-else statements, in my work.

In Go, these foundational open-source packages typically don't offer a built-in method to achieve HandleFunc design patterns.

Therefore, I create the message mux (multiplexer) based on generics, aiming to establish a message handling pattern similar to gin's HandleFunc.

## Usage example

One example like the following:

[Example](./example/main.go)

[Go Playground
](https://go.dev/play/p/sfKJiA970Qe)

```
package main

type MyMessage struct {
    Subject string
    Bytes   []byte
}

func main() {
	routeDelimiter := "/"
	getSubject := func(msg *MyMessage) string { return msg.Subject }
	mux := Artifex.NewMux[MyMessage](routeDelimiter, getSubject)

	builtInMiddleware := Artifex.MW[MyMessage]{Logger: Artifex.NewLogger(false, Artifex.LogLevelDebug)}
	mux.SetHandleError(builtInMiddleware.PrintError(getSubject))

	// Note:
	// Before registering handler, middleware must be defined;
	// otherwise, the handler won't be able to use middleware.
	mux.Middleware(builtInMiddleware.Recover())

	// When a subject cannot be found, execute the 'Default'
	mux.SetDefaultHandler(DefaultHandler)

	v1 := mux.Group("v1/").Middleware(HandleAuth().PreMiddleware())

	v1.Handler("Hello/{user}", Hello)

	db := make(map[string]any)
	v1.Handler("UpdatedProductPrice/{brand}", UpdatedProductPrice(db))

	// Endpoints:
	// [Artifex] subject=".*"                                f="main.DefaultHandler"
	// [Artifex] subject="v1/Hello/{user}"                   f="main.Hello"
	// [Artifex] subject="v1/UpdatedProductPrice/{brand}"    f="main.main.UpdatedProductPrice.func5"
	mux.PrintEndpoints(func(subject, fn string) { fmt.Printf("[Artifex] subject=%-35q f=%q\n", subject, fn) })

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

// type HandleFunc[Message any] func(message *Message, route *RouteParam) error

func UpdatedProductPrice(db map[string]any) Artifex.HandleFunc[MyMessage] {
	return func(message *MyMessage, route *Artifex.RouteParam) error {
		brand := route.Str("brand")
		db[brand] = message.Bytes
		fmt.Printf("UpdatedProductPrice: saved db: brand=%v\n", brand)
		return nil
	}
}
```

## Advanced usage

Generate code cli is used to generate template code for `message.go` and `adapter.go`.

Modify the template content according to the requirements,  
select PubSub, Publisher, or Subscriber as needed, and delete unused code.

- [Artifex-Adapter](https://github.com/KScaesar/Artifex-Adapter?tab=readme-ov-file#artifex-adapter)
    - [SSE: Publisher Example](https://github.com/KScaesar/Artifex-Adapter?tab=readme-ov-file#sse)
    - [Rabbitmq: Publisher Subscriber Example](https://github.com/KScaesar/Artifex-Adapter?tab=readme-ov-file#rabbitmq)

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
