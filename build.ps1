# Build script for Windows
Write-Host "Building HalluciDetect..." -ForegroundColor Cyan

$env:CGO_ENABLED = "0"
go test -v ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host "Tests failed!" -ForegroundColor Red
    exit 1
}

go build -v -ldflags="-s -w" -o hallucidetect.exe ./cmd/hallucidetect
if ($LASTEXITCODE -eq 0) {
    Write-Host "Build complete: hallucidetect.exe" -ForegroundColor Green
    Write-Host "Try running: .\hallucidetect.exe benchmark" -ForegroundColor Yellow
} else {
    Write-Host "Build failed!" -ForegroundColor Red
}
