function Is-Admin {
  $id = [Security.Principal.WindowsIdentity]::GetCurrent()
  (New-Object Security.Principal.WindowsPrincipal($id)).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Is-WinFspInstalled {
  if (Get-Service -Name WinFsp -ErrorAction SilentlyContinue) { return $true }
  $sys32 = Join-Path $env:SystemRoot "System32"
  $winfspDef = Join-Path $env:ProgramFilesx86 "WinFsp\lib"
  if (Test-Path (Join-Path $sys32 "winfsp-x64.dll")) { return $true }
  if (Test-Path (Join-Path $sys32 "winfsp-x86.dll")) { return $true }
  if (Test-Path (Join-Path $winfspDef "winfsp-x64.dll")) { return $true }
    if (Test-Path (Join-Path $winfspDef "winfsp-x86.dll")) { return $true }
    return $false
}

function Get-ArchSuffix {
  if ([Environment]::Is64BitOperatingSystem) { return "x64" } else { return "x86" }
}

# require admin to install; prompt/elevate if not admin
if (-not (Is-Admin)) {
  Write-Host "Elevation required. Restarting elevated..."
  $arg = @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", $PSCommandPath, "-PackageDir", $PackageDir)
  Start-Process -FilePath "powershell" -ArgumentList $arg -Verb RunAs -Wait
  exit $LASTEXITCODE
}

$arch = Get-ArchSuffix()
Write-Host "Detected OS architecture: $arch"

$bundleRoot = $PSScriptRoot
Write-Host "Bundle root: $bundleRoot"

# install target
$installDir = Join-Path $env:ProgramFiles "Mimic"
try {
  if (-not (Test-Path $installDir)) { New-Item -ItemType Directory -Path $installDir -Force | Out-Null }
} catch {
  Write-Error "Failed to create install dir $installDir: $_"
  exit 1
}

# copy expected bundle files (best-effort)
$files = @("mimic.exe","README.md","LICENCE","example-config.toml","INSTALL.md","install.ps1")
foreach ($f in $files) {
  $src = Join-Path $bundleRoot $f
  if (Test-Path $src) {
    try {
      Copy-Item -Path $src -Destination $installDir -Force -ErrorAction Stop
      Write-Host "Copied $f -> $installDir"
    } catch {
      Write-Warning "Failed to copy $f: $_"
    }
  } else {
    Write-Host "Not found in bundle (skipping): $f"
  }
}

if (Is-WinFspInstalled) {
  Write-Host "WinFsp already installed. Skipping."
  exit 0
}

$arch = Get-ArchSuffix
Write-Host "Detected architecture: $arch"
$apiUrl = "https://api.github.com/repos/billziss-gh/winfsp/releases/latest"

try {
  $rel = Invoke-RestMethod -Uri $apiUrl -UseBasicParsing -ErrorAction Stop
} catch {
  Write-Error "Failed to query WinFsp releases: $_"
  Write-Host "Please install WinFsp manually: https://github.com/billziss-gh/winfsp/releases"
  exit 2
}

$asset = $rel.assets | Where-Object { $_.name -match "(?i)winfsp.*$arch.*\.msi$" } | Select-Object -First 1
if (-not $asset) {
  Write-Error "No suitable WinFsp MSI found in latest release. See: $($rel.html_url)"
  exit 2
}

$msiUrl = $asset.browser_download_url
$tmp = Join-Path $env:TEMP ("winfsp-install-$([guid]::NewGuid()).msi")
Write-Host "Downloading WinFsp MSI..."
try {
  Invoke-WebRequest -Uri $msiUrl -OutFile $tmp -UseBasicParsing -ErrorAction Stop
} catch {
  Write-Error "Download failed: $_"
  exit 2
}

# compute and display SHA256 so operator can verify if desired
try {
  $hash = Get-FileHash -Algorithm SHA256 -Path $tmp
  Write-Host "MSI path: $tmp"
  Write-Host "SHA256: $($hash.Hash)"
} catch {
  Write-Warning "Could not compute hash of $tmp: $_"
}

Write-Host "Installing WinFsp MSI silently (msiexec) ..."
$msiArgs = "/i", "`"$msiPath`"", "/qn", "/norestart"
$proc = Start-Process -FilePath "msiexec.exe" -ArgumentList $msiArgs -Wait -PassThru

Write-Host "Installing WinFsp silently..."
$proc = Start-Process -FilePath "msiexec.exe" -ArgumentList "/i", "`"$tmp`"", "/qn", "/norestart" -Wait -PassThru
if ($proc.ExitCode -ne 0) {
  Write-Error "msiexec failed with exit code $($proc.ExitCode)"
  Remove-Item -Force $tmp -ErrorAction SilentlyContinue
  exit $proc.ExitCode
}

Remove-Item -Force $tmp -ErrorAction SilentlyContinue

if (Is-WinFspInstalled) {
  Write-Host "WinFsp installed successfully."
  Write-Host "Installation complete. Files are in: $installDir"
  exit 0
} else {
  Write-Error "WinFsp install finished but runtime not detected. You may need to reboot."
  exit 3
}