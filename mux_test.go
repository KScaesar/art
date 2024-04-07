package Artifex

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"testing"
)

func TestMessageMux_HandleMessage(t *testing.T) {
	// arrange
	type redisMessage struct {
		ctx     context.Context
		body    []byte
		channel string
	}
	recorder := &bytes.Buffer{}

	getSubject := func(message *redisMessage) string {
		return message.channel
	}

	mux := NewMux[redisMessage]("", getSubject).
		Handler("hello", func(dto *redisMessage, route *RouteParam) error {
			fmt.Fprintf(recorder, "topic=%v, payload=%v", dto.channel, string(dto.body))
			return nil
		}).
		Handler("foo", func(dto *redisMessage, route *RouteParam) error {
			fmt.Fprintf(recorder, "topic=%v, payload=%v", dto.channel, string(dto.body))
			return nil
		})

	// expect
	expected := `topic=hello, payload={"data":"world"}`

	// paramHandler
	dto := &redisMessage{
		ctx:     context.Background(),
		body:    []byte(`{"data":"world"}`),
		channel: "hello",
	}
	err := mux.HandleMessage(dto, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// assert
	if expected != recorder.String() {
		t.Errorf("handleMessage(): %v, but want: %v", recorder.String(), expected)
	}
}

func TestLinkMiddlewares_when_wildcard(t *testing.T) {
	type testcaseMessage struct {
		subject string
		body    string
	}

	recorder := []string{}
	newSubject := func(msg *testcaseMessage) string { return msg.subject }
	mux := NewMux[testcaseMessage](".", newSubject)

	v1 := mux.Group("v1")

	v1.
		PreMiddleware(func(message *testcaseMessage, route *RouteParam) error {
			message.body = "! " + message.body
			return nil
		}).
		Handler(".{kind}.book.{book_id}", func(message *testcaseMessage, _ *RouteParam) error {
			recorder = append(recorder, message.body+" {kind}")
			return nil
		})

	v1.Group(".dev.book").
		PreMiddleware(func(message *testcaseMessage, route *RouteParam) error {
			message.body = "* " + message.body
			return nil
		}).
		Handler(".discount", func(message *testcaseMessage, _ *RouteParam) error {
			recorder = append(recorder, message.body)
			return nil
		})

	expectedResponse := []string{
		"! v1.devops.book.6263334908 {kind}",
		"! v1.dev.book.1449373321 {kind}",
		"* ! v1.dev.book.discount",
	}

	messages := []*testcaseMessage{
		{subject: "v1.devops.book.6263334908"},
		{subject: "v1.dev.book.1449373321"},
		{subject: "v1.dev.book.discount"},
	}

	for i, message := range messages {
		message.body = message.subject
		err := mux.HandleMessage(message, nil)
		if err != nil {
			t.Errorf("%v: unexpected error: got %v", message.subject, err)
			break
		}
		if recorder[i] != expectedResponse[i] {
			t.Errorf("%v: unexpected output: got %s, want %s", message.subject, recorder[i], expectedResponse[i])
			break
		}
	}

}

func TestLinkMiddlewares(t *testing.T) {
	buf := new(bytes.Buffer)
	buf.WriteString("\n")

	decorator1 := func(next HandleFunc[string]) HandleFunc[string] {
		return func(dto *string, route *RouteParam) error {
			fmt.Fprintf(buf, "%s_decorator1\n", *dto)

			err := next(dto, route)
			if err != nil {
				return err
			}

			fmt.Fprintf(buf, "%s_decoratorA\n", *dto)
			return nil
		}
	}

	decorator2 := func(next HandleFunc[string]) HandleFunc[string] {
		return func(dto *string, route *RouteParam) error {
			fmt.Fprintf(buf, "%s_decorator2\n", *dto)

			err := next(dto, route)
			if err != nil {
				return err
			}

			fmt.Fprintf(buf, "%s_decoratorB\n", *dto)
			return nil
		}
	}

	decorator3 := func(next HandleFunc[string]) HandleFunc[string] {
		return func(dto *string, route *RouteParam) error {
			fmt.Fprintf(buf, "%s_decorator3\n", *dto)

			err := next(dto, route)
			if err != nil {
				return err
			}

			fmt.Fprintf(buf, "%s_decoratorC\n", *dto)
			return nil
		}
	}

	tests := []struct {
		name        string
		msg         string
		middlewares []Middleware[string]
		expected    string
	}{
		{
			name:        "Multiple decorator check sequence",
			msg:         "hello_world",
			middlewares: []Middleware[string]{decorator1, decorator2, decorator3},
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
			baseFunc := func(msg *string, route *RouteParam) error {
				fmt.Fprintf(buf, "%s_base\n", *msg)
				return nil
			}

			handler := LinkMiddlewares(baseFunc, tt.middlewares...)
			err := handler(&tt.msg, nil)
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
	newSubject := func(msg *testcaseTransformMessage) string {
		return "/" + strconv.Itoa(msg.level0TypeId)
	}

	mux := NewMux[testcaseTransformMessage]("/", newSubject)

	mux.SetDefaultHandler(func(_ *testcaseTransformMessage, _ *RouteParam) error {
		return nil
	})

	message := &testcaseTransformMessage{
		level0TypeId: 0,
		level1TypeId: 0,
		body:         "",
	}

	err := mux.HandleMessage(message, nil)
	if err != nil {
		t.Errorf("%#v :unexpected error: got %v", message, err)
	}
}

func TestMessageMux_Transform(t *testing.T) {
	recorder := []string{}
	record := func(message *testcaseTransformMessage, route *RouteParam) error {
		recorder = append(recorder, message.subject)
		return nil
	}

	newSubject := func(msg *testcaseTransformMessage) string { return msg.subject }
	mux := NewMux[testcaseTransformMessage]("/", newSubject).
		PreMiddleware(
			func(message *testcaseTransformMessage, route *RouteParam) error {
				message.subject = "!" + message.subject + "!"
				return nil
			})

	mux.HandlerByNumber(2, record)

	mux.GroupByNumber(3).Transform(testcaseTransformLevel1).
		HandlerByNumber(1, record).
		HandlerByNumber(2, record).
		GroupByNumber(5).Transform(testcaseTransformLevel2).
		PreMiddleware(
			func(message *testcaseTransformMessage, route *RouteParam) error {
				message.subject = "^" + message.subject + "^"
				return nil
			}).
		HandlerByNumber(1, record).
		HandlerByNumber(2, record)

	mux.GroupByNumber(4).Transform(testcaseTransformLevel1).
		PreMiddleware(
			func(message *testcaseTransformMessage, route *RouteParam) error {
				message.subject = "_" + message.subject + "_"
				return nil
			}).
		HandlerByNumber(1, record).
		HandlerByNumber(2, record).
		HandlerByNumber(4, record)

	expectedSubjects := []string{
		"/2",
		"/3/1", "/3/2",
		"/3/5/1", "/3/5/2",
		"/4/1", "/4/2", "/4/4",
	}

	endpoints := mux.Endpoints()
	for i, endpoint := range endpoints {
		if endpoint[0] != expectedSubjects[i] {
			t.Errorf("unexpected output: got %s, want %s", endpoint[0], expectedSubjects[i])
		}
	}

	expectedRecords := []string{
		"!/2!",
		"!/3/1!",
		"^!/3/5/1!^",
		"_!/4/2!_",
	}

	messages := []*testcaseTransformMessage{
		{"/2", 2, -1, "msg /2"},
		{"/3", 3, 1, "msg /3/1"},
		{"/3", 3, 5, "1"},
		{"/4", 4, 2, "msg /4/2"},
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

func testcaseTransformLevel1(msg *testcaseTransformMessage, route *RouteParam) (err error) {
	old := msg
	msg.level0TypeId = old.level1TypeId
	msg.level1TypeId = 0
	msg.subject += "/" + strconv.Itoa(msg.level0TypeId)
	return nil
}

func testcaseTransformLevel2(msg *testcaseTransformMessage, route *RouteParam) (err error) {
	level2TypeId, err := strconv.Atoi(msg.body)
	if err != nil {
		return err
	}

	msg.level0TypeId = level2TypeId
	msg.level1TypeId = 0

	msg.subject += "/" + strconv.Itoa(msg.level0TypeId)
	return nil
}

type testcaseTransformMessage struct {
	subject      string
	level0TypeId int
	level1TypeId int
	body         string
}

func TestMessageMux_Group(t *testing.T) {
	type Subject string
	type testcaseGroupMessage struct {
		subject Subject
		body    string
	}

	recorder := []string{}
	record := func(message *testcaseGroupMessage, route *RouteParam) error {
		recorder = append(recorder, message.body)
		return nil
	}

	newSubject := func(msg *testcaseGroupMessage) string { return string(msg.subject) }
	mux := NewMux[testcaseGroupMessage]("/", newSubject)

	mux.PreMiddleware(func(message *testcaseGroupMessage, route *RouteParam) error {
		message.body = "*" + message.body + "*"
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
		PreMiddleware(func(message *testcaseGroupMessage, route *RouteParam) error {
			message.body = "&" + message.body + "&"
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

	endpoints := mux.Endpoints()
	for i, endpoint := range endpoints {
		if endpoint[0] != expectedSubjects[i] {
			t.Errorf("unexpected output: got %s, want %s", endpoint[0], expectedSubjects[i])
		}
	}

	subjects := []Subject{
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
		message := &testcaseGroupMessage{
			subject: subject,
			body:    string(subject),
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

func TestMux_SetDefaultHandler(t *testing.T) {
	type Subject string
	type testcaseDefaultHandler struct {
		subject Subject
		body    string
	}

	isCalled := false
	newSubject := func(msg *testcaseDefaultHandler) string { return string(msg.subject) }
	mux := NewMux[testcaseDefaultHandler]("/", newSubject).
		SetDefaultHandler(func(message *testcaseDefaultHandler, route *RouteParam) error {
			isCalled = true
			return nil
		})

	msg := &testcaseDefaultHandler{}
	err := mux.HandleMessage(msg, nil)
	if err != nil || isCalled != true {
		t.Errorf("unexpected error: got %v", err)
	}
}

func TestMux_SetDefaultHandler_when_wildcard(t *testing.T) {
	type testcaseMessage struct {
		subject string
		body    string
	}

	recorder := []string{}
	newSubject := func(msg *testcaseMessage) string { return msg.subject }
	mux := NewMux[testcaseMessage](".", newSubject).
		Middleware(MW_Recover[testcaseMessage]()).
		SetDefaultHandler(func(message *testcaseMessage, _ *RouteParam) error {
			recorder = append(recorder, message.body+" default")
			return nil
		})

	v1 := mux.Group("v1")

	v1.
		PreMiddleware(func(message *testcaseMessage, route *RouteParam) error {
			message.body = "! " + message.body
			return nil
		}).
		Handler(".{kind}.book.{book_id}", func(message *testcaseMessage, _ *RouteParam) error {
			recorder = append(recorder, message.body+" {kind}")
			return nil
		})

	v1.Group(".dev.book").
		PreMiddleware(func(message *testcaseMessage, route *RouteParam) error {
			message.body = "* " + message.body
			return nil
		}).
		Handler(".discount", func(message *testcaseMessage, _ *RouteParam) error {
			recorder = append(recorder, message.body)
			return nil
		})

	expectedResponse := []string{
		"! v1.devops.book.6263334908 {kind}",
		"! v1.dev.book.1449373321 {kind}",
		"* ! v1.dev.book.discount",
		"v2 default",
	}

	messages := []*testcaseMessage{
		{subject: "v1.devops.book.6263334908"},
		{subject: "v1.dev.book.1449373321"},
		{subject: "v1.dev.book.discount"},
		{subject: "v2"},
	}

	for i, message := range messages {
		message.body = message.subject
		err := mux.HandleMessage(message, nil)
		if err != nil {
			t.Errorf("%v: unexpected error: got %v", message.subject, err)
			break
		}
		if recorder[i] != expectedResponse[i] {
			t.Errorf("%v: unexpected output: got %s, want %s", message.subject, recorder[i], expectedResponse[i])
			break
		}
	}

}

func TestMux_RouteParam_when_wildcard_subject(t *testing.T) {
	type Subject = string
	type testcaseRouteParam struct {
		subject Subject
		body    string
	}

	newSubject := func(msg *testcaseRouteParam) string { return string(msg.subject) }
	mux := NewMux[testcaseRouteParam]("/", newSubject)

	actual := []string{}
	mux.
		Handler("order/kind/game", func(message *testcaseRouteParam, route *RouteParam) error {
			actual = append(actual, message.subject)
			return nil
		}).
		Handler("order/{user_id}", func(message *testcaseRouteParam, route *RouteParam) error {
			actual = append(actual, route.Str("user_id"))
			return nil
		}).
		Handler("/get/test/abc/", func(message *testcaseRouteParam, route *RouteParam) error {
			actual = append(actual, message.subject)
			return nil
		}).
		Handler("/get/{param}/abc/", func(message *testcaseRouteParam, route *RouteParam) error {
			actual = append(actual, route.Str("param"))
			return nil
		}).
		Handler("{kind}/book/{book_id}", func(message *testcaseRouteParam, route *RouteParam) error {
			actual = append(actual, route.Str("kind")+" "+route.Str("book_id"))
			return nil
		}).
		Handler("dev/book/{book_id}", func(message *testcaseRouteParam, route *RouteParam) error {
			actual = append(actual, "dev book "+route.Str("book_id"))
			return nil
		}).
		Handler("dev/ebook/{book_id}", func(message *testcaseRouteParam, route *RouteParam) error {
			actual = append(actual, "dev ebook "+route.Str("book_id"))
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

	endpoints := mux.Endpoints()
	for i, endpoint := range endpoints {
		if endpoint[0] != expectedSubjects[i] {
			t.Errorf("unexpected output: got %s, want %s", endpoint[0], expectedSubjects[i])
		}
	}

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

	messages := []*testcaseRouteParam{
		{subject: "order/kind/game"},
		{subject: "order/qaz1017"},
		{subject: "/get/test/abc/"},
		{subject: "/get/xyz/abc/"},
		{subject: "/get/tt/abc/"},
		{subject: "devops/book/6263334908"},
		{subject: "dev/book/1449373321"},
		{subject: "dev/ebook/1492052205"},
	}

	for i, message := range messages {
		err := mux.HandleMessage(message, nil)
		if err != nil {
			t.Errorf("%v: unexpected error: got %v", message.subject, err)
			break
		}
		if actual[i] != expectedResponse[i] {
			t.Errorf("%v: unexpected output: got %s, want %s", message.subject, actual[i], expectedResponse[i])
			break
		}
	}
}

func TestMessageMux_Recover(t *testing.T) {
	type testcaseRecover struct {
		subject string
		body    string
	}

	newSubject := func(msg *testcaseRecover) string {
		return msg.subject
	}

	SetDefaultLogger(SilentLogger())
	mux := NewMux[testcaseRecover]("/", newSubject).
		Middleware(MW_Recover[testcaseRecover]()).
		SetDefaultHandler(func(_ *testcaseRecover, _ *RouteParam) error {
			panic("testcase")
			return nil
		})

	message := &testcaseRecover{
		subject: "create_order",
		body:    "",
	}

	err := mux.HandleMessage(message, nil)
	if err != nil {
		t.Errorf("%#v :unexpected error: got %v", message, err)
	}
}
