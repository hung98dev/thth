# ADR-0077: detached Unity materialization for the `Unity (Windows)` job. The
# launcher step starts this script right after the `client/Library` cache
# restore and licence env are ready; the foreground then installs Go, restores
# the Go module/build cache and computes the Unity mode plan while the editor
# materializes here. Completion is signalled only through
# $env:RUNNER_TEMP\unity-materialize-state\exitcode.txt — the join step trusts
# that sentinel, not the process exit status. A runner that is cancelled or
# loses the detached process leaves no sentinel; the join then runs one
# synchronous materialization instead of failing blind.
$ErrorActionPreference = 'Continue'
$state = Join-Path $env:RUNNER_TEMP 'unity-materialize-state'
New-Item -ItemType Directory -Force -Path $state | Out-Null
$rc = 0
try {
    Set-Location $env:GITHUB_WORKSPACE
    . "$env:GITHUB_WORKSPACE\.devin\scripts\cache_telemetry.ps1"
    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    $libHit = if ($env:CACHE_HIT_LIBRARY -eq 'true') { 'hit' } else { 'miss' }
    $ulf = 'C:\ProgramData\Unity\Unity_lic.ulf'
    $logPath = "$env:RUNNER_TEMP\unity-materialize.log"
    $proj = "$env:GITHUB_WORKSPACE\client"
    $ok = $false
    $prevCompile = $false
    foreach ($i in 1..5) {
        Remove-Item $logPath -Force -ErrorAction SilentlyContinue
        $uargs = @('-batchmode', '-nographics', '-projectPath', $proj, '-quit', '-logFile', $logPath)
        if (-not (Test-Path $ulf)) {
            $uargs += @('-serial', "$env:UNITY_SERIAL", '-username', "$env:UNITY_EMAIL", '-password', "$env:UNITY_PASSWORD")
        }
        $p = Start-Process -FilePath $env:UNITY_EDITOR_PATH -ArgumentList $uargs -PassThru -Wait -NoNewWindow
        $p_rc = $p.ExitCode
        $log = ''
        if (Test-Path $logPath) {
            $log = Get-Content $logPath -Raw
            Get-Content $logPath -Tail 60
        }
        if ($log -match 'Exiting batchmode successfully' -and $log -notmatch 'Scripts have compiler errors' -and (Test-Path "$proj\Library\PackageCache")) { $ok = $true; break }
        if ($log -match 'license|licence' -and $log -match 'fail|error|invalid') { Remove-Item $ulf -Force -ErrorAction SilentlyContinue }
        if ($p_rc -eq 198) { Write-Host "unity licence activation failed deterministically (rc=$p_rc) - not retrying"; break }
        # ADR-0077: compiler errors retry at once (ADR-0073 item 5) and stop
        # when they repeat on the next attempt.
        $compile = $log -match 'Scripts have compiler errors'
        if ($compile -and $prevCompile) { Write-Host 'unity materialization: compiler errors on two consecutive attempts - deterministic, not retrying'; break }
        $prevCompile = $compile
        Write-Host "unity materialization attempt $i failed (rc=$p_rc); $(if ($compile) { 'compiler errors; retrying now' } else { 'retrying after 60s' })"
        if ($i -lt 5 -and -not $compile) { Start-Sleep -Seconds 60 }
    }
    Write-CacheTelemetry -Step 'unity-library' -Result $libHit -WallSeconds $sw.Elapsed.TotalSeconds
    if (-not $ok) { $rc = 1 }
} catch {
    $_ | Out-String | Write-Host
    $rc = 1
}
Set-Content -Path (Join-Path $state 'exitcode.txt') -Value "$rc" -NoNewline
exit $rc
