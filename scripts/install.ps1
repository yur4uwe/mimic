function Test-Admin {
  $id = [Security.Principal.WindowsIdentity]::GetCurrent()
  (New-Object Security.Principal.WindowsPrincipal($id)).IsInRole(
    [Security.Principal.WindowsBuiltInRole]::Administrator
  )
}

function Test-WinFspInstalled {
  Write-Host "Checking for existing WinFsp installation..."
  if (Get-Service -Name WinFsp -ErrorAction SilentlyContinue) { return $true }

  $sys32 = Join-Path $env:SystemRoot "System32"
  $pf86 = ${env:ProgramFiles(x86)}
  $pf = $env:ProgramFiles

  if (Test-Path (Join-Path $sys32 "winfsp-x64.dll")) { return $true }
  if (Test-Path (Join-Path $sys32 "winfsp-x86.dll")) { return $true }

  if ($pf86 -and (Test-Path (Join-Path $pf86 "WinFsp\bin\winfsp-x64.dll"))) { return $true }
  if ($pf86 -and (Test-Path (Join-Path $pf86 "WinFsp\bin\winfsp-x86.dll"))) { return $true }

  if ($pf -and (Test-Path (Join-Path $pf "WinFsp\bin\winfsp-x64.dll"))) { return $true }
  if ($pf -and (Test-Path (Join-Path $pf "WinFsp\bin\winfsp-x86.dll"))) { return $true }

  return $false
}

function Escape-Toml($s) {
  return ($s -replace '\\','\\\\' -replace '"','\"' -replace "`n","\n")
}

function Get-ArchSuffix {
  if ([Environment]::Is64BitOperatingSystem) { return "x64" } else { return "x86" }
}

function Get-UserConfigPath {
  param(
    [string]$AppName = "mimic",
    [string]$FileName = "config.toml"
  )

  $base = $env:APPDATA
  if (-not $base) {
    $home = $env:USERPROFILE
    if (-not $home) { return $null }
    $base = Join-Path $home "AppData\Roaming"
  }

  $dir = Join-Path $base $AppName
  try {
    if (-not (Test-Path $dir)) {
      New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }
  }
  catch {
    Write-Warning "Could not create user config dir ${dir}: $($_.Exception.Message)"
    return $null
  }

  return Join-Path $dir $FileName
}

