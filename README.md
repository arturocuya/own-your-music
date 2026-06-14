# Own Your Music

> A local web app that scans your Spotify Liked Songs and finds the same tracks on Bandcamp, so you can buy and actually own the music you love.

## Tech stack

`Go` · `HTMX` · `Templ`

## Overview

Streaming gives you access, not ownership. Own Your Music helps you close that gap: it reads the songs you have liked on Spotify and looks each one up on Bandcamp, where buying a track means it is yours to keep and download. Everything runs locally on your machine, so your library data stays with you.

> Note: Carefully read Bandcamp's Acceptable Use and Moderation policy before using this program. Just saying :p

## Features

- Scans your Spotify Liked Songs as the source library.
- Searches Bandcamp for matching tracks so you can buy and own them.
- Includes an Amazon Music provider as an additional source.
- Runs entirely as a local web app, no hosted service required.

## How it works

You run the app locally and it starts a small web server on your machine. It reads your liked tracks from the source provider (Spotify, with Amazon Music also supported), then queries Bandcamp for each one and renders the matches in the browser via HTMX and Templ. From there you can follow through to Bandcamp to purchase the tracks you want to own. Because it runs on your own machine, the matching and your library data never leave it.

> _Demo: add a screenshot or short GIF of the scan results in the browser here._

## Install and usage

You will need [Go](https://go.dev/dl/) installed.

```bash
git clone https://github.com/arturocuya/own-your-music.git
cd own-your-music
go run *.go
```

Then open the local web app in your browser

## Roadmap

- [x] Implement Amazon Music provider
- [ ] Add a proxy for Amazon (to avoid being flagged as a crawler bot)
- [ ] Windows binary
- [ ] macOS binary
