package Artifex

type Unmarshal[T any] func(data []byte, v *T) error

type Marshal func(v any) ([]byte, error)
