package Artifex

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/exp/slices"
)

func TestMessageMux_handle(t *testing.T) {
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
	err := mux.handle(dto)
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
	newSubject := func(msg *testcaseTransformMessage) (string, error) {
		return strconv.Itoa(msg.level0TypeId) + "/", nil
	}
	recorder := &strings.Builder{}
	recorder.WriteString("\n")

	mux := NewMessageMux[int, *testcaseTransformMessage](newSubject, NewLogger(true, LogLevelInfo))

	mux.RegisterHandler(2, writeSubject(recorder))

	mux.Group(3).Transform(testcaseTransformLevel1).
		RegisterHandler(1, writeSubject(recorder)).
		RegisterHandler(2, writeSubject(recorder)).
		Group(5).Transform(testcaseTransformLevel2).
		AddPreMiddleware(
			func(message *testcaseTransformMessage) error {
				message.body = "^" + message.body + "^"
				return nil
			}).
		RegisterHandler(1, writeSubject(recorder)).
		RegisterHandler(2, writeSubject(recorder))

	mux.Group(4).Transform(testcaseTransformLevel1).
		AddPreMiddleware(
			func(message *testcaseTransformMessage) error {
				message.body = "_" + message.body + "_"
				return nil
			}).
		RegisterHandler(1, writeSubject(recorder)).
		RegisterHandler(2, writeSubject(recorder)).
		RegisterHandler(4, writeSubject(recorder))

	expectedSubjects := []string{
		"2/",
		"3/1/", "3/2/",
		"3/5/1/", "3/5/2/",
		"4/1/", "4/2/", "4/4/",
	}

	gotSubjects := mux.Subjects()
	if !slices.Equal(gotSubjects, expectedSubjects) {
		t.Errorf("unexpected output: got %s, want %s", gotSubjects, expectedSubjects)
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

	expectedRecord := `
msg 2/
msg 3/1/
^TransformLevel2 5/1/^
_msg 4/2/_
`

	gotRecord := recorder.String()
	if gotRecord != expectedRecord {
		t.Errorf("unexpected output: got %s, want %s", gotRecord, expectedRecord)
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

func writeSubject(w *strings.Builder) func(message *testcaseTransformMessage) error {
	return func(message *testcaseTransformMessage) error {
		w.WriteString(message.body)
		w.WriteString("\n")
		return nil
	}
}
