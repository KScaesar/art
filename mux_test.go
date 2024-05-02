package art

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"
)

func TestMessageMux_HandleMessage(t *testing.T) {
	// arrange
	recorder := &bytes.Buffer{}

	mux := NewMux("").
		Handler("hello", func(message *Message, dep any) error {
			fmt.Fprintf(recorder, "topic=%v, payload=%v", message.Subject, string(message.Bytes))
			return nil
		}).
		Handler("foo", func(message *Message, dep any) error {
			fmt.Fprintf(recorder, "topic=%v, payload=%v", message.Subject, string(message.Bytes))
			return nil
		})

	// expect
	expected := `topic=hello, payload={"data":"world"}`

	// paramHandler
	message := &Message{
		Bytes:   []byte(`{"data":"world"}`),
		Subject: "hello",
	}
	err := mux.HandleMessage(message, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// assert
	if expected != recorder.String() {
		t.Errorf("handleMessage(): %v, but want: %v", recorder.String(), expected)
	}
}

func TestLinkMiddlewares_when_wildcard(t *testing.T) {
	recorder := []string{}
	mux := NewMux(".")

	v1 := mux.Group("v1")

	v1.
		PreMiddleware(func(message *Message, dep any) error {
			message.Bytes = append([]byte("! "), message.Bytes...)
			return nil
		}).
		Handler(".{kind}.book.{book_id}", func(message *Message, dep any) error {
			data := append(message.Bytes, []byte(" {kind}")...)
			recorder = append(recorder, string(data))
			return nil
		})

	v1.Group(".dev.book").
		PreMiddleware(func(message *Message, _ any) error {
			message.Bytes = append([]byte("* "), message.Bytes...)
			return nil
		}).
		Handler(".discount", func(message *Message, _ any) error {
			recorder = append(recorder, string(message.Bytes))
			return nil
		})

	expectedResponse := []string{
		"! v1.devops.book.6263334908 {kind}",
		"! v1.dev.book.1449373321 {kind}",
		"* ! v1.dev.book.discount",
	}

	messages := []*Message{
		{Subject: "v1.devops.book.6263334908", RouteParam: map[string]any{}},
		{Subject: "v1.dev.book.1449373321", RouteParam: map[string]any{}},
		{Subject: "v1.dev.book.discount", RouteParam: map[string]any{}},
	}

	for i, message := range messages {
		message.Bytes = []byte(message.Subject)
		err := mux.HandleMessage(message, nil)
		if err != nil {
			t.Errorf("%v: unexpected error: got %v", message.Subject, err)
			break
		}
		if recorder[i] != expectedResponse[i] {
			t.Errorf("%v: unexpected output: got %s, want %s", message.Subject, recorder[i], expectedResponse[i])
			break
		}
	}

}

func TestLink(t *testing.T) {
	buf := new(bytes.Buffer)
	buf.WriteString("\n")

	decorator1 := func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			fmt.Fprintf(buf, "%s_decorator1\n", message.Subject)

			err := next(message, dep)
			if err != nil {
				return err
			}

			fmt.Fprintf(buf, "%s_decoratorA\n", message.Subject)
			return nil
		}
	}

	decorator2 := func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			fmt.Fprintf(buf, "%s_decorator2\n", message.Subject)

			err := next(message, dep)
			if err != nil {
				return err
			}

			fmt.Fprintf(buf, "%s_decoratorB\n", message.Subject)
			return nil
		}
	}

	decorator3 := func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			fmt.Fprintf(buf, "%s_decorator3\n", message.Subject)

			err := next(message, dep)
			if err != nil {
				return err
			}

			fmt.Fprintf(buf, "%s_decoratorC\n", message.Subject)
			return nil
		}
	}

	tests := []struct {
		name        string
		msg         *Message
		middlewares []Middleware
		expected    string
	}{
		{
			name:        "Multiple decorator check sequence",
			msg:         &Message{Subject: "hello_world"},
			middlewares: []Middleware{decorator1, decorator2, decorator3},
			expected: `
hello_world_decorator1
hello_world_decorator2
hello_world_decorator3
hello_world_base
hello_world_decoratorC
hello_world_decoratorB
hello_world_decoratorA
`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			baseFunc := func(msg *Message, dep any) error {
				fmt.Fprintf(buf, "%s_base\n", msg.Subject)
				return nil
			}

			handler := Link(baseFunc, tt.middlewares...)
			err := handler(tt.msg, nil)
			if err != nil {
				t.Errorf("unexpected error: got %v", err)
			}
			if got := buf.String(); got != tt.expected {
				t.Errorf("unexpected output: got %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestMessageMux_Transform_when_only_defaultHandler(t *testing.T) {
	mux := NewMux("/")

	mux.DefaultHandler(func(message *Message, dep any) error { return nil })

	message := &Message{}

	err := mux.HandleMessage(message, nil)
	if err != nil {
		t.Errorf("%#v :unexpected error: got %v", message, err)
	}
}

func TestMessageMux_Transform(t *testing.T) {
	const (
		levelKey = "lv"
	)

	recorder := []string{}
	record := func(message *Message, dep any) error {
		recorder = append(recorder, message.Subject)
		return nil
	}

	mux := NewMux("/")

	mux.HandlerByNumber(2, record)

	mux.GroupByNumber(3).Transform(func(message *Message, dep any) error {
		message.Subject += (string(message.Bytes) + "/")
		return nil
	}).
		HandlerByNumber(1, record).
		HandlerByNumber(2, record).
		GroupByNumber(5).Transform(func(message *Message, dep any) error {
		message.Subject += (message.Metadata.Str(levelKey) + "/")
		return nil
	}).
		PreMiddleware(
			func(message *Message, dep any) error {
				message.Subject = "^" + message.Subject + "^"
				return nil
			}).
		HandlerByNumber(1, record).
		HandlerByNumber(2, record)

	mux.GroupByNumber(4).Transform(func(message *Message, dep any) error {
		message.Subject += (string(message.Bytes) + "/")
		return nil
	}).
		PreMiddleware(
			func(message *Message, dep any) error {
				message.Subject = "_" + message.Subject + "_"
				return nil
			}).
		HandlerByNumber(1, record).
		HandlerByNumber(2, record).
		HandlerByNumber(4, record)

	expectedSubjects := []string{
		"2/",
		"3/1/", "3/2/",
		"3/5/1/", "3/5/2/",
		"4/1/", "4/2/", "4/4/",
	}

	i := 0
	mux.Endpoints(func(subject, handler string) {
		if subject != expectedSubjects[i] {
			t.Errorf("unexpected output: got %s, want %s", subject, expectedSubjects[i])
		}
		i++
	})

	expectedRecords := []string{
		"2/",
		"3/1/",
		"^3/5/1/^",
		"_4/2/_",
	}

	messages := []*Message{
		{
			Subject: strconv.Itoa(2) + "/",
			Bytes:   nil,
		},
		{
			Subject: strconv.Itoa(3) + "/",
			Bytes:   []byte("1"),
		},
		{
			Subject:  strconv.Itoa(3) + "/",
			Bytes:    []byte("5"),
			Metadata: map[string]any{levelKey: "1"},
		},
		{
			Subject: strconv.Itoa(4) + "/",
			Bytes:   []byte("2"),
		},
	}

	for i, message := range messages {
		err := mux.HandleMessage(message, nil)
		if err != nil {
			t.Errorf("%#v :unexpected error: got %v", message, err)
			break
		}
		if recorder[i] != expectedRecords[i] {
			t.Errorf("%#v :unexpected output: got %s, want %s", message, recorder[i], expectedRecords[i])
			break
		}
	}
}

func TestMessageMux_Group(t *testing.T) {
	recorder := []string{}
	record := func(message *Message, dep any) error {
		recorder = append(recorder, string(message.Bytes))
		return nil
	}

	mux := NewMux("/")

	mux.PreMiddleware(func(message *Message, dep any) error {
		message.Bytes = []byte(fmt.Sprintf("*%s*", message.Bytes))
		return nil
	})

	mux.Handler("/topic1", record)

	mux.Group("/topic2").
		Handler("/orders", record).
		Handler("/users", record)

	mux.Group("/topic3").
		Handler("/created/orders", record).
		Handler("/login/users", record).
		Handler("/upgraded.users", record)

	topic4 := mux.Group("/topic4").
		Handler("/game1", record).
		Handler("/game2/kindA", record)
	topic4_game3 := topic4.Group("/game3").
		PreMiddleware(func(message *Message, dep any) error {
			message.Bytes = []byte(fmt.Sprintf("&%s&", message.Bytes))
			return nil
		}).
		Handler("/kindX", record).
		Handler("/kindY", record)
	topic4_game3.Group("/v2").
		Handler("/kindX", record)
	topic4_game3.Group("/v3").
		Handler("/kindY", record)

	mux.Handler("/topic5/game1", record)
	mux.Handler("/topic5/game2/kindA", record)
	mux.Handler("/topic5/game3/kindX", record)
	mux.Handler("/topic5/game3/kindY", record)
	mux.Handler("/topic5/game3/v2/kindX", record)
	mux.Handler("/topic5/game3/v3/kindY", record)

	expectedSubjects := []string{
		"/topic1",
		"/topic2/orders", "/topic2/users",
		"/topic3/created/orders", "/topic3/login/users", "/topic3/upgraded.users",
		"/topic4/game1", "/topic4/game2/kindA", "/topic4/game3/kindX", "/topic4/game3/kindY", "/topic4/game3/v2/kindX", "/topic4/game3/v3/kindY",
		"/topic5/game1", "/topic5/game2/kindA", "/topic5/game3/kindX", "/topic5/game3/kindY", "/topic5/game3/v2/kindX", "/topic5/game3/v3/kindY",
	}

	i := 0
	mux.Endpoints(func(subject, handler string) {
		if subject != expectedSubjects[i] {
			t.Errorf("unexpected output: got %s, want %s", subject, expectedSubjects[i])
		}
		i++
	})

	subjects := []string{
		"/topic3/upgraded.users",
		"/topic4/game3/kindX",
		"/topic5/game3/kindY",
		"/topic4/game3/kindY",
		"/topic2/users",
		"/topic3/created/orders",
		"/topic5/game3/v2/kindX",
		"/topic5/game2/kindA",
		"/topic3/login/users",
		"/topic1",
		"/topic4/game3/v3/kindY",
		"/topic5/game3/kindX",
		"/topic5/game1",
		"/topic4/game1",
		"/topic4/game3/v2/kindX",
		"/topic2/orders",
		"/topic4/game2/kindA",
	}

	for _, subject := range subjects {
		message := &Message{
			Subject: subject,
			Bytes:   []byte(subject),
		}
		err := mux.HandleMessage(message, nil)
		if err != nil {
			t.Errorf("unexpected error: got %v", err)
			break
		}
	}

	expectedRecords := []string{
		"*/topic3/upgraded.users*",
		"&*/topic4/game3/kindX*&",
		"*/topic5/game3/kindY*",
		"&*/topic4/game3/kindY*&",
		"*/topic2/users*",
		"*/topic3/created/orders*",
		"*/topic5/game3/v2/kindX*",
		"*/topic5/game2/kindA*",
		"*/topic3/login/users*",
		"*/topic1*",
		"&*/topic4/game3/v3/kindY*&",
		"*/topic5/game3/kindX*",
		"*/topic5/game1*",
		"*/topic4/game1*",
		"&*/topic4/game3/v2/kindX*&",
		"*/topic2/orders*",
		"*/topic4/game2/kindA*",
	}

	for i, gotRecord := range recorder {
		if gotRecord != expectedRecords[i] {
			t.Errorf("unexpected output: got %s, want %s", gotRecord, expectedRecords[i])
		}
	}
}

func TestMux_DefaultHandler(t *testing.T) {
	isCalled := false
	mux := NewMux("/").
		DefaultHandler(func(message *Message, dep any) error {
			isCalled = true
			return nil
		})

	msg := &Message{}
	err := mux.HandleMessage(msg, nil)
	if err != nil || isCalled != true {
		t.Errorf("unexpected error: got %v", err)
	}
}

func TestMux_SetDefaultHandler_when_wildcard(t *testing.T) {
	recorder := []string{}
	mux := NewMux(".").
		Middleware(UseRecover()).
		DefaultHandler(func(message *Message, dep any) error {
			recorder = append(recorder, string(message.Bytes)+" default")
			return nil
		})

	v1 := mux.Group("v1")

	v1.
		PreMiddleware(func(message *Message, dep any) error {
			message.Bytes = []byte(fmt.Sprintf("! %s", message.Bytes))
			return nil
		}).
		Handler(".{kind}.book.{book_id}", func(message *Message, dep any) error {
			recorder = append(recorder, string(message.Bytes)+" {kind}")
			return nil
		})

	v1.Group(".dev.book").
		PreMiddleware(func(message *Message, dep any) error {
			message.Bytes = []byte(fmt.Sprintf("* %s", message.Bytes))
			return nil
		}).
		Handler(".discount", func(message *Message, dep any) error {
			recorder = append(recorder, string(message.Bytes))
			return nil
		})

	expectedResponse := []string{
		"! v1.devops.book.6263334908 {kind}",
		"! v1.dev.book.1449373321 {kind}",
		"* ! v1.dev.book.discount",
		"v2 default",
	}

	messages := []*Message{
		{Subject: "v1.devops.book.6263334908", RouteParam: map[string]any{}},
		{Subject: "v1.dev.book.1449373321", RouteParam: map[string]any{}},
		{Subject: "v1.dev.book.discount", RouteParam: map[string]any{}},
		{Subject: "v2", RouteParam: map[string]any{}},
	}

	for i, message := range messages {
		message.Bytes = []byte(message.Subject)
		err := mux.HandleMessage(message, nil)
		if err != nil {
			t.Errorf("%v: unexpected error: got %v", message.Subject, err)
			break
		}
		if recorder[i] != expectedResponse[i] {
			t.Errorf("%v: unexpected output: got %s, want %s", message.Subject, recorder[i], expectedResponse[i])
			break
		}
	}

}

func TestMux_RouteParam_when_wildcard_subject(t *testing.T) {
	mux := NewMux("/")

	actual := []string{}
	mux.
		Handler("order/kind/game", func(message *Message, dep any) error {
			actual = append(actual, message.Subject)
			return nil
		}).
		Handler("order/{user_id}", func(message *Message, dep any) error {
			actual = append(actual, message.RouteParam.Str("user_id"))
			return nil
		}).
		Handler("/get/test/abc/", func(message *Message, dep any) error {
			actual = append(actual, message.Subject)
			return nil
		}).
		Handler("/get/{param}/abc/", func(message *Message, dep any) error {
			actual = append(actual, message.RouteParam.Str("param"))
			return nil
		}).
		Handler("{kind}/book/{book_id}", func(message *Message, dep any) error {
			actual = append(actual, message.RouteParam.Str("kind")+" "+message.RouteParam.Str("book_id"))
			return nil
		}).
		Handler("dev/book/{book_id}", func(message *Message, dep any) error {
			actual = append(actual, "dev book "+message.RouteParam.Str("book_id"))
			return nil
		}).
		Handler("dev/ebook/{book_id}", func(message *Message, dep any) error {
			actual = append(actual, "dev ebook "+message.RouteParam.Str("book_id"))
			return nil
		})

	expectedSubjects := []string{
		"/get/test/abc/",
		"/get/{param}/abc/",
		"dev/book/{book_id}",
		"dev/ebook/{book_id}",
		"order/kind/game",
		"order/{user_id}",
		"{kind}/book/{book_id}",
	}

	i := 0
	mux.Endpoints(func(subject, handler string) {
		if subject != expectedSubjects[i] {
			t.Errorf("unexpected output: got %s, want %s", subject, expectedSubjects[i])
		}
		i++
	})

	expectedResponse := []string{
		"order/kind/game",
		"qaz1017",
		"/get/test/abc/",
		"xyz",
		"tt",
		"devops 6263334908",
		"dev book 1449373321",
		"dev ebook 1492052205",
	}

	messages := []*Message{
		{Subject: "order/kind/game", RouteParam: map[string]any{}},
		{Subject: "order/qaz1017", RouteParam: map[string]any{}},
		{Subject: "/get/test/abc/", RouteParam: map[string]any{}},
		{Subject: "/get/xyz/abc/", RouteParam: map[string]any{}},
		{Subject: "/get/tt/abc/", RouteParam: map[string]any{}},
		{Subject: "devops/book/6263334908", RouteParam: map[string]any{}},
		{Subject: "dev/book/1449373321", RouteParam: map[string]any{}},
		{Subject: "dev/ebook/1492052205", RouteParam: map[string]any{}},
	}

	for i, message := range messages {
		err := mux.HandleMessage(message, nil)
		if err != nil {
			t.Errorf("%v: unexpected error: got %v", message.Subject, err)
			break
		}
		if actual[i] != expectedResponse[i] {
			t.Errorf("%v: unexpected output: got %s, want %s", message.Subject, actual[i], expectedResponse[i])
			break
		}
	}
}

func TestMessageMux_Recover(t *testing.T) {

	// SetDefaultLogger(SilentLogger())

	mux := NewMux("/").
		Middleware(
			UseRecover(),
		).
		DefaultHandler(func(_ *Message, dep any) error {
			panic("dependency is nil")
			return nil
		})

	message := &Message{
		Subject: "create_order",
	}

	mux.HandleMessage(message, nil)
}
