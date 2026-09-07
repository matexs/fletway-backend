# scripts/dev.ps1 — arranque de desarrollo en Windows/PowerShell.
# Uso: .\scripts\dev.ps1

$ErrorActionPreference = "Stop"
Set-Location (Split-Path $PSScriptRoot -Parent)

if (-not (Test-Path ".env")) {
    Write-Host "No existe .env — copiando desde .env.example" -ForegroundColor Yellow
    Copy-Item ".env.example" ".env"
    Write-Host "Completá .env antes de continuar." -ForegroundColor Yellow
    exit 1
}

# Carga .env al entorno del proceso
Get-Content ".env" | Where-Object { $_ -match '^\s*[^#].*=' } | ForEach-Object {
    $name, $value = $_ -split '=', 2
    [Environment]::SetEnvironmentVariable($name.Trim(), $value.Trim(), "Process")
}

go run ./cmd/api
