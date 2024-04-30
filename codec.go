package Artifex

type Unmarshal func(bData []byte, v any) error

type Marshal func(v any) ([]byte, error)
