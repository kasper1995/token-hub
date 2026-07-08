package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const defaultTeamDeskGitLabProviderSlug = "gitlab"

type teamDeskTokenSummary struct {
	Id                 int      `json:"id"`
	Name               string   `json:"name"`
	Status             string   `json:"status"`
	Group              string   `json:"group"`
	MaskedKey          string   `json:"masked_key"`
	ModelLimits        []string `json:"model_limits"`
	ModelLimitsEnabled bool     `json:"model_limits_enabled"`
	HapiAvailable      bool     `json:"hapi_available"`
	ExpiredTime        int64    `json:"expired_time"`
	RemainQuota        int      `json:"remain_quota"`
	UnlimitedQuota     bool     `json:"unlimited_quota"`
}

type teamDeskSetupConfigRequest struct {
	TokenId       int      `json:"token_id"`
	IncludeSecret bool     `json:"include_secret"`
	Modules       []string `json:"modules"`
}

// GetTeamDeskGitLabMe 向 Team Desk 暴露当前 GitLab 账号关联的 token-hub 只读信息。
func GetTeamDeskGitLabMe(c *gin.Context) {
	identity, err := resolveTeamDeskGitLabIdentity(c)
	if err != nil {
		if errors.Is(err, errTeamDeskBindingNotFound) {
			common.ApiSuccess(c, gin.H{
				"connected":        false,
				"provider":         identity.Provider.Slug,
				"provider_user_id": identity.ProviderUserId,
				"gitlab_user":      identity.GitLabUser,
				"tokens":           []teamDeskTokenSummary{},
			})
			return
		}
		writeTeamDeskIdentityError(c, err)
		return
	}

	tokens, err := model.GetAllUserTokens(identity.User.Id, 0, 100)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	summaries := make([]teamDeskTokenSummary, 0, len(tokens))
	for _, token := range tokens {
		summaries = append(summaries, buildTeamDeskTokenSummary(token))
	}

	common.ApiSuccess(c, gin.H{
		"connected":        true,
		"provider":         identity.Provider.Slug,
		"provider_user_id": identity.ProviderUserId,
		"user": gin.H{
			"id":           identity.User.Id,
			"username":     identity.User.Username,
			"display_name": identity.User.DisplayName,
			"email":        identity.User.Email,
		},
		"tokens": summaries,
	})
}

// GetTeamDeskGitLabSetupConfig 向 Team Desk 返回指定 token 的本机初始化配置。
func GetTeamDeskGitLabSetupConfig(c *gin.Context) {
	identity, err := resolveTeamDeskGitLabIdentity(c)
	if err != nil {
		if errors.Is(err, errTeamDeskBindingNotFound) {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "token-hub account is not connected."})
			return
		}
		writeTeamDeskIdentityError(c, err)
		return
	}

	var request teamDeskSetupConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	if request.TokenId <= 0 {
		common.ApiErrorMsg(c, "token_id is required.")
		return
	}

	token, err := model.GetTokenByIds(request.TokenId, identity.User.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := validateTeamDeskSetupToken(token); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	tokenHubBaseURL := requestBaseURL(c)
	data := gin.H{
		"token": buildTeamDeskTokenSummary(token),
		"endpoints": gin.H{
			"token_hub":      tokenHubBaseURL,
			"anthropic_base": tokenHubBaseURL,
			"openai_base":    strings.TrimRight(tokenHubBaseURL, "/") + "/v1",
			"gitlab":         strings.TrimSuffix(gitlabAPIUserEndpoint(identity.Provider), "/api/v4/user"),
		},
	}

	if request.IncludeSecret {
		data["key"] = token.GetFullKey()
		if wantsTeamDeskModule(request.Modules, "hapi") {
			config, err := service.BuildHapiSetupConfig(c.Request.Context(), token, tokenHubBaseURL)
			if err != nil {
				common.ApiError(c, err)
				return
			}
			data["hapi"] = config
		}
	}

	common.ApiSuccess(c, data)
}

type teamDeskGitLabIdentity struct {
	Provider       *model.CustomOAuthProvider
	ProviderUserId string
	GitLabUser     map[string]any
	User           model.User
}

var errTeamDeskBindingNotFound = errors.New("token-hub account is not connected")

func resolveTeamDeskGitLabIdentity(c *gin.Context) (*teamDeskGitLabIdentity, error) {
	accessToken, err := bearerToken(c.GetHeader("Authorization"))
	if err != nil {
		return nil, err
	}

	provider, err := model.GetCustomOAuthProviderBySlug(teamDeskGitLabProviderSlug())
	if err != nil {
		return nil, errors.New("GitLab OAuth provider is not configured.")
	}
	if !provider.Enabled {
		return nil, errors.New("GitLab OAuth provider is disabled.")
	}

	providerUserId, userInfo, err := fetchProviderUserId(c.Request.Context(), provider, accessToken)
	if err != nil {
		return nil, err
	}

	binding, err := model.GetOAuthBindingByProviderUserId(provider.Id, providerUserId)
	if err != nil {
		return &teamDeskGitLabIdentity{
			Provider:       provider,
			ProviderUserId: providerUserId,
			GitLabUser:     userInfo,
		}, errTeamDeskBindingNotFound
	}

	user := model.User{Id: binding.UserId}
	if err := user.FillUserById(); err != nil {
		return nil, err
	}
	if user.Status != common.UserStatusEnabled {
		return nil, errors.New("token-hub user is disabled.")
	}

	return &teamDeskGitLabIdentity{
		Provider:       provider,
		ProviderUserId: providerUserId,
		GitLabUser:     userInfo,
		User:           user,
	}, nil
}

