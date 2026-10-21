@echo off
rem Etros: the Pi 5 starts Windows 11 while autoboot.txt on the BATOCERA
rem partition says so; take it away and restart, back into Etros.
for %%d in (C D E F G H I J K L M N O P Q R S T U V W Y Z) do (
    if exist %%d:\etros\etros.squashfs if exist %%d:\autoboot.txt del /f /q %%d:\autoboot.txt
)
shutdown /r /t 0
