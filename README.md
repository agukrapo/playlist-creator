# spotify-playlist-creator

Creates a Spotify playlist from a text file containing a song name per line

The playlist name will be the file name without extension

## usage

```
spotify-playlist-creator friday-party.txt
```

Needs a valid Spotify OAuth token in the **SPOTIFY_TOKEN** environment variable (.env file supported)

Generate a OAuth token for the currently logged user in here https://developer.spotify.com/console/get-search-item/

Make sure the token has the **playlist-modify-private** scope