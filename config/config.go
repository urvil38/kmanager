package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func GetConfigPath(clusterName string) (string, error) {
	confPath, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	kConfPath := filepath.Join(confPath, "kmanager", clusterName)

	err = os.MkdirAll(kConfPath, 0777)
	if err != nil {
		return "", err
	}

	return kConfPath, nil
}

func ClusterConfigPath(clusterName string) (string, error) {
	confPath, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	kConfPath := filepath.Join(confPath, "kmanager", clusterName)

	if _, err := os.Stat(kConfPath); errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("No cluster found with name \"%s\"", clusterName)
	}

	return kConfPath, nil
}

func KmanagerConfigPath() (string, error) {
	confPath, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	kConfPath := filepath.Join(confPath, "kmanager")

	return kConfPath, nil
}
