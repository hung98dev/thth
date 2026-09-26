#!/usr/bin/env pwsh
#Requires -Version 7.0
<#
.SYNOPSIS
    Canonical protobuf codegen (IMP-061).

.DESCRIPTION
    Regenerates the canonical wire code from proto/thinhthan/v1/ using the
    pinned toolchain: protoc 36.2 and protoc-gen-go v1.36.12
    (docs/00_context/technology_versions.md). Runs unchanged on Linux and
    Windows under pwsh 7.6.6 (ADR-0050: no Bash wrapper).

    Writes only generated `*.pb.go` (server/internal/protocol/v1/) and `*.cs`
    (client/Assets/Scripts/Protocol/) files. Never deletes or rewrites
    `ThinhThan.Protocol.asmdef` (IMP-000, ADR-0068) or any `.meta` file —
    `.meta` for generated C# is editor-materialized in CI and committed from
    the `unity-materialized-<os>` artifact (agent_execution_protocol.md §4b,
    ADR-0072).

    Every generated `.cs` file begins with the deterministic CODE-004 header:
    `#nullable disable` + the protobuf `#pragma warning disable` set
    (engineering_conventions.md §2.7).

    Pinned tools live under the gitignored `tools/` directory; they are
    downloaded with sha256 verification (protoc) or the pinned module version
    (protoc-gen-go). PATH tools matching the exact pin are reused.

.EXAMPLE
    pwsh -NoProfile -File scripts/codegen.ps1
.EXAMPLE
    pwsh -NoProfile -File scripts/codegen.ps1 -RepoRoot /path/to/repo
