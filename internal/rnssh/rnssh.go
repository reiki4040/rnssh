package rnssh

import (
	"os"
)

const (
	ENV_HOME = "HOME"

	RNSSH_DIR_NAME = ".rnssh"
)

type Choosable interface {
	Choice() string
	GetSshTarget() string
}

func GetRnsshDir() string {
	rnzooDir := os.Getenv(ENV_HOME) + string(os.PathSeparator) + RNSSH_DIR_NAME
	return rnzooDir
}

func CreateRnsshDir() error {
	rnzooDir := GetRnsshDir()

	if _, err := os.Stat(rnzooDir); os.IsNotExist(err) {
		err = os.Mkdir(rnzooDir, 0700)
		if err != nil {
			if !os.IsExist(err) {
				return err
			}
		}
	}

	return nil
}
