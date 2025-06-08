# Streamshower

A simple frontend to show live streams in a TUI that also acts as a daemon to
prevent excessive API calls.

## Help

See the commands `:help` and `:map` inside the TUI.

## Navigation
standard vim navigation: `jkl` or arrow keys + enter

`f` to open a filter dialog

`F` to clear filter

Twitch filters: pressing `!` inverts the showing matches

Strims filters: numbers work as a minimum-rustler threshold

filter window supports regular readline keys such as ctrl-u to clear, ctrl-a to
go to beginning of line, ctrl-e to go to end of line etc

`i` to fullscreen the Stream Info

`o` to switch between lists

`r` to force the server to refresh data

`q` to quit