#>
[CmdletBinding()]
param(
    [string]$RepoRoot = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if ($RepoRoot) {
    $Root = (Resolve-Path $RepoRoot).Path
} else {
    $Root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
}
$ProtoDir   = Join-Path $Root 'proto'
$GoOutDir   = Join-Path $Root 'server/internal/protocol/v1'
$CsOutDir   = Join-Path $Root 'client/Assets/Scripts/Protocol'
$ToolsDir   = Join-Path $Root 'tools/protobuf'

$ProtocVersion      = '36.2'
$ProtocGenGoVersion = 'v1.36.12'

$IsWindowsHost = $IsWindows -or ($env:OS -eq 'Windows_NT')
if ($IsWindowsHost) {
    $ProtocZipName   = "protoc-$ProtocVersion-win64.zip"
    $ProtocZipSha256 = 'f0c128dc0d8492eceece83bb459a4c0e316764b929ffbf1aa416357fd644edd3'
    $ProtocBinName   = 'protoc.exe'
    $GenGoBinName    = 'protoc-gen-go.exe'
} elseif ($IsLinux) {
    $ProtocZipName   = "protoc-$ProtocVersion-linux-x86_64.zip"
    $ProtocZipSha256 = '121f6c7afe1d4d0e3ea6aab9432038599250134cbf4474cb1167d2c7decd4278'
    $ProtocBinName   = 'protoc'
    $GenGoBinName    = 'protoc-gen-go'
} else {
    throw "codegen.ps1: unsupported host OS (pinned CI hosts are Linux/Windows, ADR-0058)"
}

$ToolsBin   = Join-Path $ToolsDir 'bin'
$ProtocDir_ = Join-Path $ToolsDir 'protoc'
$ProtocBin  = Join-Path $ProtocDir_ "bin/$ProtocBinName"
$GenGoBin   = Join-Path $ToolsBin $GenGoBinName

# ---------------------------------------------------------------------------
# Tool resolution: tools/ cache -> PATH (exact pin match) -> download/install.
# ---------------------------------------------------------------------------
function Get-BinVersion([string]$bin) {
    try {
        return (& $bin --version) -join ' '
    } catch {
        return $null
    }
}

function Find-PinOnPath([string]$name, [string]$wantPrefix, [string]$want) {
    $cmd = Get-Command $name -ErrorAction SilentlyContinue
    if (-not $cmd) { return $null }
    $ver = Get-BinVersion $cmd.Source
    if ($ver -and $ver.StartsWith($wantPrefix) -and $ver.Contains($want)) {
        return $cmd.Source
    }
    return $null
}

# --- protoc -----------------------------------------------------------------
if (Test-Path $ProtocBin) {
    $ver = Get-BinVersion $ProtocBin
    if ($ver -ne "libprotoc $ProtocVersion") {
        Remove-Item -Recurse -Force $ProtocDir_
    }
}
if (-not (Test-Path $ProtocBin)) {
    $onPath = Find-PinOnPath 'protoc' 'libprotoc ' $ProtocVersion
    if ($onPath) {
        $ProtocBin = $onPath
    } else {
        New-Item -ItemType Directory -Force -Path $ToolsDir | Out-Null
        $zipPath = Join-Path $ToolsDir $ProtocZipName
        if (-not (Test-Path $zipPath)) {
            Invoke-WebRequest -UseBasicParsing -OutFile $zipPath -Uri `
                "https://github.com/protocolbuffers/protobuf/releases/download/v$ProtocVersion/$ProtocZipName"
        }
        $actual = (Get-FileHash -Algorithm SHA256 $zipPath).Hash.ToLowerInvariant()
        if ($actual -ne $ProtocZipSha256) {
            Remove-Item -Force $zipPath
            throw "protoc archive sha256 mismatch: $actual != $ProtocZipSha256"
        }
        if (Test-Path $ProtocDir_) { Remove-Item -Recurse -Force $ProtocDir_ }
        Expand-Archive -Path $zipPath -DestinationPath $ProtocDir_
        if (-not $IsWindowsHost) {
            chmod +x (Join-Path $ProtocDir_ "bin/$ProtocBinName")
        }
    }
}
if ((Get-BinVersion $ProtocBin) -ne "libprotoc $ProtocVersion") {
    throw "protoc version mismatch: wanted libprotoc $ProtocVersion"
}

# --- protoc-gen-go ----------------------------------------------------------
if (Test-Path $GenGoBin) {
    $ver = Get-BinVersion $GenGoBin
    if ($ver -ne "protoc-gen-go $ProtocGenGoVersion") {
        Remove-Item -Force $GenGoBin
    }
}
if (-not (Test-Path $GenGoBin)) {
    $onPath = Find-PinOnPath 'protoc-gen-go' 'protoc-gen-go ' $ProtocGenGoVersion
    if ($onPath) {
        $GenGoBin = $onPath
    } else {
        $go = Get-Command go -ErrorAction SilentlyContinue
        if (-not $go) { throw "go not on PATH; install Go 1.27.1 (technology_versions.md)" }
        New-Item -ItemType Directory -Force -Path $ToolsBin | Out-Null
        $env:GOBIN = $ToolsBin
        & $go.Source install "google.golang.org/protobuf/cmd/protoc-gen-go@$ProtocGenGoVersion"
        if ($LASTEXITCODE -ne 0) { throw "go install protoc-gen-go failed ($LASTEXITCODE)" }
    }
}
if ((Get-BinVersion $GenGoBin) -ne "protoc-gen-go $ProtocGenGoVersion") {
    throw "protoc-gen-go version mismatch: wanted $ProtocGenGoVersion"
}
$env:PATH = (Split-Path -Parent $GenGoBin) + [IO.Path]::PathSeparator + $env:PATH

# ---------------------------------------------------------------------------
# Regenerate. Orphan cleanup deletes only generated globs (*.pb.go, *.cs);
# the asmdef and every .meta are owned elsewhere and never touched.
# ---------------------------------------------------------------------------
New-Item -ItemType Directory -Force -Path $GoOutDir, $CsOutDir | Out-Null
Remove-Item -Force -ErrorAction SilentlyContinue `
    (Join-Path $GoOutDir '*.pb.go'), `
    (Join-Path $CsOutDir '*.cs')

# Deterministic input order (sorted file names, identical on every host).
$protoFiles = Get-ChildItem -Path (Join-Path $ProtoDir 'thinhthan/v1') -Filter '*.proto' |
    Sort-Object Name | ForEach-Object { "thinhthan/v1/$($_.Name)" }
if ($protoFiles.Count -eq 0) { throw "no .proto files under proto/thinhthan/v1" }

& $ProtocBin `
    -I $ProtoDir `
    --go_out="$(Join-Path $Root 'server')" --go_opt=module=thinhthan `
    --csharp_out=$CsOutDir `
    $protoFiles
if ($LASTEXITCODE -ne 0) { throw "protoc failed with exit code $LASTEXITCODE" }

# ---------------------------------------------------------------------------
# CODE-004: prepend the deterministic generated-C# header. protoc's own
# `#pragma warning disable 1591, 0612, 3021, 8981` remains; the prepended
# block makes `#nullable disable` the first line. UTF-8 no BOM, LF endings.
# ---------------------------------------------------------------------------
$csHeader = "#nullable disable`n#pragma warning disable 1591, 0612, 3021, 8981`n"
$utf8NoBom = New-Object System.Text.UTF8Encoding $false
foreach ($cs in (Get-ChildItem -Path $CsOutDir -Filter '*.cs' | Sort-Object Name)) {
    $body = [IO.File]::ReadAllText($cs.FullName)
    $body = $body -replace "`r`n?", "`n"
    if (-not $body.StartsWith('#nullable disable')) {
        $body = $csHeader + $body
    }
    [IO.File]::WriteAllText($cs.FullName, $body, $utf8NoBom)
}

Write-Host "codegen: $($protoFiles.Count) proto files -> server/internal/protocol/v1/ + client/Assets/Scripts/Protocol/"
