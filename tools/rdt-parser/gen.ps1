# Use "Set-ExecutionPolicy RemoteSigned -Scope CurrentUser" before ".\gen.ps1".
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Check if ANTLR is available
try {
    antlr4 2>&1 | Out-Null
}
catch {
    Write-Error "require ANTLR Parser Generator in your system"
    exit 1
}

# Validate ANTLR version >= 4.13
$antlrVersion = antlr4 2>&1 | Select-String -Pattern 'version (\d+\.\d+)'
if (-not $antlrVersion) {
    Write-Error "Failed to detect ANTLR version"
    exit 1
}

$ver = $antlrVersion.Matches.Groups[1].Value -split '\.'
$major = [int]$ver[0]
$minor = [int]$ver[1]
if ($major -lt 4 -or ($major -eq 4 -and $minor -lt 13)) {
    Write-Error "require version of antlr >= 4.13"
    exit 1
}

# Check if Go is installed
try {
    go version 2>&1 | Out-Null
}
catch {
    Write-Error "require go in your system"
    exit 1
}

# Set target language and find all .g4 files
$Lang = "Go"
$g4Files = Get-ChildItem -Filter "*.g4" -File

# Generate code from each .g4 file
foreach ($file in $g4Files) {
    antlr4 -Werror -Dlanguage="$Lang" -no-visitor -listener "$file.FullName" -o "$PWD" -Xexact-output-dir
}

# Replace package name from 'parser' to 'main' for Go target
if ($Lang -eq "Go") {
    $goFiles = Get-ChildItem -Recurse -Filter "*.go" -File |
        Select-String -Pattern "package parser" |
        Where-Object { $_.Path -notmatch "gen" } |
        ForEach-Object { $_.Path }
    if ($goFiles) {
        foreach ($f in $goFiles) {
            (Get-Content $f -Raw) -replace "package parser", "package main" | Set-Content $f
        }
    }
}

# Run Go generate
go generate

# Clean up temporary ANTLR files
Remove-Item -Path "./*.interp", "./*.tokens" -ErrorAction SilentlyContinue
