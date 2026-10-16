ETROS - YOUR GAMES
==================

Put your own ROM files in the folder for their console, then put the card
back in the device and switch it on. Etros adds them to the game list and
empties these folders.

  snes   Super Nintendo (.sfc .smc)     gba   Game Boy Advance (.gba)
  gb     Game Boy (.gb)                 gbc   Game Boy Color (.gbc)
  nes    NES (.nes)                     n64   Nintendo 64 (.z64 .n64)
  nds    Nintendo DS (.nds)             megadrive  Mega Drive / Genesis
  psx    PlayStation (.chd .cue+.bin .pbp)

Any other Batocera system works too: make a folder with its name.

ROM HACKS
Put the hack's .bps or .ips patch in the same folder as the game it
patches, e.g. a Super Metroid hack next to your Super Metroid .sfc. The
patched game is added under the patch's name. A .bps patch finds the right
game by itself; an .ips patch needs to be the only game in its folder.

If a patch doesn't match your copy of the game, the files stay here and
Etros tries again on the next start. The reason is in system/logs/etros.log on the SHARE drive
(or \\ETROS\share\system\logs over the network).

Only use games you own.
