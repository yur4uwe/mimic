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

function ConvertTo-TomlString {
  param(
    [string]$s
  )
  return ($s -replace '\\', '\\\\' -replace '"', '\"' -replace "`n", "\n")
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
    $HomeDir = $env:USERPROFILE
    if (-not $HomeDir) { return $null }
    $base = Join-Path $HomeDir "AppData\Roaming"
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

function Install-WinFSP {
  if (Test-WinFspInstalled) { return $true }

  $apiUrl = "https://api.github.com/repos/billziss-gh/winfsp/releases/latest"

  try {
    $rel = Invoke-RestMethod -Uri $apiUrl -UseBasicParsing -ErrorAction Stop
  }
  catch {
    Write-Error "Failed to query WinFsp releases: $_"
    return $false
  }

  $asset = $rel.assets | Where-Object { $_.name -match "(?i)winfsp.*\.msi$" } | Select-Object -First 1
  if (-not $asset) {
    Write-Error "No suitable WinFsp MSI found in latest release. See: $($rel.html_url)"
    return $false
  }

  $msiUrl = $asset.browser_download_url
  $tmp = Join-Path $env:TEMP ("winfsp-install-$([guid]::NewGuid()).msi")

  Write-Host "Downloading WinFsp MSI..."
  try {
    Invoke-WebRequest -Uri $msiUrl -OutFile $tmp -UseBasicParsing -ErrorAction Stop
  }
  catch {
    Write-Error "Download failed: $_"
    return $false
  }

  try {
    $hash = Get-FileHash -Algorithm SHA256 -Path $tmp
    Write-Host "MSI path: $tmp"
    Write-Host "SHA256: $($hash.Hash)"
  }
  catch {
    Write-Warning "Could not compute hash of ${tmp}: $_"
  }

  Write-Host "Installing WinFsp MSI silently (msiexec) ..."
  $proc = Start-Process -FilePath "msiexec.exe" -ArgumentList "/i", $tmp, "/qn", "/norestart" -Wait -PassThru
  $exit = $proc.ExitCode

  Remove-Item -Force $tmp -ErrorAction SilentlyContinue

  if ($exit -ne 0) {
    Write-Error "msiexec failed with exit code $exit"
    return $false
  }

  if (Test-WinFspInstalled) {
    Write-Host "WinFsp installed successfully."
    return $true
  }
  else {
    Write-Error "WinFsp install finished but runtime not detected. You may need to reboot."
    return $false
  }
}

if (-not (Test-Admin)) {
  Write-Host "Elevation required. Restarting elevated..."

  $scriptFile = $PSCommandPath
  if (-not $scriptFile) { $scriptFile = $MyInvocation.MyCommand.Path }

  $arg = @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", $scriptFile)
  Start-Process -FilePath "powershell" -ArgumentList $arg -Verb RunAs -Wait
  exit
}

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

$DEF_MPOINT = "X:"
$DEF_URL = "https://webdav.example.com"
$DEF_USERNAME = "user"
$DEF_PASSWORD = "pass"
$DEF_TTL = "5s"
$DEF_MAX_ENTRIES = 100
$DEF_VERBOSE = $false
$DEF_ERR = "stderr"
$DEF_STD = "discard"

$ask = Read-Host "Would you like to provide configuration values now? (Y/n)"
if ([string]::IsNullOrWhiteSpace($ask)) { $ask = "Y" }

if ($ask -match '^(?i)y(es)?$') {
  $mpoint = Read-Host "Mount point [$DEF_MPOINT]"
  if (-not $mpoint) { $mpoint = $DEF_MPOINT }

  $url = Read-Host "Server URL [$DEF_URL]"
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
  $mpoint = $DEF_MPOINT
  $url = $DEF_URL
  $username = $DEF_USERNAME
  $password = $DEF_PASSWORD
  $ttl = $DEF_TTL
  $max_entries = $DEF_MAX_ENTRIES
  $verbose = $DEF_VERBOSE
  $err = $DEF_ERR
  $std = $DEF_STD
}

$cfgContent = @"
# Mimic configuration (generated)
mpoint = "$(ConvertTo-TomlString $mpoint)"
url = "$(ConvertTo-TomlString $url)"
username = "$(ConvertTo-TomlString $username)"
password = "$(ConvertTo-TomlString $password)"
ttl = "$ttl"
max-entries = $max_entries
verbose = $($verbose.ToString().ToLower())
err = "$(ConvertTo-TomlString $err)"
std = "$(ConvertTo-TomlString $std)"
"@

if ($userCfg) {
  $cfgContent | Out-File -FilePath $userCfg -Encoding UTF8 -Force
  Write-Host "Wrote config to $userCfg"
}

$sysCfg = Join-Path $installDir "config.toml"
$cfgContent | Out-File -FilePath $sysCfg -Encoding UTF8 -Force
Write-Host "Wrote config to $sysCfg"

if (-not (Test-WinFspInstalled)) {
  Write-Host "WinFsp not found - attempting automatic install..."
  if (-not (Install-WinFSP)) {
    Write-Host "Automatic WinFsp install failed. Please install manually: https://github.com/winfsp/winfsp/releases"
    Read-Host -Prompt "Press Enter to exit"
    exit 2
  }
}

Add-ToSystemPath -Dir $installDir

Write-Host ""
Write-Host "Installation complete."
Write-Host "Open a NEW terminal and run:"
Write-Host "  mimic --help"
Write-Host "Default mount point: X:"
Write-Host "Install dir: $installDir"
Read-Host -Prompt "Press Enter to exit"
