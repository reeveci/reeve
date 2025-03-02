package server

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func setupConfigFile(v *viper.Viper) {
	v.SetConfigFile(configFile)
	if ext := filepath.Ext(configFile); ext == "" || ext == "." {
		v.SetConfigType("toml")
	}
}

func loadConfigFile(v *viper.Viper) error {
	setupConfigFile(v)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