func writeTeamDeskIdentityError(c *gin.Context, err error) {
	if strings.Contains(err.Error(), "GitLab userinfo failed") || strings.Contains(err.Error(), "Missing Bearer token") {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": err.Error()})
		return
	}
	common.ApiErrorMsg(c, err.Error())
}

func validateTeamDeskSetupToken(token *model.Token) error {
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

func wantsTeamDeskModule(modules []string, name string) bool {
	for _, module := range modules {
		if strings.EqualFold(strings.TrimSpace(module), name) {
			return true
		}
	}
	return false
}

func bearerToken(authHeader string) (string, error) {
	parts := strings.Fields(authHeader)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", errors.New("Missing Bearer token.")
	}
	return parts[1], nil
}

func teamDeskGitLabProviderSlug() string {
	if slug := strings.TrimSpace(os.Getenv("TEAM_DESK_GITLAB_PROVIDER_SLUG")); slug != "" {
		return slug
	}
	return defaultTeamDeskGitLabProviderSlug
}

func fetchProviderUserId(ctx context.Context, provider *model.CustomOAuthProvider, accessToken string) (string, map[string]any, error) {
	providerUserId, userInfo, err := fetchProviderUserIdFromEndpoint(ctx, provider, provider.UserInfoEndpoint, accessToken)
	if err == nil {
		return providerUserId, userInfo, nil
	}

	gitlabAPIUserEndpoint := gitlabAPIUserEndpoint(provider)
	if gitlabAPIUserEndpoint == "" || gitlabAPIUserEndpoint == provider.UserInfoEndpoint {
		return "", nil, err
	}

	return fetchProviderUserIdFromEndpoint(ctx, &model.CustomOAuthProvider{
		UserIdField:      "id",
		UsernameField:    "username",
		DisplayNameField: "name",
		EmailField:       "email",
	}, gitlabAPIUserEndpoint, accessToken)
}

func fetchProviderUserIdFromEndpoint(ctx context.Context, provider *model.CustomOAuthProvider, endpoint string, accessToken string) (string, map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", nil, fmt.Errorf("GitLab userinfo failed: %d", res.StatusCode)
	}

	providerUserId := strings.TrimSpace(gjson.GetBytes(body, provider.UserIdField).String())
	if providerUserId == "" {
		return "", nil, errors.New("GitLab userinfo missing user id.")
	}

	userInfo := map[string]any{}
	_ = common.Unmarshal(body, &userInfo)
	return providerUserId, publicGitLabUserInfo(provider, userInfo), nil
}

func gitlabAPIUserEndpoint(provider *model.CustomOAuthProvider) string {
	endpoint := strings.TrimSpace(provider.UserInfoEndpoint)
	if endpoint == "" {
		return ""
	}
	if strings.Contains(endpoint, "/oauth/userinfo") {
		return strings.Replace(endpoint, "/oauth/userinfo", "/api/v4/user", 1)
	}
	if strings.Contains(endpoint, "/api/v4/user") {
		return endpoint
	}
	authEndpoint := strings.TrimSpace(provider.AuthorizationEndpoint)
	if strings.Contains(authEndpoint, "/oauth/authorize") {
		return strings.Replace(authEndpoint, "/oauth/authorize", "/api/v4/user", 1)
	}
	return ""
}

func publicGitLabUserInfo(provider *model.CustomOAuthProvider, raw map[string]any) map[string]any {
	return map[string]any{
		"id":       gjson.Get(toJSON(raw), provider.UserIdField).String(),
		"username": gjson.Get(toJSON(raw), provider.UsernameField).String(),
		"name":     gjson.Get(toJSON(raw), provider.DisplayNameField).String(),
		"email":    gjson.Get(toJSON(raw), provider.EmailField).String(),
	}
}

func toJSON(value any) string {
	data, err := common.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func buildTeamDeskTokenSummary(token *model.Token) teamDeskTokenSummary {
	return teamDeskTokenSummary{
		Id:                 token.Id,
		Name:               token.Name,
		Status:             tokenStatusName(token.Status),
		Group:              token.Group,
		MaskedKey:          token.GetMaskedKey(),
		ModelLimits:        token.GetModelLimits(),
		ModelLimitsEnabled: token.ModelLimitsEnabled,
		HapiAvailable:      token.Status == common.TokenStatusEnabled && (token.ExpiredTime == -1 || token.ExpiredTime >= common.GetTimestamp()) && (token.UnlimitedQuota || token.RemainQuota > 0),
		ExpiredTime:        token.ExpiredTime,
		RemainQuota:        token.RemainQuota,
		UnlimitedQuota:     token.UnlimitedQuota,
	}
}

func tokenStatusName(status int) string {
	switch status {
	case common.TokenStatusEnabled:
		return "enabled"
	case common.TokenStatusDisabled:
		return "disabled"
	case common.TokenStatusExpired:
		return "expired"
	case common.TokenStatusExhausted:
		return "exhausted"
	default:
		return "unknown"
	}
}
