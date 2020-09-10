package config

import (
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
