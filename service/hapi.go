package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	hapiSetupTokenPrefix = "hapi_setup_"
)

var (
	hapiBaseTokenCacheMu sync.Mutex
	hapiBaseTokenCache   struct {
		value     string
		expiresAt time.Time
	}
)

type HapiSetupConfig struct {
	ApiURL                 string `json:"api_url"`
	Namespace              string `json:"namespace"`
	CliApiToken            string `json:"cli_api_token"`
	SetupToken             string `json:"setup_token,omitempty"`
	SetupCommand           string `json:"setup_command,omitempty"`
	SetupShellCommand      string `json:"setup_shell_command,omitempty"`
	SetupPowerShellCommand string `json:"setup_powershell_command,omitempty"`
	InstallScriptURL       string `json:"install_script_url,omitempty"`
	InstallPowerShellURL   string `json:"install_powershell_url,omitempty"`
	GuideURL               string `json:"guide_url,omitempty"`
}

func BuildHapiSetupConfig(ctx context.Context, token *model.Token, tokenHubBaseURL string) (*HapiSetupConfig, error) {
	if err := validateHapiToken(token); err != nil {
		return nil, err
	}

	publicURL := strings.TrimRight(common.GetEnvOrDefaultString("HAPI_PUBLIC_URL", ""), "/")
	if publicURL == "" {
		return nil, errors.New("HAPI_PUBLIC_URL is not configured")
	}

	baseToken, err := GetHapiBaseToken(ctx)
	if err != nil {
		return nil, err
	}

	namespace, err := BuildHapiNamespace(token)
	if err != nil {
		return nil, err
	}

	setupToken, err := BuildHapiSetupToken(token)
	if err != nil {
		return nil, err
	}

	tokenHubBaseURL = strings.TrimRight(tokenHubBaseURL, "/")
	config := &HapiSetupConfig{
		ApiURL:      publicURL,
		Namespace:   namespace,
		CliApiToken: baseToken + ":" + namespace,
		SetupToken:  setupToken,
	}
	if tokenHubBaseURL != "" {
		config.InstallScriptURL = tokenHubBaseURL + "/api/hapi/install.sh"
		config.InstallPowerShellURL = tokenHubBaseURL + "/api/hapi/install.ps1"
		config.SetupShellCommand = fmt.Sprintf("HAPI_SETUP_TOKEN='%s' bash -c \"$(curl -fsSL %s/api/hapi/install.sh)\"", setupToken, tokenHubBaseURL)
		config.SetupPowerShellCommand = fmt.Sprintf("$env:HAPI_SETUP_TOKEN='%s'; iex ((iwr -UseBasicParsing %s/api/hapi/install.ps1).Content)", setupToken, tokenHubBaseURL)
		config.SetupCommand = config.SetupShellCommand
		config.GuideURL = tokenHubBaseURL + "/guide#hapi"
	}
	return config, nil
}

func BuildHapiSetupConfigBySetupToken(ctx context.Context, setupToken string) (*HapiSetupConfig, error) {
	token, err := ResolveHapiSetupToken(setupToken)
	if err != nil {
		return nil, err
	}
	return BuildHapiSetupConfig(ctx, token, "")
}

func BuildHapiNamespace(token *model.Token) (string, error) {
	if token == nil {
		return "", errors.New("token not found")
	}
	salt := common.GetEnvOrDefaultString("HAPI_NAMESPACE_SALT", "")
	if salt == "" {
		return "", errors.New("HAPI_NAMESPACE_SALT is not configured")
	}
	sum := md5.Sum([]byte(fmt.Sprintf("%d:%s:%s", token.Id, token.Key, salt)))
	return "hapi_ns_" + hex.EncodeToString(sum[:])[:16], nil
}

