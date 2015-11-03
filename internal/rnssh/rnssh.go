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
	rnsshDir := os.Getenv(ENV_HOME) + string(os.PathSeparator) + RNSSH_DIR_NAME
	return rnsshDir
}

func CreateRnsshDir() error {
	rnsshDir := GetRnsshDir()

	if _, err := os.Stat(rnsshDir); os.IsNotExist(err) {
		err = os.Mkdir(rnsshDir, 0700)
		if err != nil {
			if !os.IsExist(err) {
				return err
			}
		}
	}

	return nil
}
