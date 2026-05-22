# Use "Set-ExecutionPolicy RemoteSigned -Scope CurrentUser" before ".\gen.ps1".
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Test-CommandExists {
    param($command)
    $exists = $null -ne (Get-Command $command -ErrorAction SilentlyContinue)
    return $exists
}

# Verify antlr4 is installed
if (-not (Test-CommandExists antlr4)) {
    Write-Error "require ANTLR Parser Generator in your system"
    exit 1
}

# Verify ANTLR version >= 4.13
$antlrOutput = antlr4 2>&1 | Out-String
if ($antlrOutput -match 'version (\d+)\.(\d+)') {
    $verMajor = [int]$matches[1]
    $verMinor = [int]$matches[2]
    if ($verMajor -lt 4 -or ($verMajor -eq 4 -and $verMinor -lt 13)) {
        Write-Error "require version of antlr >= 4.13"
        exit 1
    }
} else {
    Write-Error "require version of antlr >= 4.13"
    exit 1
}

# Verify go is present
if (-not (Test-CommandExists go)) {
    Write-Error "require go in your system"
    exit 1
}

$Lang = "Go"

# Generate code from all .g4 files
Get-ChildItem -Filter *.g4 -File | ForEach-Object {
    antlr4 -Werror -Dlanguage="$Lang" -no-visitor -listener -o $PWD.Path -Xexact-output-dir $_.FullName
}

# Replace package parser with package main (exclude gen directories)
if ($Lang -eq "Go") {
    Get-ChildItem -Recurse -File | Where-Object {
        $_.DirectoryName -notmatch "[\\/]gen[\\/]" -and
        (Select-String -Path $_.FullName -Pattern "package parser" -Quiet -SimpleMatch)
    } | ForEach-Object {
        $content = Get-Content -Raw $_.FullName
        $updated = $content -replace "package parser", "package main"
        Set-Content -Path $_.FullName -Value $updated -Encoding UTF8
    }
}

# Add build tags to rdt*.go files if not present
$buildTag = @'
//go:build tools
// +build tools

'@

Get-ChildItem -Filter rdt*.go -File | ForEach-Object {
    $content = Get-Content -Raw $_.FullName
    if ($content -notmatch [regex]::Escape('//go:build tools')) {
        $newContent = $buildTag + $content
        Set-Content -Path $_.FullName -Value $newContent -Encoding UTF8
    }
}

go generate

# Cleanup temporary files
Remove-Item *.interp -Force -ErrorAction SilentlyContinue
Remove-Item *.tokens -Force -ErrorAction SilentlyContinue