func BuildHapiSetupToken(token *model.Token) (string, error) {
	if token == nil {
		return "", errors.New("token not found")
	}
	salt := getHapiSetupSalt()
	if salt == "" {
		return "", errors.New("HAPI_NAMESPACE_SALT is not configured")
	}
	message := fmt.Sprintf("%d:%d:%s", token.Id, token.UserId, token.Key)
	signature := common.HmacSha256(message, salt)
	return fmt.Sprintf("%s%d_%s", hapiSetupTokenPrefix, token.Id, signature[:32]), nil
}

func ResolveHapiSetupToken(setupToken string) (*model.Token, error) {
	setupToken = strings.TrimSpace(setupToken)
	if !strings.HasPrefix(setupToken, hapiSetupTokenPrefix) {
		return nil, errors.New("invalid setup token")
	}

	raw := strings.TrimPrefix(setupToken, hapiSetupTokenPrefix)
	parts := strings.Split(raw, "_")
	if len(parts) != 2 {
		return nil, errors.New("invalid setup token")
	}

	tokenID, err := strconv.Atoi(parts[0])
	if err != nil || tokenID <= 0 {
		return nil, errors.New("invalid setup token")
	}

	token, err := model.GetTokenById(tokenID)
	if err != nil {
		return nil, errors.New("invalid setup token")
	}

	expected, err := BuildHapiSetupToken(token)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(expected, setupToken) {
		return nil, errors.New("invalid setup token")
	}
	if err := validateHapiToken(token); err != nil {
		return nil, err
	}
	return token, nil
}

func GetHapiBaseToken(ctx context.Context) (string, error) {
	if direct := strings.TrimSpace(common.GetEnvOrDefaultString("HAPI_BASE_TOKEN", "")); direct != "" {
		return direct, nil
	}

	command := strings.TrimSpace(common.GetEnvOrDefaultString("HAPI_BASE_TOKEN_COMMAND", ""))
	if command == "" {
		return "", errors.New("HAPI_BASE_TOKEN or HAPI_BASE_TOKEN_COMMAND is not configured")
	}

	ttl := time.Duration(common.GetEnvOrDefault("HAPI_BASE_TOKEN_CACHE_SECONDS", 60)) * time.Second
	now := time.Now()
	hapiBaseTokenCacheMu.Lock()
	if hapiBaseTokenCache.value != "" && now.Before(hapiBaseTokenCache.expiresAt) {
		value := hapiBaseTokenCache.value
		hapiBaseTokenCacheMu.Unlock()
		return value, nil
	}
	hapiBaseTokenCacheMu.Unlock()

	timeout := time.Duration(common.GetEnvOrDefault("HAPI_BASE_TOKEN_COMMAND_TIMEOUT_SECONDS", 5)) * time.Second
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	out, err := exec.CommandContext(cmdCtx, "/bin/sh", "-c", command).Output()
	if err != nil {
		return "", errors.New("failed to read HAPI base token")
	}
	value := strings.TrimSpace(string(out))
	if strings.Contains(value, "\n") {
		value = strings.TrimSpace(strings.SplitN(value, "\n", 2)[0])
	}
	if value == "" {
		return "", errors.New("HAPI base token is empty")
	}

	hapiBaseTokenCacheMu.Lock()
	hapiBaseTokenCache.value = value
	hapiBaseTokenCache.expiresAt = now.Add(ttl)
	hapiBaseTokenCacheMu.Unlock()
	return value, nil
}

func getHapiSetupSalt() string {
	if salt := common.GetEnvOrDefaultString("HAPI_SETUP_SALT", ""); salt != "" {
		return salt
	}
	return common.GetEnvOrDefaultString("HAPI_NAMESPACE_SALT", "")
}

func validateHapiToken(token *model.Token) error {
	if token == nil {
		return errors.New("token not found")
	}
	if token.Status != common.TokenStatusEnabled {
		return errors.New("token is not enabled")
	}
	if token.ExpiredTime != -1 && token.ExpiredTime < common.GetTimestamp() {
		return errors.New("token is expired")
	}
	if !token.UnlimitedQuota && token.RemainQuota <= 0 {
		return errors.New("token quota is exhausted")
	}
	return nil
}
