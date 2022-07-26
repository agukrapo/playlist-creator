package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agukrapo/spotify-playlist-creator/spotify"
	"github.com/joho/godotenv"
)

func main() {
	if err := exec(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func exec() error {
	ctx := context.Background()

	token, err := tokenEnv()
	if err != nil {
		return err
	}

	c := spotify.New(token)

	lines, name, err := openFile()
	if err != nil {
		return err
	}

	tracks := make([]string, 0, len(lines))
	for _, l := range lines {
		trackURI, found, err := c.SearchTrack(ctx, l)
		if err != nil {
			return err
		}

		if !found {
			_, _ = fmt.Fprintf(os.Stderr, "Track %q not found\n", l)

			continue
		}

		tracks = append(tracks, trackURI)
	}

	if len(tracks) == 0 {
		return errors.New("No tracks left")
	}

	fmt.Printf("Creating playlist %q with %d tracks\n", name, len(tracks))
	fmt.Println("Press the Enter Key to continue")
	if _, err = fmt.Scanln(); err != nil {
		return err
	}

	userID, err := c.Me(ctx)
	if err != nil {
		return err
	}

	playlistID, playlistURL, err := c.CreatePlaylist(ctx, userID, name)
	if err != nil {
		return err
	}

	if err = c.AddTracksToPlaylist(ctx, playlistID, tracks); err != nil {
		return err
	}

	fmt.Printf("Playlist created: %s\n", playlistURL)

	return nil
}

func openFile() ([]string, string, error) {
	if len(os.Args) < 2 {
		return nil, "", errors.New("Filename argument missing")
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	var lines []string

	sc := bufio.NewScanner(file)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}

	if err = sc.Err(); err != nil {
		return nil, "", err
	}

	name := strings.TrimSuffix(filepath.Base(os.Args[1]), filepath.Ext(os.Args[1]))
	return lines, name, nil
}

func tokenEnv() (string, error) {
	_ = godotenv.Load()

	out, ok := os.LookupEnv("SPOTIFY_TOKEN")
	if !ok {
		return "", errors.New("Environment variable SPOTIFY_TOKEN not found")
	}

	return out, nil
}
