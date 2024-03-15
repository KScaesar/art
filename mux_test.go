package Artifex

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
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
		return message.channel + "/", nil
	}

	mux := NewMessageMux[string, *redisMessage](getSubject, DefaultLogger()).
		RegisterHandler("hello", func(dto *redisMessage) error {
			fmt.Fprintf(recorder, "topic=%v, payload=%v", dto.channel, string(dto.body))
			return nil
		}).
		RegisterHandler("foo", func(dto *redisMessage) error {
			fmt.Fprintf(recorder, "topic=%v, payload=%v", dto.channel, string(dto.body))
			return nil
		})

	// expect
	expected := `topic=hello, payload={"data":"world"}`

	// action
	dto := &redisMessage{
		ctx:     context.Background(),
		body:    []byte(`{"data":"world"}`),
		channel: "hello",
	}
	err := mux.HandleMessage(dto)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// assert
	if expected != recorder.String() {
		t.Errorf("handle(): %v, but want: %v", recorder.String(), expected)
	}
}

func TestLinkMiddlewares(t *testing.T) {
	buf := new(bytes.Buffer)
	buf.WriteString("\n")

	decorator1 := func(next MessageHandleFunc[string]) MessageHandleFunc[string] {
		return func(dto string) error {
			fmt.Fprintf(buf, "%s_decorator1\n", dto)

			err := next(dto)
			if err != nil {
				return err
			}

			fmt.Fprintf(buf, "%s_decoratorA\n", dto)
			return nil
		}
	}

	decorator2 := func(next MessageHandleFunc[string]) MessageHandleFunc[string] {
		return func(dto string) error {
			fmt.Fprintf(buf, "%s_decorator2\n", dto)

			err := next(dto)
			if err != nil {
				return err
			}

			fmt.Fprintf(buf, "%s_decoratorB\n", dto)
			return nil
		}
	}

	decorator3 := func(next MessageHandleFunc[string]) MessageHandleFunc[string] {
		return func(dto string) error {
			fmt.Fprintf(buf, "%s_decorator3\n", dto)

			err := next(dto)
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
		middlewares []MessageDecorator[string]
		expected    string
	}{
		{
			name:        "Multiple decorator check sequence",
			msg:         "hello_world",
			middlewares: []MessageDecorator[string]{decorator1, decorator2, decorator3},
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
			baseFunc := func(msg string) error {
				fmt.Fprintf(buf, "%s_base\n", msg)
				return nil
			}

			handler := LinkMiddlewares(baseFunc, tt.middlewares...)
			err := handler(tt.msg)
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
	record := func(message *testcaseTransformMessage) error {
		recorder = append(recorder, message.body)
		return nil
	}

	newSubject := func(msg *testcaseTransformMessage) (string, error) {
		return strconv.Itoa(msg.level0TypeId) + "/", nil
	}
	mux := NewMessageMux[int, *testcaseTransformMessage](newSubject, NewLogger(true, LogLevelInfo))

	mux.RegisterHandler(2, record)

	mux.Group(3).Transform(testcaseTransformLevel1).
		RegisterHandler(1, record).
		RegisterHandler(2, record).
		Group(5).Transform(testcaseTransformLevel2).
		AddPreMiddleware(
			func(message *testcaseTransformMessage) error {
				message.body = "^" + message.body + "^"
				return nil
			}).
		RegisterHandler(1, record).
		RegisterHandler(2, record)

	mux.Group(4).Transform(testcaseTransformLevel1).
		AddPreMiddleware(
			func(message *testcaseTransformMessage) error {
				message.body = "_" + message.body + "_"
				return nil
			}).
		RegisterHandler(1, record).
		RegisterHandler(2, record).
		RegisterHandler(4, record)

	expectedSubjects := []string{
		"2/",
		"3/1/", "3/2/",
		"3/5/1/", "3/5/2/",
		"4/1/", "4/2/", "4/4/",
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
		err := mux.HandleMessage(message)
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

func testcaseTransformLevel1(old *testcaseTransformMessage) (fresh *testcaseTransformMessage, err error) {
	return &testcaseTransformMessage{
		level0TypeId: old.level1TypeId,
		level1TypeId: 0,
		body:         old.body,
	}, nil
}

func testcaseTransformLevel2(old *testcaseTransformMessage) (fresh *testcaseTransformMessage, err error) {
	level2TypeId, err := strconv.Atoi(old.body)
	if err != nil {
		return nil, err
	}
	originalLevel1TypeId := old.level0TypeId
	return &testcaseTransformMessage{
		level0TypeId: level2TypeId,
		level1TypeId: 0,
		body:         fmt.Sprintf("TransformLevel2 %v/%v/", originalLevel1TypeId, level2TypeId),
	}, nil
}

type testcaseTransformMessage struct {
	level0TypeId int
	level1TypeId int
	body         string
}

func Test_parseGroupSubject(t *testing.T) {
	groupDelimiter := "/"
	tests := []struct {
		name    string
		subject string
		want    [][]string
	}{
		{
			subject: "1/2/3/",
			want: [][]string{
				{"1/2/3/", ""},
				{"1/2/", "3/"},
				{"1/", "2/3/"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseGroupSubject(tt.subject, groupDelimiter); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseGroupSubject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageMux_Group(t *testing.T) {
	type Subject string
	type testcaseGroupMessage struct {
		subject Subject
		body    string
	}

	recorder := []string{}
	record := func(message *testcaseGroupMessage) error {
		recorder = append(recorder, message.body)
		return nil
	}

	newSubject := func(msg *testcaseGroupMessage) (string, error) { return string(msg.subject), nil }
	mux := NewMessageMux[Subject, *testcaseGroupMessage](newSubject, NewLogger(true, LogLevelInfo))
	mux.SetGroupDelimiter("/").
		AddPreMiddleware(func(message *testcaseGroupMessage) error {
			message.body = "*" + message.body + "*"
			return nil
		})

	mux.RegisterHandler("topic1", record)

	mux.Group("/topic2").
		RegisterHandler("/orders", record).
		RegisterHandler("/users", record)

	mux.Group("topic3/").
		RegisterHandler("created/orders/", record).
		RegisterHandler("login/users/", record).
		RegisterHandler("upgraded.users/", record)

	topic4 := mux.Group("/topic4/").
		RegisterHandler("game1", record).
		RegisterHandler("/game2/kindA/", record)
	topic4_game3 := topic4.Group("game3").
		AddPreMiddleware(func(message *testcaseGroupMessage) error {
			message.body = "&" + message.body + "&"
			return nil
		}).
		RegisterHandler("kindX", record).
		RegisterHandler("kindY", record)
	topic4_game3.Group("v2").
		RegisterHandler("kindX", record)
	topic4_game3.Group("v3").
		RegisterHandler("kindY", record)

	mux.RegisterHandler("topic5/game1/", record)
	mux.RegisterHandler("topic5/game2/kindA", record)
	mux.RegisterHandler("//topic5/game3/kindX/", record)
	mux.RegisterHandler("topic5/game3/kindY/", record)
	mux.RegisterHandler("topic5/game3/v2/kindX//", record)
	mux.RegisterHandler("topic5/game3/v3/kindY", record)

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
		err := mux.HandleMessage(message)
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