function Add-ToSystemPath {
  param([Parameter(Mandatory)][string]$Dir)

  $reg = "HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Environment"
  $current = (Get-ItemProperty $reg -Name Path).Path

  $parts = $current -split ';' | Where-Object { $_ -ne "" }
  if ($parts | Where-Object { $_.TrimEnd('\') -ieq $Dir.TrimEnd('\') }) {
    Write-Host "PATH already contains $Dir"
    return
  }

  $newPath = ($parts + $Dir) -join ';'
  Set-ItemProperty -Path $reg -Name Path -Value $newPath

  Write-Host "Added $Dir to system PATH (new shells only)."
}

if (-not (Test-Admin)) {
  Write-Host "Elevation required. Restarting elevated..."

  $scriptFile = $PSCommandPath
  if (-not $scriptFile) { $scriptFile = $MyInvocation.MyCommand.Path }

  Start-Process powershell `
    -ArgumentList "-NoProfile -ExecutionPolicy Bypass -File `"$scriptFile`"" `
    -Verb RunAs -Wait

  exit
}

$arch = Get-ArchSuffix
Write-Host "Detected OS architecture: $arch"

$bundleRoot = $PSScriptRoot
Write-Host "Bundle root: $bundleRoot"

$installDir = Join-Path $env:ProgramFiles "Mimic"

if (-not (Test-Path $installDir)) {
  New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

$files = @(
  "mimic.exe",
  "README.md",
  "LICENCE",
  "example-config.toml",
  "INSTALL",
  "install.ps1"
)

foreach ($f in $files) {
  $src = Join-Path $bundleRoot $f
  if (Test-Path $src) {
    Copy-Item -Path $src -Destination $installDir -Force
    Write-Host "Copied $f"
  }
}

$userCfg = Get-UserConfigPath
if (-not $userCfg) {
  Write-Warning "Per-user config path unavailable"
}

$DEF_MPOINT      = "X:"
$DEF_URL         = "https://webdav.example.com"
$DEF_USERNAME    = "user"
$DEF_PASSWORD    = "pass"
$DEF_TTL         = "5s"
$DEF_MAX_ENTRIES = 100
$DEF_VERBOSE     = $false
$DEF_ERR         = "stderr"
$DEF_STD         = "discard"

$ask = Read-Host "Would you like to provide configuration values now? (Y/n)"
if ([string]::IsNullOrWhiteSpace($ask)) { $ask = "Y" }

if ($ask -match '^(?i)y(es)?$') {
  $mpoint   = Read-Host "Mount point [$DEF_MPOINT]"
  if (-not $mpoint) { $mpoint = $DEF_MPOINT }

  $url      = Read-Host "Server URL [$DEF_URL]"
  if (-not $url) { $url = $DEF_URL }

  $username = Read-Host "Username [$DEF_USERNAME]"
  if (-not $username) { $username = $DEF_USERNAME }

  $password = Read-Host "Password [$DEF_PASSWORD]"
  if (-not $password) { $password = $DEF_PASSWORD }

  $ttl = Read-Host "Cache TTL [$DEF_TTL]"
  if (-not $ttl) { $ttl = $DEF_TTL }

  $max_entries = Read-Host "Cache max-entries [$DEF_MAX_ENTRIES]"
  if (-not $max_entries) { $max_entries = $DEF_MAX_ENTRIES }
  $max_entries = [int]$max_entries

  $verbose_in = Read-Host "Verbose logging (true/false) [$DEF_VERBOSE]"
  if (-not $verbose_in) { $verbose = $DEF_VERBOSE }
  else { $verbose = [System.Convert]::ToBoolean($verbose_in) }

  $err = Read-Host "stderr target [$DEF_ERR]"
  if (-not $err) { $err = $DEF_ERR }

  $std = Read-Host "stdout target [$DEF_STD]"
  if (-not $std) { $std = $DEF_STD }
}
else {
  Write-Host "Using default configuration values."
  $mpoint      = $DEF_MPOINT
  $url         = $DEF_URL
  $username    = $DEF_USERNAME
  $password    = $DEF_PASSWORD
  $ttl         = $DEF_TTL
  $max_entries = $DEF_MAX_ENTRIES
  $verbose     = $DEF_VERBOSE
  $err         = $DEF_ERR
  $std         = $DEF_STD
}

$cfgContent = @"
# Mimic configuration (generated)
mpoint = "$(Escape-Toml $mpoint)"
url = "$(Escape-Toml $url)"
username = "$(Escape-Toml $username)"
password = "$(Escape-Toml $password)"
ttl = "$ttl"
max-entries = $max_entries
verbose = $($verbose.ToString().ToLower())
err = "$(Escape-Toml $err)"
std = "$(Escape-Toml $std)"
"@

if ($userCfg) {
  $cfgContent | Out-File -FilePath $userCfg -Encoding UTF8 -Force
  Write-Host "Wrote config to $userCfg"
}

$sysCfg = Join-Path $installDir "config.toml"
$cfgContent | Out-File -FilePath $sysCfg -Encoding UTF8 -Force
Write-Host "Wrote config to $sysCfg"

if (-not (Test-WinFspInstalled)) {
  Write-Host "WinFsp not found. Please install it manually:"
  Write-Host "https://github.com/billziss-gh/winfsp/releases"
  exit 2
}

Add-ToSystemPath -Dir $installDir

Write-Host ""
Write-Host "Installation complete."
Write-Host "Open a NEW terminal and run:"
Write-Host "  mimic --help"
Write-Host "Default mount point: X:"
Write-Host "Install dir: $installDir"
