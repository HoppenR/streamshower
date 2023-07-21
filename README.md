# Streamshower

All settings are stored in a config.json file

To generate this file, simply run the server as usual and provide  Client ID,
Client Secret, and User ID when prompter.

Explanation:

`Client ID`: The api key of your dev.twitch.tv project

`Client Secret`: The secret of your dev.twitch.tv project

`User ID`: Your twitch username

# Usage
You can run streamshower as a server in the background and it will check for
streams and `notify-send` whenever one comes online.

The interactive UI also assumes there is a server running in the background and
requests data from it.

# Navigation
standard vim navigation: `jkl` or arrow keys + enter

`f` to open a filter dialog

`F` to clear filter

Twitch filters: pressing `!` inverts the showing matches

Strims filters: numbers work as a minimum-rustler-threshold

filter window supports regular readline keys such as ctrl-u to clear, ctrl-a to
go to beginning of line, ctrl-e to go to end of line etc

`i` to fullscreen the Stream Info

`o` to switch between lists

`r` to force the server to refresh data

`q` to quit
# streamshower
