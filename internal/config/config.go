package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	DBSSLMode          string
	ValkeyAddr         string
	ValkeyPassword     string
	GitHubClientID     string
	GitHubClientSecret string
	GitHubRedirectURL  string
	GitLabClientID     string
	GitLabClientSecret string
	GitLabRedirectURL  string
	CSRFAuthKey        string
	RewardGitHubRepo   string
	RewardGitLabRepo   string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment variables")
	}

	return &Config{
		Port:               getEnv("PORT", "8080"),
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "5432"),
		DBUser:             getEnv("DB_USER", "shreelance"),
		DBPassword:         getEnv("DB_PASSWORD", "shreelance_password"),
		DBName:             getEnv("DB_NAME", "shreelance"),
		DBSSLMode:          getEnv("DB_SSLMODE", "disable"),
		ValkeyAddr:         getEnv("VALKEY_ADDR", "localhost:6379"),
		ValkeyPassword:     getEnv("VALKEY_PASSWORD", ""),
		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		GitHubRedirectURL:  getEnv("GITHUB_REDIRECT_URL", "http://localhost:8080/auth/github/callback"),
		GitLabClientID:     getEnv("GITLAB_CLIENT_ID", ""),
		GitLabClientSecret: getEnv("GITLAB_CLIENT_SECRET", ""),
		GitLabRedirectURL:  getEnv("GITLAB_REDIRECT_URL", "http://localhost:8080/auth/gitlab/callback"),
		CSRFAuthKey:        getEnv("CSRF_AUTH_KEY", "32-byte-long-csrf-auth-key-default-32"),
		RewardGitHubRepo:   getEnv("REWARD_GITHUB_REPO", "succeio/shreelance"),
		RewardGitLabRepo:   getEnv("REWARD_GITLAB_REPO", "blackteka/hikkasay"),
	}
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
