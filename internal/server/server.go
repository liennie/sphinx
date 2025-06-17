// Package server provides HTTP server setup and lifecycle management for the Sphinx application.
package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sphinx/internal/ctxlog"
	"strings"
	"time"
)

type Server struct {
	addr            string
	handler         http.Handler
	shutdownTimeout time.Duration
}

func New(config Config) *Server {
	if config.Port == 0 {
		panic("server: port is required")
	}
	if config.AntidosBuckets == 0 {
		panic("server: antidosBuckets is required")
	}
	if config.AntidosPeriod == 0 {
		panic("server: antidosPeriod is required")
	}
	if config.DataDir == "" {
		panic("server: dataDir is required")
	}
	if config.ShutdownTimeout == 0 {
		panic("server: shutdownTimeout is required")
	}

	anti := newAntidos(config.AntidosBuckets, config.AntidosPeriod)

	mux := http.NewServeMux()

	fsys := os.DirFS(filepath.FromSlash(config.DataDir))

	slog.Info("registering handler", "path", "/", "src", "static/404.html")
	mux.Handle("GET /", anti.middleware(notFoundHandler(dataFile(fsys, "static/404.html"))))

	// slog.Info("registering handler", "path", "/favicon.ico", "src", "static/favicon.ico")
	// mux.Handle("GET /favicon.ico", cachedHandler(dataFile(fsys, "static/favicon.ico")))

	const puzzlesDir = "puzzles"
	puzzles, err := fs.ReadDir(fsys, puzzlesDir)
	if err != nil {
		panic(fmt.Errorf("server: list puzzles directory: %w", err))
	}

	for _, puzzle := range puzzles {
		if !puzzle.IsDir() {
			slog.Warn("puzzle not a directory, skipping", "name", puzzle.Name())
			continue
		}

		dir := "/"
		if n := puzzle.Name(); n != "index" {
			dir += n + "/"
		}

		extMap := map[string]string{}
		prefix := path.Join(puzzlesDir, puzzle.Name())
		err := fs.WalkDir(fsys, prefix, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			name := d.Name()
			if name == "index.html" {
				return nil
			}

			subPath, ok := strings.CutPrefix(p, prefix+"/")
			if !ok {
				return fmt.Errorf("%q is not a subpath of %q", p, prefix+"/")
			}

			ext := path.Ext(name)

			h := sha256.New()
			h.Write([]byte(p))
			ep := base64.RawURLEncoding.EncodeToString(h.Sum(nil)) + ext

			slog.Info("registering handler", "path", dir+ep, "src", p)
			mux.Handle("GET "+dir+ep, anti.middleware(cachedHandler(dataFile(fsys, p))))

			extMap[subPath] = dir + ep

			return nil
		})
		if err != nil {
			panic(fmt.Errorf("server: walk puzzles directory: %w", err))
		}

		index := path.Join(prefix, "index.html")
		content, ct := dataFile(fsys, index)

		for orig, new := range extMap {
			content = bytes.ReplaceAll(content, []byte(orig), []byte(new))
		}

		slog.Info("registering handler", "path", dir+"{$}", "src", index)
		mux.Handle("GET "+dir+"{$}", anti.middleware(cachedHandler(content, ct)))
	}

	handler := http.Handler(mux)
	handler = logMiddleware(handler)

	return &Server{
		addr:            fmt.Sprintf("0.0.0.0:%d", config.Port),
		handler:         handler,
		shutdownTimeout: config.ShutdownTimeout,
	}
}

func (s *Server) Run(ctx context.Context) error {
	logger := ctxlog.Get(ctx)

	srv := &http.Server{
		Addr:        s.addr,
		Handler:     s.handler,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	serveErrCh := make(chan error, 1)
	go func() {
		defer cancel()
		logger.Info("server is running", "addr", s.addr)
		serveErrCh <- srv.ListenAndServe()
	}()

	<-ctx.Done()

	logger.Info("server is shutting down")

	stopCtx, stopCancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer stopCancel()
	shutdownErr := srv.Shutdown(stopCtx)

	<-stopCtx.Done()
	if errors.Is(stopCtx.Err(), context.DeadlineExceeded) {
		logger.Error("server shutdown timeout exceeded")

	} else if errors.Is(stopCtx.Err(), context.Canceled) {
		logger.Info("all clients closed successfully")
	}

	serveErr := <-serveErrCh
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}

	return errors.Join(serveErr, shutdownErr)
}
