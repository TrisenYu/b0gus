# Use "Set-ExecutionPolicy RemoteSigned -Scope CurrentUser" before ".\gen.ps1".
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

antlr4 | grep -in "antlr" > $null 2>&1
if (-not $?) {
    Write-Error "require ANTLR Parser Generator in your system"
    exit 1
}

# version number must >= 4.13
$version = (antlr4 2>&1) -match 'version (\d+)\.(\d+)' | Out-Null
if ($matches[1] -lt 4 -or $matches[2] -lt 13) {
    Write-Error "require version of antlr >= 4.13"
    exit 1
}

go version | grep -in "go" > $null 2>&1
if (-not $?) {
    Write-Error "require go in your system"
    exit 1
}

$Lang = "Go"

Get-ChildItem -Filter *.g4 | ForEach-Object {
    antlr4 -Werror -Dlanguage="$Lang" -no-visitor -listener -o "$PWD/" -Xexact-output-dir $_.FullName
}

if ($Lang -eq "Go") {
    Get-ChildItem -Recurse -File | Where-Object {
        $_.FullName -notmatch "\\gen" -and (Select-String -Path $_ -Pattern "package parser" -Quiet)
    } | ForEach-Object {
        (Get-Content -Raw $_) -replace "package parser", "package main" | Set-Content $_
    }
}

go generate

Remove-Item ./*.interp -Force -ErrorAction SilentlyContinue
Remove-Item ./*.tokens -Force -ErrorAction SilentlyContinue