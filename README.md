# Streamshower

A simple frontend to show live streams in a TUI that also acts as a daemon to
prevent excessive API calls.

![](demo.gif)

## Usage
Running Streamshower as a server in the background with the `-b` flag will fetch
Twitch and Strims data every refresh interval (`-r`), and then serve this data
at the specified address (`-a`). The address should match the one set as the
callback URI in your [Twitch project](https://dev.twitch.tv)

Running the program normally, at the same address as the server, then fetches
data from the background instance and displays it in an interactive TUI.


## Config
All settings are stored in a config.json file, except for the environment
variable `$BROWSER` which is used to open links in the TUI.

To generate this file, simply run the program in background mode (`-b`)  and
provide `Client ID`, `Client Secret`, and `User ID` when prompted.

Currently the recommended way to generate the configuration file when running
in docker is to run the container interactively for the first run. Example

```console

docker build . -t streamshower
docker run --name streamshower -p 8181:8181 -it streamshower:latest
```

Explanation:

`Client ID`: The api key of your Twitch project

`Client Secret`: The secret of your Twitch project

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
