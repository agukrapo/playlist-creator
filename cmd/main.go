package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agukrapo/go-http-client/client"
	"github.com/agukrapo/spotify-playlist-creator/spotify"
	"github.com/joho/godotenv"
)

const (
	errTokenNotFound   = cmdError("Environment variable SPOTIFY_TOKEN not found")
	errFilenameMissing = cmdError("Filename argument missing")
	errNoTracksLeft    = cmdError("No tracks left")
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	token, err := tokenEnv()
	if err != nil {
		return err
	}

	spotCli := spotify.New(token, client.New())

	lines, name, err := openFile()
	if err != nil {
		return err
	}

	tracks, err := getTracks(ctx, spotCli, lines)
	if err != nil {
		return err
	}

	if len(tracks) == 0 {
		return errNoTracksLeft
	}

	fmt.Printf("Creating playlist %q with %d tracks\nPress the Enter Key to continue\n", name, len(tracks))

	if _, err := fmt.Scanln(); err != nil {
		return err
	}

	userID, err := spotCli.Me(ctx)
	if err != nil {
		return err
	}

	playlistID, playlistURL, err := spotCli.CreatePlaylist(ctx, userID, name)
	if err != nil {
		return err
	}

	if err := spotCli.AddTracksToPlaylist(ctx, playlistID, tracks); err != nil {
		return err
	}

	fmt.Printf("Playlist created: %s\n", playlistURL)

	return nil
}

func openFile() ([]string, string, error) {
	if len(os.Args) < 2 {
		return nil, "", errFilenameMissing
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	var lines []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		lines = append(lines, line)
	}

	if err = scanner.Err(); err != nil {
		return nil, "", err
	}

	name := strings.TrimSuffix(filepath.Base(os.Args[1]), filepath.Ext(os.Args[1]))

	return lines, name, nil
}

func tokenEnv() (string, error) {
	_ = godotenv.Load()

	out, ok := os.LookupEnv("SPOTIFY_TOKEN")
	if !ok {
		return "", errTokenNotFound
	}

	return out, nil
}

func getTracks(ctx context.Context, spotCli *spotify.Client, lines []string) ([]string, error) {
	out := make([]string, 0, len(lines))

	for _, l := range lines {
		trackURI, found, err := spotCli.SearchTrack(ctx, l)
		if err != nil {
			return nil, err
		}

		if !found {
			_, _ = fmt.Fprintf(os.Stderr, "Track %q not found\n", l)

			continue
		}

		out = append(out, trackURI)
	}

	return out, nil
}

type cmdError string

func (ce cmdError) Error() string {
	return string(ce)
}
