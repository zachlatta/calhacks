package config

import (
	"log"
	"os"

	"github.com/calhacks/calhacks/osutil"
	"github.com/kylelemons/go-gypsy/yaml"
)

var (
	config   *yaml.File
	dbConfig *yaml.File
)

func init() {
	baseCfgPath, err := osutil.ResolveFilePathInEnv("GOPATH",
		"/src/github.com/calhacks/calhacks/")
	if err != nil {
		panic(err)
	}

	cfgPath := baseCfgPath + "config/config.yml"
	dbCfgPath := baseCfgPath + "db/dbconf.yml"

	config, err = yaml.ReadFile(cfgPath)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal("Error loading config", err)
	}

	dbConfig, err = yaml.ReadFile(dbCfgPath)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal("Error loading config", err)
	}
}

func Get(param string) string {
	val := os.Getenv(param)
	if val == "" && config != nil {
		val, _ = config.Get(param)
		val = os.ExpandEnv(val)
	}
	return val
}

// TODO: Actually read and dynamically load from dbconf.yml.
func DatabaseURL() string {
	val := os.Getenv("DATABASE_URL")
	if val == "" {
		return os.ExpandEnv(
			"postgres://docker:docker@$DB_1_PORT_5432_TCP_ADDR/docker")
	}
	return ""
}
