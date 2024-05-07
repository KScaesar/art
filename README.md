# art

- [Features](#Features)
- [Installation go package](#installation-go-package)
- [Why Create This Package](#why-create-this-package)
- [Usage example](#usage-example)
- [Advanced usage](#advanced-usage)
- [Class Diagram](#Class-Diagram)

## Features

- Routes Group
- Routes Parameter: `/users/{user_id}/orders/{order_id}`
- Extendable: middleware support
- Universal: message-driven architecture, stream processing ...etc, use it for whatever you need.
- Adapter Lifecycle Management


## Installation go package

```shell
go get -u github.com/KScaesar/art
```

## Why Create This Package

Simplifying Message Handling in Go.

I believe that most Go developers have used the Gin HTTP package, and my favorite part of it is the way Message HandleFunc are written.

This approach not only satisfies the Single Responsibility Principle (SRP) but also utilizes middleware design to enhance the code's extensibility, fulfilling the requirements of the Open-Closed Principle (OCP).

In everyday work, we not only handle HTTP messages but also utilize other backend common messaging methods such as Redis, RabbitMQ, WebSocket, SSE, and Kafka.

Unfortunately, I often encounter code that is difficult to maintain, written using basic switch-case or if-else statements, in my work.

In Go, these foundational open-source packages typically don't offer a built-in method to achieve HandleFunc design patterns.

Therefore, I create the message mux (multiplexer), aiming to establish a message handling pattern similar to gin's HandleFunc.

## Usage example

One example like the following:

[Example](./example/main.go)

[Go Playground
](https://go.dev/play/p/_E5wrg609Q0)

```go
package main

func main() {
	art.SetDefaultLogger(art.NewLogger(false, art.LogLevelDebug))

	routeDelimiter := "/"
	mux := art.NewMux(routeDelimiter)

	mux.ErrorHandler(art.UsePrintResult{}.PrintIngress().PostMiddleware())

	// Note:
	// Before registering handler, middleware must be defined;
	// otherwise, the handler won't be able to use middleware.
	mux.Middleware(
		art.UseRecover(),
		art.UsePrintDetail().
			Link(art.UseExclude([]string{"RegisterUser"})).
			PostMiddleware(),
		art.UseLogger(true, art.SafeConcurrency_Skip),
		art.UseHowMuchTime(),
		art.UseAdHocFunc(func(message *art.Message, dep any) error {
			logger := art.CtxGetLogger(message.Ctx, dep)
			logger.Info("    >> recv %q <<", message.Subject)
			return nil
		}).PreMiddleware(),
	)

	// When a subject cannot be found, execute the 'Default'
	mux.DefaultHandler(art.UseSkipMessage())

	v1 := mux.Group("v1/").Middleware(HandleAuth().PreMiddleware())

	v1.Handler("Hello/{user}", Hello)

	db := make(map[string]any)
	v1.Handler("UpdatedProductPrice/{brand}", UpdatedProductPrice(db))

	// Endpoints:
	// [art] subject=".*"                                f="main.main.UseSkipMessage.func11"
	// [art] subject="v1/Hello/{user}"                   f="main.Hello"
	// [art] subject="v1/UpdatedProductPrice/{brand}"    f="main.main.UpdatedProductPrice.func14"
	mux.Endpoints(func(subject, fn string) { fmt.Printf("[art] subject=%-35q f=%q\n", subject, fn) })

	intervalSecond := 2
	Listen(mux, intervalSecond)
}
```

## Advanced usage

Generate code cli is used to generate template code for `message.go` and `adapter.go`.

Modify the template content according to the requirements,  
select PubSub, Publisher, or Subscriber as needed, and delete unused code.

- [artisan](https://github.com/KScaesar/artisan?tab=readme-ov-file#art-adapter)
    - [SSE: Producer Example](https://github.com/KScaesar/artisan?tab=readme-ov-file#sse)
    - [Rabbitmq: Producer Consumer Example](https://github.com/KScaesar/artisan?tab=readme-ov-file#rabbitmq)

```shell
go install github.com/KScaesar/art/cmd/art@latest
```

```
art gen

or

art gen -dir {Path} -pkg {Package} -f {File}
```

```
art -h

help: 
    art gen -dir  ./    -pkg  infra    -f  kafka 
    art gen -dir {Path} -pkg {Package} -f {File} 

-dir  Generate code to dir
-f    File prefix name
-pkg  Package name
```

## Class Diagram

```mermaid
classDiagram
direction RL

namespace HandleMessage {
    class Message {
        <<datatype>>
        + Subject: string
    }
    class Ingress
    class Egress

    class Mux {
        + HandleMessage(Message,Dependency)
        + Register(Subject,Handler)
    }

    class Handler {
        <<interface>>
        + HandleMessage(Message,Dependency)
    }
}

namespace AdapterLayer {
    class IAdapter {
        <<interface>>
        + Identifier() string
        + Stop() 
    }

    class Consumer {
        <<interface>>
    }

    class Producer {
        <<interface>>
    }

    class Hub {
        + Join(ID,IAdapter) error
        + RemoveByKey(ID)
        + DoSync(action func(IAdapter)bool)
        + DoAsync(action func(IAdapter))
    }

    class Adapter {
        - IngressMux: Mux
        - EgressMux: Mux
        - hub: Hub

        + Identifier() string
        + Stop() 
        + Send(Message) error
        + RawSend(Message) error
        + Listen() error
    }

    class AdapterOption {
        + Buidl() IAdapter
    }
}

namespace artisan_pubsub {
    class KafkaProducer
    class RabbitMqConsumer
    class Websocket
}

    Message <|-- Ingress
    Message <|-- Egress

    Mux --o Adapter: aggregation
    Handler <|.. Mux: implement
    Handler <.. Mux: register
    Message <.. Mux: handle message
    IAdapter .. Mux: dependency is IAdapter

    IAdapter "n" --* "1" Hub: composition
    Adapter <.. AdapterOption: build

    IAdapter <|.. Adapter: implement
    Consumer <|.. Adapter: implement
    Producer <|.. Adapter: implement
    IAdapter <|.. Consumer
    IAdapter <|.. Producer

    AdapterOption <.. KafkaProducer: use
    AdapterOption <.. RabbitMqConsumer: use
    AdapterOption <.. Websocket: use
```