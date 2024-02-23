package Artifex

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/user"
	"path/filepath"
)

func LoadJsonFileByLocal[T any](path, defaultEnv, defaultName string) (T, error) {
	decode := func(data []byte, v *T) error {
		return json.Unmarshal(data, v)
	}
	return LoadFileByLocal[T](decode, path, defaultEnv, defaultName)
}

func LoadFileByLocal[T any](decode Unmarshal[T], path, defaultEnv, defaultFileName string) (T, error) {
	var empty T
	obj, err := loadFileByLocal(decode, path, defaultEnv, defaultFileName)
	if err != nil {
		return empty, ErrorJoin3rdParty(ErrUniversal, err)
	}
	return obj, nil
}

func loadFileByLocal[T any](decode Unmarshal[T], path, defaultEnv, defaultFileName string) (T, error) {
	const (
		byNormal int = iota + 1
		byEnvironmentVariable
		byHomeDir
		stop
	)

	getPath := func(search int) (string, error) {
		switch search {
		case byNormal:
			return path, nil

		case byEnvironmentVariable:
			byEnvPath, _ := os.LookupEnv(defaultEnv)
			return byEnvPath, nil

		case byHomeDir:
			osUser, err := user.Current()
			if err != nil {
				return "", err
			}
			return filepath.Join(osUser.HomeDir, defaultFileName), nil

		default:
			return "", nil
		}
	}

	var targetPath string
	var file *os.File
	var empty T
	var err error

	for searchKind := byNormal; searchKind < stop; searchKind++ {
		targetPath, err = getPath(searchKind)
		if err != nil {
			continue
		}

		if targetPath == "" {
			err = errors.New("path is empty")
			continue
		}

		DefaultLogger().Info("open config from %v", targetPath)
		file, err = os.Open(targetPath)
		if err == nil {
			break
		}
	}

	if err != nil {
		return empty, err
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return empty, err
	}

	body := new(T)
	err = decode(data, body)
	if err != nil {
		return empty, err
	}

	return *body, err
}
