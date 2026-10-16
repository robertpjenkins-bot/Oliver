#!/bin/bash
/usr/bin/etros-sort-roms >> /userdata/system/logs/etros.log 2>&1
# Show the newly sorted games.
curl -s http://localhost:1234/reloadgames > /dev/null 2>&1
exit 0
