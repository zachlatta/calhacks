package config

import (
	"log"
	"os"

	"code.google.com/p/goauth2/oauth"

	"github.com/zachlatta/calhacks/osutil"
	"github.com/kylelemons/go-gypsy/yaml"
)

var (
	config   *yaml.File
	dbConfig *yaml.File

	githubOauthConfig *oauth.Config
)

func init() {
	baseCfgPath, err := osutil.ResolveFilePathInEnv("GOPATH",
		"/src/github.com/zachlatta/calhacks/")
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

	githubOauthConfig = &oauth.Config{
		ClientId:     GitHubClientID(),
		ClientSecret: GitHubClientSecret(),
		Scope:        "public",
		AuthURL:      "https://github.com/login/oauth/authorize",
		TokenURL:     "https://github.com/login/oauth/access_token",
		RedirectURL:  RedirectURL(),
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
	return val
}

func GitHubOauthConfig() *oauth.Config {
	return githubOauthConfig
}

func GitHubClientID() string {
	return Get("GITHUB_CLIENT_ID")
}

func GitHubClientSecret() string {
	return Get("GITHUB_CLIENT_SECRET")
}

func RedirectURL() string {
	return Get("REDIRECT_URL")
}

func JWTSecret() string {
	return Get("JWT_SECRET")
}
