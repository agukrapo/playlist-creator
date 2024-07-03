package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/agukrapo/go-http-client/client"
	"github.com/agukrapo/spotify-playlist-creator/deezer"
	"github.com/agukrapo/spotify-playlist-creator/playlists"
	"github.com/agukrapo/spotify-playlist-creator/spotify"
	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		<-ctx.Done()
		cancel()
	}()

	manager, err := buildManager()
	if err != nil {
		return err
	}

	lines, name, err := openFile()
	if err != nil {
		return err
	}

	data, err := manager.Gather(ctx, name, lines)
	if err != nil {
		return err
	}

	fmt.Printf("Creating playlist %q with %d tracks\n\n", name, data.Length())
	fmt.Println("Press the Enter Key to continue")

	if _, err := fmt.Scanln(); err != nil {
		return err
	}

	if err := manager.Push(ctx, data); err != nil {
		return err
	}

	fmt.Println("Playlist created")

	return nil
}

func buildManager() (*playlists.Manager, error) {
	if len(os.Args) < 2 {
		return nil, errors.New("target argument missing")
	}

	var target playlists.Target
	switch os.Args[1] {
	case "spotify":
		token, err := env("SPOTIFY_TOKEN")
		if err != nil {
			return nil, err
		}
		target = spotify.New(client.New(), token)
	case "deezer":
		cookie, err := env("DEEZER_ARL_COOKIE")
		if err != nil {
			return nil, err
		}
		target = deezer.New(client.New(), cookie)
	default:
		return nil, fmt.Errorf("unknown target %s", os.Args[1])
	}

	return playlists.NewManager(target), nil
}

func openFile() ([]string, string, error) {
	if len(os.Args) < 3 {
		return nil, "", errors.New("filename argument missing")
	}

	path := os.Args[2]

	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	var lines []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			lines = append(lines, line)
		}
	}

	if err = scanner.Err(); err != nil {
		return nil, "", err
	}

	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	return lines, name, nil
}

func env(name string) (string, error) {
	_ = godotenv.Load()

	out, ok := os.LookupEnv(name)
	if !ok {
		return "", fmt.Errorf("environment variable %s not found", name)
	}

	return out, nil
}
