@echo off
rem Etros: Windows's installer (WinPE) starts this. It puts Windows 11 on
rem the WINDOWS partition from the image Ollie OS copied there, makes
rem Windows's boot files on WINBOOT and restarts into Windows.
wpeinit
echo.
echo  Etros is installing Windows 11. This takes a while: don't switch off.
echo.
set SRC=
set ESP=
for %%d in (C D E F G H I J K L M N O P Q R S T U V W Y Z) do (
    if exist %%d:\etros-winboot.txt set ESP=%%d:
)
if "%ESP%"=="" goto fail
set /p WIMNAME=<%ESP%\etros-win-image.txt
set /p IDX=<%ESP%\etros-win-index.txt
for %%d in (C D E F G H I J K L M N O P Q R S T U V W Y Z) do (
    if exist %%d:\etros-win\%WIMNAME% set SRC=%%d:\etros-win\%WIMNAME%& set WIN=%%d:
)
if "%SRC%"=="" goto fail
dism /Apply-Image /ImageFile:%SRC% /Index:%IDX% /ApplyDir:%WIN%\ || goto fail
mkdir %WIN%\Windows\Panther 2>nul
copy /y %ESP%\unattend.xml %WIN%\Windows\Panther\unattend.xml || goto fail
mkdir %WIN%\Windows\Setup\Scripts 2>nul
copy /y "%ESP%\Back to Etros.bat" "%WIN%\Windows\Setup\Scripts\Back to Etros.bat"
bcdboot %WIN%\Windows /s %ESP% /f UEFI || goto fail
del /f /q %SRC%
rmdir %WIN%\etros-win 2>nul
echo installed> %ESP%\etros-win-installed.txt
wpeutil reboot
exit

:fail
echo failed> %ESP%\etros-win-failed.txt
echo.
echo  Windows 11 couldn't be installed. Switch off, put the card in a computer
echo  and run Etros Updater: it sets the card back to starting Etros.
pause
