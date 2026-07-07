package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetTokenHapiSetup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	userID := c.GetInt("id")
	if err != nil {
		common.ApiError(c, err)
		return
	}

	token, err := model.GetTokenByIds(id, userID)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	config, err := service.BuildHapiSetupConfig(c.Request.Context(), token, requestBaseURL(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, config)
}

func GetHapiSetupConfig(c *gin.Context) {
	setupToken := strings.TrimSpace(c.Query("setup_token"))
	if setupToken == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			setupToken = strings.TrimSpace(authHeader[len("Bearer "):])
		}
	}
	if setupToken == "" {
		common.ApiErrorMsg(c, "setup token is required")
		return
	}

	config, err := service.BuildHapiSetupConfigBySetupToken(c.Request.Context(), setupToken)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, config)
}

func GetHapiInstallScript(c *gin.Context) {
	c.Data(http.StatusOK, "text/x-shellscript; charset=utf-8", []byte(buildHapiInstallScript(requestBaseURL(c))))
}

func GetHapiInstallPowerShell(c *gin.Context) {
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(buildHapiInstallPowerShell(requestBaseURL(c))))
}

func requestBaseURL(c *gin.Context) string {
	if publicURL := strings.TrimRight(common.GetEnvOrDefaultString("HAPI_TOKEN_HUB_PUBLIC_URL", ""), "/"); publicURL != "" {
		return publicURL
	}

	proto := c.GetHeader("X-Forwarded-Proto")
	if proto == "" {
		if c.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return strings.TrimRight(proto+"://"+host, "/")
}

func buildHapiInstallScript(tokenHubBaseURL string) string {
	return `#!/usr/bin/env bash
set -euo pipefail

TOKEN_HUB_URL="${TOKEN_HUB_URL:-` + tokenHubBaseURL + `}"
HAPI_HOME="${HAPI_HOME:-$HOME/.hapi}"

if [ -z "${HAPI_SETUP_TOKEN:-}" ]; then
  echo "HAPI_SETUP_TOKEN is required." >&2
  exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required." >&2
  exit 1
fi

if ! command -v node >/dev/null 2>&1; then
  echo "Node.js is required before installing HAPI." >&2
  exit 1
fi

if ! command -v hapi >/dev/null 2>&1; then
  if ! command -v npm >/dev/null 2>&1; then
    echo "npm is required to install @twsxtd/hapi." >&2
    exit 1
  fi
  npm install -g @twsxtd/hapi
fi

CONFIG_JSON="$(curl -fsSL -H "Authorization: Bearer ${HAPI_SETUP_TOKEN}" "${TOKEN_HUB_URL%/}/api/hapi/setup-config")"
export CONFIG_JSON
export HAPI_HOME

node <<'NODE'
const fs = require('fs')
const path = require('path')

const payload = JSON.parse(process.env.CONFIG_JSON)
if (!payload.success || !payload.data) {
  throw new Error(payload.message || 'Failed to fetch HAPI setup config')
}

const data = payload.data
const hapiHome = process.env.HAPI_HOME
fs.mkdirSync(hapiHome, { recursive: true })

const settingsPath = path.join(hapiHome, 'settings.json')
let settings = {}
if (fs.existsSync(settingsPath)) {
  try {
    settings = JSON.parse(fs.readFileSync(settingsPath, 'utf8'))
  } catch {
    settings = {}
  }
}

settings.apiUrl = data.api_url
settings.cliApiToken = data.cli_api_token

fs.writeFileSync(settingsPath, JSON.stringify(settings, null, 2) + '\n', 'utf8')
console.log('HAPI settings written to ' + settingsPath)
console.log('Namespace: ' + data.namespace)
NODE

hapi --version || true
echo "Run HAPI with: HAPI_HOME=\"${HAPI_HOME}\" hapi"
`
}

func buildHapiInstallPowerShell(tokenHubBaseURL string) string {
	return `$ErrorActionPreference = "Stop"

$TokenHubUrl = if ($env:TOKEN_HUB_URL) { $env:TOKEN_HUB_URL.TrimEnd("/") } else { "` + tokenHubBaseURL + `" }
$HapiHome = if ($env:HAPI_HOME) { $env:HAPI_HOME } else { Join-Path $HOME ".hapi" }

if (-not $env:HAPI_SETUP_TOKEN) {
  throw "HAPI_SETUP_TOKEN is required."
}

if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
  throw "Node.js is required before installing HAPI."
}

if (-not (Get-Command hapi -ErrorAction SilentlyContinue)) {
  if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
    throw "npm is required to install @twsxtd/hapi."
  }
  npm install -g @twsxtd/hapi
}

$Headers = @{ Authorization = "Bearer $env:HAPI_SETUP_TOKEN" }
$Payload = Invoke-RestMethod -Headers $Headers -Uri "$TokenHubUrl/api/hapi/setup-config"
if (-not $Payload.success -or -not $Payload.data) {
  throw $(if ($Payload.message) { $Payload.message } else { "Failed to fetch HAPI setup config" })
}

New-Item -ItemType Directory -Force -Path $HapiHome | Out-Null
$SettingsPath = Join-Path $HapiHome "settings.json"
$Settings = [ordered]@{}
if (Test-Path $SettingsPath) {
  try {
    $Existing = Get-Content $SettingsPath -Raw | ConvertFrom-Json
    foreach ($Property in $Existing.PSObject.Properties) {
      $Settings[$Property.Name] = $Property.Value
    }
  } catch {
    $Settings = [ordered]@{}
  }
}

$Settings["apiUrl"] = $Payload.data.api_url
$Settings["cliApiToken"] = $Payload.data.cli_api_token
$Json = $Settings | ConvertTo-Json -Depth 10
$Utf8NoBom = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllText($SettingsPath, $Json + [Environment]::NewLine, $Utf8NoBom)

Write-Host "HAPI settings written to $SettingsPath"
Write-Host "Namespace: $($Payload.data.namespace)"
hapi --version
Write-Host "Run HAPI with HAPI_HOME set to $HapiHome, then run hapi"
`
}
