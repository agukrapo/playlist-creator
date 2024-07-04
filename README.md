# playlist-creator

CLI tool to create a playlist on the provided target using a text file containing a song name per line

Playlist name will be the file name without extension

## Usage

```
playlist-creator <target> <file>
```

Example
```
playlist-creator spotify friday-party.txt
```

## Install
```
go install github.com/agukrapo/playlist-creator/cmd@latest
```

Or download binary from the [latest release](https://github.com/agukrapo/playlist-creator/releases/latest)

## Available targets

### Spotify
Needs a valid Spotify OAuth token in the **SPOTIFY_TOKEN** environment variable (.env file supported)

Generate a OAuth token for the currently logged user in here https://developer.spotify.com/console/get-search-item/

Make sure the token has the **playlist-modify-private** scope

### Deezer
Uses a valid Deezer ARL cookie in the **DEEZER_ARL_COOKIE** environment variable (.env file supported)

Check [here](https://github.com/d-fi/d-fi-core/blob/master/docs/faq.md) how to get this cookie.
