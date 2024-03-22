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

	getSubject := func(message *redisMessage) (string, error) {
		return message.channel, nil
	}

	mux := NewMux[string, *redisMessage](getSubject).
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

	// routeHandler
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

func TestLinkMiddlewares(t *testing.T) {
	buf := new(bytes.Buffer)
	buf.WriteString("\n")

	decorator1 := func(next HandleFunc[string]) HandleFunc[string] {
		return func(dto string, route *RouteParam) error {
			fmt.Fprintf(buf, "%s_decorator1\n", dto)

			err := next(dto, route)
			if err != nil {
				return err
			}

			fmt.Fprintf(buf, "%s_decoratorA\n", dto)
			return nil
		}
	}

	decorator2 := func(next HandleFunc[string]) HandleFunc[string] {
		return func(dto string, route *RouteParam) error {
			fmt.Fprintf(buf, "%s_decorator2\n", dto)

			err := next(dto, route)
			if err != nil {
				return err
			}

			fmt.Fprintf(buf, "%s_decoratorB\n", dto)
			return nil
		}
	}

	decorator3 := func(next HandleFunc[string]) HandleFunc[string] {
		return func(dto string, route *RouteParam) error {
			fmt.Fprintf(buf, "%s_decorator3\n", dto)

			err := next(dto, route)
			if err != nil {
				return err
			}

			fmt.Fprintf(buf, "%s_decoratorC\n", dto)
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
			baseFunc := func(msg string, route *RouteParam) error {
				fmt.Fprintf(buf, "%s_base\n", msg)
				return nil
			}

			handler := LinkMiddlewares(baseFunc, tt.middlewares...)
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

func TestMessageMux_Transform(t *testing.T) {
	recorder := []string{}
	record := func(message *testcaseTransformMessage, route *RouteParam) error {
		recorder = append(recorder, message.body)
		return nil
	}

	newSubject := func(msg *testcaseTransformMessage) (string, error) {
		return "/" + strconv.Itoa(msg.level0TypeId), nil
	}
	mux := NewMux[int, *testcaseTransformMessage](newSubject).
		SetDelimiter('/', true)

	mux.Handler(2, record)

	mux.Group(3).Transform(testcaseTransformLevel1).
		Handler(1, record).
		Handler(2, record).
		Group(5).Transform(testcaseTransformLevel2).
		PreMiddleware(
			func(message *testcaseTransformMessage, route *RouteParam) error {
				message.body = "^" + message.body + "^"
				return nil
			}).
		Handler(1, record).
		Handler(2, record)

	mux.Group(4).Transform(testcaseTransformLevel1).
		PreMiddleware(
			func(message *testcaseTransformMessage, route *RouteParam) error {
				message.body = "_" + message.body + "_"
				return nil
			}).
		Handler(1, record).
		Handler(2, record).
		Handler(4, record)

	expectedSubjects := []string{
		"/2",
		"/3/1", "/3/2",
		"/3/5/1", "/3/5/2",
		"/4/1", "/4/2", "/4/4",
	}

	gotSubjects := mux.Subjects()
	for i, gotSubject := range gotSubjects {
		if gotSubject != expectedSubjects[i] {
			t.Errorf("unexpected output: got %s, want %s", gotSubject, expectedSubjects[i])
		}
	}

	messages := []*testcaseTransformMessage{
		{2, -1, "msg 2/"},
		{3, 1, "msg 3/1/"},
		{3, 5, "1"},
		{4, 2, "msg 4/2/"},
	}

	for _, message := range messages {
		err := mux.HandleMessage(message, nil)
		if err != nil {
			t.Errorf("unexpected error: got %v", err)
			break
		}
	}

	expectedRecords := []string{
		"msg 2/",
		"msg 3/1/",
		"^TransformLevel2 5/1/^",
		"_msg 4/2/_",
	}

	for i, gotRecord := range recorder {
		if gotRecord != expectedRecords[i] {
			t.Errorf("unexpected output: got %s, want %s", gotRecord, expectedRecords[i])
		}
	}
}

func testcaseTransformLevel1(msg *testcaseTransformMessage, route *RouteParam) (err error) {
	old := msg
	msg.level0TypeId = old.level1TypeId
	msg.level1TypeId = 0
	return nil
}

func testcaseTransformLevel2(msg *testcaseTransformMessage, route *RouteParam) (err error) {
	level2TypeId, err := strconv.Atoi(msg.body)
	if err != nil {
		return err
	}
	originalLevel1TypeId := msg.level0TypeId

	msg.level0TypeId = level2TypeId
	msg.level1TypeId = 0
	msg.body = fmt.Sprintf("TransformLevel2 %v/%v/", originalLevel1TypeId, level2TypeId)

	return nil
}

type testcaseTransformMessage struct {
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

	newSubject := func(msg *testcaseGroupMessage) (string, error) { return string(msg.subject), nil }
	mux := NewMux[Subject, *testcaseGroupMessage](newSubject).
		SetCleanSubject(true).
		SetDelimiter('/', false)

	mux.PreMiddleware(func(message *testcaseGroupMessage, route *RouteParam) error {
		message.body = "*" + message.body + "*"
		return nil
	})

	mux.Handler("topic1", record)

	mux.Group("/topic2").
		Handler("/orders", record).
		Handler("/users", record)

	mux.Group("topic3/").
		Handler("created/orders/", record).
		Handler("login/users/", record).
		Handler("upgraded.users/", record)

	topic4 := mux.Group("/topic4/").
		Handler("game1", record).
		Handler("/game2/kindA/", record)
	topic4_game3 := topic4.Group("game3").
		PreMiddleware(func(message *testcaseGroupMessage, route *RouteParam) error {
			message.body = "&" + message.body + "&"
			return nil
		}).
		Handler("kindX", record).
		Handler("kindY", record)
	topic4_game3.Group("v2").
		Handler("kindX", record)
	topic4_game3.Group("v3").
		Handler("kindY", record)

	mux.Handler("topic5/game1/", record)
	mux.Handler("topic5/game2/kindA", record)
	mux.Handler("//topic5/game3/kindX/", record)
	mux.Handler("topic5/game3/kindY/", record)
	mux.Handler("topic5/game3/v2/kindX//", record)
	mux.Handler("topic5/game3/v3/kindY", record)

	expectedSubjects := []string{
		"topic1/",
		"topic2/orders/", "topic2/users/",
		"topic3/created/orders/", "topic3/login/users/", "topic3/upgraded.users/",
		"topic4/game1/", "topic4/game2/kindA/", "topic4/game3/kindX/", "topic4/game3/kindY/", "topic4/game3/v2/kindX/", "topic4/game3/v3/kindY/",
		"topic5/game1/", "topic5/game2/kindA/", "topic5/game3/kindX/", "topic5/game3/kindY/", "topic5/game3/v2/kindX/", "topic5/game3/v3/kindY/",
	}

	gotSubjects := mux.Subjects()
	for i, gotSubject := range gotSubjects {
		if gotSubject != expectedSubjects[i] {
			t.Errorf("unexpected output: got %s, want %s", gotSubject, expectedSubjects[i])
		}
	}

	subjects := []Subject{
		"topic3/upgraded.users/",
		"topic4/game3/kindX/",
		"topic5/game3/kindY/",
		"topic4/game3/kindY/",
		"topic2/users/",
		"topic3/created/orders/",
		"topic5/game3/v2/kindX/",
		"topic5/game2/kindA/",
		"topic3/login/users/",
		"topic1/",
		"topic4/game3/v3/kindY/",
		"topic5/game3/kindX/",
		"topic5/game1/",
		"topic4/game1/",
		"topic4/game3/v2/kindX/",
		"topic2/orders/",
		"topic4/game2/kindA/",
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
		"*topic3/upgraded.users/*",
		"&*topic4/game3/kindX/*&",
		"*topic5/game3/kindY/*",
		"&*topic4/game3/kindY/*&",
		"*topic2/users/*",
		"*topic3/created/orders/*",
		"*topic5/game3/v2/kindX/*",
		"*topic5/game2/kindA/*",
		"*topic3/login/users/*",
		"*topic1/*",
		"&*topic4/game3/v3/kindY/*&",
		"*topic5/game3/kindX/*",
		"*topic5/game1/*",
		"*topic4/game1/*",
		"&*topic4/game3/v2/kindX/*&",
		"*topic2/orders/*",
		"*topic4/game2/kindA/*",
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

	newSubject := func(msg *testcaseDefaultHandler) (string, error) { return string(msg.subject), nil }
	mux := NewMux[Subject, *testcaseDefaultHandler](newSubject).
		SetDefaultHandler(func(message *testcaseDefaultHandler, route *RouteParam) error {
			return nil
		})

	msg := &testcaseDefaultHandler{}
	err := mux.HandleMessage(msg, nil)
	if err != nil {
		t.Errorf("unexpected error: got %v", err)
	}
}
