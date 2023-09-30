# Streamshower

A simple frontend + backend to show live streams in a TUI.

## Usage
Running Streamshower as a server in the background with the `-b` flag will fetch
and serve Twitch and Strims data every refresh interval (`-r`) and will then
serve this data at the specified address (`-a`).

Running the program normally, with the same address as the server, then requests
data and displays it in an interactive TUI (as opposed to making API calls to
Twitch and Strims every time the TUI is shown).


## Config
All settings are stored in a config.json file

To generate this file, simply run the program in background mode (`-b`)  and
provide Client ID, Client Secret, and User ID when prompted.

Explanation:

`Client ID`: The api key of your [Twitch project](dev.twitch.tv)

`Client Secret`: The secret of your [Twitch project](dev.twitch.tv)

`User ID`: Your Twitch username

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
