package main

import (
	"fmt"
	"io"
	"time"

	"github.com/KScaesar/Artifex"
)

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

// adapter

func NewAdapter(second int) *Adapter {
	mq := make(chan *MyMessage, 1)
	size := len(messages)

	go func() {
		ticker := time.NewTicker(time.Duration(second) * time.Second)
		defer ticker.Stop()

		cursor := 0
		for {
			select {
			case <-ticker.C:
				if cursor >= size {
					close(mq)
					return
				}
				mq <- messages[cursor]
				cursor++
			}
		}
	}()

	return &Adapter{mq: mq}
}

type Adapter struct {
	mq chan *MyMessage
}

func (adp *Adapter) Recv() (msg *MyMessage, err error) {
	defer func() {
		if err == nil {
			fmt.Printf("recv message: subject=%v\n", msg.Subject)
		} else {
			fmt.Printf("recv message fail\n")
		}
	}()

	message, ok := <-adp.mq
	if !ok {
		return nil, io.EOF
	}
	return message, nil
}

// message

type MyMessage struct {
	Subject string
	Bytes   []byte
}

var messages = []*MyMessage{
	{
		Subject: "RegisterUser",
		Bytes:   []byte(`{"user_id": "123456", "username": "john_doe", "email": "john.doe@example.com", "age": 30, "country": "United States"}`),
	},
	{
		Subject: "v1/Hello/ff1017",
		Bytes:   []byte("world"),
	},
	{
		Subject: "UpdatedUser",
		Bytes:   []byte(`{"user_id": "789012", "username": "jane_smith", "email": "jane.smith@example.com", "age": 25, "country": "Canada"}`),
	},
	{
		Subject: "v1/UpdatedProductPrice/Samsung",
		Bytes:   []byte(`{"product_id": "67890", "name": "Samsung Galaxy Watch 4", "price": 349, "brand": "Samsung", "category": "Wearable Technology"}`),
	},
	{
		Subject: "UpdateLocation",
		Bytes:   []byte(`{"location_id": "002", "name": "Eiffel Tower", "city": "Paris", "country": "France", "latitude": 48.8584, "longitude": 2.2945}`),
	},
	{
		Subject: "CreatedOrder",
		Bytes:   []byte(`{"order_id": "ABC123", "customer_name": "John Smith", "total_amount": 150.75, "items": ["T-shirt", "Jeans", "Sneakers"]}`),
	},
}

// handler

func HandleAuth() Artifex.HandleFunc[MyMessage] {
	return func(message *MyMessage, route *Artifex.RouteParam) error {
		fmt.Println("Middleware: Auth Ok")
		return nil
	}
}

func DefaultHandler(message *MyMessage, _ *Artifex.RouteParam) error {
	fmt.Printf("Default: AutoAck message: subject=%v body=%v\n", message.Subject, string(message.Bytes))
	return nil
}

func Hello(message *MyMessage, route *Artifex.RouteParam) error {
	fmt.Printf("Hello: body=%v user=%v\n", string(message.Bytes), route.Get("user"))
	return nil
}

func UpdatedProductPrice(db map[string]any) func(message *MyMessage, route *Artifex.RouteParam) error {
	return func(message *MyMessage, route *Artifex.RouteParam) error {
		brand := route.Str("brand")
		db[brand] = message.Bytes
		fmt.Printf("UpdatedProductPrice: saved db: brand=%v\n", brand)
		return nil
	}
}
