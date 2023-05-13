# SoundCloud Downloader

## Overview

This is a command-line tool written in Go, designed to scrape SoundCloud for songs from a specified album or playlist and download them. The tool is highly concurrent, leveraging the power of Go's goroutines and WaitGroups for speedy, efficient downloading. The tool also uses a neat CLI interface courtesy of the progressbar library.

## How It Works

The program is divided into several parts, each handling a distinct aspect of the downloading process:

1. `main.go`: This is the entry point for the program, responsible for parsing command-line arguments, initializing the SoundCloud client, and managing the download process.

2. `pkg/scd/scd.go`: This file handles all interactions with SoundCloud, fetching album and playlist information and downloading tracks.

3. `pkg/scd/types.go`: This file defines several data structures used throughout the program.

4. `pkg/scd/util.go`: This file contains utility functions that are used throughout the program.

## Usage

### Requirements

- Go 1.18 or later.

### Installation

Clone the repository and build the binary:

```bash
git clone https://github.com/yourusername/soundcloud-downloader.git
cd soundcloud-downloader
go build -o scdownloader
```

### Examples

Search for a playlist:

```bash
./scdownloader search -p "playlist name"
```

Search for an album:

```bash
./scdownloader search -a "album name"
```

Search for a track:

```bash
./scdownloader search -t "track name"
```

## License

MIT License. See `LICENSE` for more information.

## Disclaimer

This project is intended for educational purposes only. It is not affiliated with or endorsed by SoundCloud. The developer is not responsible for any misuse or violation of SoundCloud's terms of service. Always respect the rights of content creators and only download content with their explicit permission and/or for personal use.
