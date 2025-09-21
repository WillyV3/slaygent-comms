Home
Install Tmux
Theming & Customizing
Plugins & Tools
Submit Cheats

Docker

Kubectl
Tmux Cheat Sheet & Quick Reference
Search cheats here

Sessions
tmux
tmux new
tmux new-session
new
Start a new session

tmux new-session -A -s mysession
Start a new session or attach to an existing session named mysession

tmux new -s mysession
new -s mysession
Start a new session with the name mysession

kill-session
kill/delete the current session

tmux kill-ses -t mysession
tmux kill-session -t mysession
kill/delete session mysession

tmux kill-session -a
kill/delete all sessions but the current

tmux kill-session -a -t mysession
kill/delete all sessions but mysession

Rename session

Detach from session

attach -d
Detach others on the session (Maximize window by detach other clients)

tmux ls
tmux list-sessions
Show all sessions

tmux a
tmux at
tmux attach
tmux attach-session
Attach to last session

tmux a -t mysession
tmux at -t mysession
tmux attach -t mysession
tmux attach-session -t mysession
Attach to a session with the name mysession

Session and Window Preview

Move to previous session

Move to next session


Windows
tmux new -s mysession -n mywindow
start a new session with the name mysession and window mywindow

Create window

Rename current window

Close current window

List windows

Previous window

Next window

Switch/select window by number

Toggle last active window

swap-window -s 2 -t 1
Reorder window, swap window number 2(src) and 1(dst)

swap-window -t -1
Move current window to the left by one position

move-window -s src_ses:win -t target_ses:win
movew -s foo:0 -t bar:9
movew -s 0:0 -t 1:9
Move window from source to target

move-window -s src_session:src_window
movew -s 0:9
Reposition window in the current session

move-window -r
movew -r
Renumber windows to remove gap in the sequence

Panes
Toggle last active pane

split-window -h
Split the current pane with a vertical line to create a horizontal layout

split-window -v
Split the current with a horizontal line to create a vertical layout

join-pane -s 2 -t 1
Join two windows as panes (Merge window 2 to window 1 as panes)

join-pane -s 2.1 -t 1.0
Move pane from one window to another (Move pane 1 from window 2 to pane after 0 of window 1)

Move the current pane left

Move the current pane right

Switch to pane to the direction

setw synchronize-panes
Toggle synchronize-panes(send command to all panes)

Toggle between pane layouts

Switch to next pane

Show pane numbers

Switch/select pane by number

Toggle pane zoom

Convert pane into a window

Resize current pane height(holding second key is optional)

Resize current pane width(holding second key is optional)

Close current pane


Copy Mode
setw -g mode-keys vi
use vi keys in buffer

Enter copy mode

Enter copy mode and scroll one page up

Quit mode

Go to top line

Go to bottom line

Scroll up

Scroll down

Move cursor left

Move cursor down

Move cursor up

Move cursor right

Move cursor forward one word at a time

Move cursor backward one word at a time

Search forward

Search backward

Next keyword occurance

Previous keyword occurance

Start selection

Clear selection

Copy selection

Paste contents of buffer_0

show-buffer
display buffer_0 contents

capture-pane
copy entire visible contents of pane to a buffer

list-buffers
Show all buffers

choose-buffer
Show all buffers and paste selected

save-buffer buf.txt
Save buffer contents to buf.txt

delete-buffer -b 1
delete buffer_1

Misc
Enter command mode

set -g OPTION
Set OPTION for all sessions

setw -g OPTION
Set OPTION for all windows

set mouse on
Enable mouse mode

Help
tmux list-keys
list-keys
List key bindings(shortcuts)

tmux info
Show every session, window, pane, etc...

©2025 Tmuxcheatsheet.com Privacy Policy

