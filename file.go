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
	decode := func(data []byte, v any) error {
		return json.Unmarshal(data, v)
	}
	return LoadFileByLocal[T](decode, path, defaultEnv, defaultName)
}

func LoadFileByLocal[T any](decode Unmarshal, path, defaultEnv, defaultFileName string) (T, error) {
	obj, err := loadFileByLocal[T](decode, path, defaultEnv, defaultFileName)
	if err != nil {
		return obj, ErrorJoin3rdParty(ErrUniversal, err)
	}
	return obj, nil
}

func loadFileByLocal[T any](decode Unmarshal, path, defaultEnv, defaultFileName string) (obj T, err error) {
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
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return
	}

	body := new(T)
	err = decode(data, body)
	if err != nil {
		return
	}

	return *body, err
}
