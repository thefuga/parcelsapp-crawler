# Parcels App Crawler
This is a silly automation to track parcels from Parcels App from the terminal, 
allowing stuff like formatting with JQ, TMUX and Polybar visualization, and so on.

## How to use it?
- Copy the `config.example.json` file to `config.json` and fill it with the proper values.
	- `chrome_path` is optional and must only be used if the application doesn't locate it.
- Build the app (`go build main.go`) and run the binary, or simply run it with `go run main.go`.
- Wait for the application to crawl the website and fetch the tracking info.

### Requirements
- Go 1.19 (>1.15 should do it)
