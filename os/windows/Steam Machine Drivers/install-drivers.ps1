# Steam Machine driver installer (started by "Install All Drivers.bat", as
# administrator). For every driver in this folder:
#   a .zip          is unpacked into a folder of its own, then installed as below
#   setup.exe, an .exe or .msi installer   is run (quietly where it can be)
#   .inf files (a driver with no installer) are added with pnputil
# A log is saved next to it: install-log.txt.
$ErrorActionPreference = 'Continue'
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $here
Start-Transcript -Path (Join-Path $here 'install-log.txt') -Append | Out-Null
Write-Host "`n  Steam Machine driver installer`n" -ForegroundColor Cyan

# 1. Unpack the zips.
Get-ChildItem -Path $here -Filter *.zip -File | ForEach-Object {
    $dest = Join-Path $here ($_.BaseName)
    if (-not (Test-Path $dest)) {
        Write-Host "Unpacking $($_.Name)"
        Expand-Archive -Path $_.FullName -DestinationPath $dest -Force
    }
}

# 2. Installers: loose ones here, and one per driver folder (its top-most).
function Install-One($file) {
    Write-Host "Installing $($file.Name)" -ForegroundColor Yellow
    switch ($file.Extension.ToLower()) {
        '.msi' { Start-Process msiexec.exe -ArgumentList "/i `"$($file.FullName)`" /passive /norestart" -Wait }
        '.exe' { Start-Process $file.FullName -WorkingDirectory $file.DirectoryName -Wait }
    }
}
$done = @{}
Get-ChildItem -Path $here -File | Where-Object { $_.Extension -in '.exe', '.msi' } | ForEach-Object { Install-One $_ }
Get-ChildItem -Path $here -Directory | ForEach-Object {
    $folder = $_
    $installer = Get-ChildItem -Path $folder.FullName -Recurse -File -Include setup.exe, *.msi, *.exe |
        Sort-Object { ($_.FullName -split '\\').Count }, { $_.Name -ne 'setup.exe' } | Select-Object -First 1
    if ($installer) {
        Install-One $installer
        $done[$folder.FullName] = $true
    }
}

# 3. Drivers with no installer: their .inf files.
Get-ChildItem -Path $here -Directory | Where-Object { -not $done[$_.FullName] } | ForEach-Object {
    if (Get-ChildItem -Path $_.FullName -Recurse -Filter *.inf -File) {
        Write-Host "Adding the drivers in $($_.Name)" -ForegroundColor Yellow
        pnputil.exe /add-driver (Join-Path $_.FullName '*.inf') /subdirs /install
    }
}

Write-Host "`n  Done. Restart the Steam Machine so every driver starts.`n  (What happened is in install-log.txt.)" -ForegroundColor Green
Stop-Transcript | Out-Null
Read-Host "Press Enter to close"
