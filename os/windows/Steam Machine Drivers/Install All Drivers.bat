@echo off
rem Steam Machine driver installer: installs every driver in this folder.
rem Put the driver zips (or .exe / .msi installers) from Valve's Steam Machine
rem Windows drivers page next to this file, then double-click it.
rem Windows asks once "Do you want to allow this app to make changes" - click Yes.
cd /d "%~dp0"
powershell -NoProfile -ExecutionPolicy Bypass -Command "Start-Process powershell -Verb RunAs -ArgumentList '-NoProfile -ExecutionPolicy Bypass -File \"%~dp0install-drivers.ps1\"'"
