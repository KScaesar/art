package Artifex

import (
	"bytes"
	"context"
	"fmt"
	"testing"
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

	mux := NewMessageMux(getSubject, DefaultLogger()).
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

	decorator1 := func(next MessageHandler[string]) MessageHandler[string] {
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

	decorator2 := func(next MessageHandler[string]) MessageHandler[string] {
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

	decorator3 := func(next MessageHandler[string]) MessageHandler[string] {
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
