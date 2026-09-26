#!/usr/bin/env pwsh
#Requires -Version 7.0
<#
.SYNOPSIS
    Canonical Q0-Q6 verifier entrypoint (IMP-000). Runs the Go verifier under
    server/cmd/verify and writes verify-report.json.

.DESCRIPTION
    Runs unchanged on Linux and Windows under pwsh 7.6.6. The Go verifier does
    all checking; this script only bootstraps the tool boundary.

    -UnityResultsDir   directory containing Unity test-results XML (CI only)
    -LocalDeferMissing local-only: missing Unity editor / PostgreSQL /
                       Windows-only binaries / cgo compile to
                       DEFERRED(local-missing) in verify-report.json.
                       CI (GITHUB_ACTIONS=true) must never pass it.
    On Linux without THINHTHAN_TEST_PG_DSN it starts the pinned postgres:18.6
    digest with `docker run` when Docker is available.

.EXAMPLE
    pwsh -NoProfile -File scripts/verify.ps1 -LocalDeferMissing
#>
[CmdletBinding()]
param(
    [string]$UnityResultsDir = "",
    [switch]$LocalDeferMissing,
    [switch]$MergeReports,              # internal: evidence job merges reports
    [string]$MergeDir = "",
    [string]$TaskID = "",
    [string]$ReportOut = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# CI must never pass -LocalDeferMissing.
if ($env:GITHUB_ACTIONS -eq 'true' -and $LocalDeferMissing) {
    Write-Error "verify.ps1: -LocalDeferMissing is forbidden in CI"
    exit 2
}

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $RepoRoot

# ---------------------------------------------------------------------------
# Optional local PostgreSQL: Linux without THINHTHAN_TEST_PG_DSN starts the
# pinned postgres:18.6 image digest when docker exists.
# ---------------------------------------------------------------------------
$script:pgContainer = ""
function Start-LocalPostgres {
    if ($env:THINHTHAN_TEST_PG_DSN) { return }
    if (-not ($IsLinux -or $env:OS -match 'Linux')) { return }
    $docker = Get-Command docker -ErrorAction SilentlyContinue
    if (-not $docker) { return }
    $image = "postgres:18.6@sha256:5a5a84b19854a9ffaa54082c166ff4ec27473a361e496e5ea167f298f2da9722"
    try {
        docker image inspect $image | Out-Null
    } catch {
        docker pull $image | Out-Null
    }
    $script:pgContainer = (docker run -d --rm -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=thinhthan_test -p 0:5432 $image).Trim()
    if ($script:pgContainer) {
        $port = (docker port $script:pgContainer 5432/tcp).Split(':')[-1]
        $env:THINHTHAN_TEST_PG_DSN = "postgres://postgres:postgres@127.0.0.1:$port/thinhthan_test?sslmode=disable"
        Write-Verbose "started pinned postgres on port $port"
    }
}
function Stop-LocalPostgres {
    if ($script:pgContainer) {
        docker rm -f $script:pgContainer | Out-Null
    }
}

# ---------------------------------------------------------------------------
# Locate the pinned Go toolchain (CI puts it on PATH; local uses PATH too).
# ---------------------------------------------------------------------------
$go = Get-Command go -ErrorAction SilentlyContinue
if (-not $go) {
    Write-Error "verify.ps1: go 1.27.1 not on PATH (install the pinned toolchain)"
    exit 2
}
$env:THINHTHAN_GO_VERSION = (& go version)

try {
    if (-not $LocalDeferMissing) { Start-LocalPostgres }

    # Resolve path args against the repo root before Push-Location server —
    # relative paths would otherwise land under server/.
    if ($UnityResultsDir) { $UnityResultsDir = [IO.Path]::GetFullPath($UnityResultsDir) }
    if ($ReportOut) { $ReportOut = [IO.Path]::GetFullPath($ReportOut) }
    if ($MergeDir) { $MergeDir = [IO.Path]::GetFullPath($MergeDir) }

    $args = @('run', './cmd/verify')
    if ($LocalDeferMissing) { $args += '-local-defer' }
    if ($UnityResultsDir) { $args += @('-unity-results-dir', $UnityResultsDir) }
    if ($ReportOut) { $args += @('-report-out', $ReportOut) }
    if ($MergeReports -and $MergeDir) {
        $args += @('-merge-reports', $MergeDir)
        if ($TaskID) { $args += @('-task', $TaskID) }
    }

    Push-Location (Join-Path $RepoRoot 'server')
    try {
        & go @args
        $exit = $LASTEXITCODE
    } finally {
        Pop-Location
    }
    exit $exit
} finally {
    Stop-LocalPostgres
}
