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
	if config.AntidosMaxConcurrent == 0 {
		panic("server: antidosMaxConcurrent is required")
	}
	if config.DataDir == "" {
		panic("server: dataDir is required")
	}
	if config.ShutdownTimeout == 0 {
		panic("server: shutdownTimeout is required")
	}

	fsys := os.DirFS(filepath.FromSlash(config.DataDir))

	antidos := newAntidos(
		config.AntidosBuckets, config.AntidosPeriod, config.AntidosMaxConcurrent,
		tooManyRequestsHandler(dataFile(fsys, "static/429.html")),
	)

	// TODO team middleware

	mux := http.NewServeMux()

	registerHandler := func(method, path, src string, handler http.Handler) {
		slog.Info("registering handler", "src", src, "method", method, "path", path)
		mux.Handle(method+" "+path, handler)
	}

	registerHandler("GET", "/", "static/404.html", antidos.middleware(notFoundHandler(dataFile(fsys, "static/404.html"))))
	// registerHandler("GET", "/favicon.ico", "static/favicon.ico", cachedHandler(dataFile(fsys, "static/favicon.ico"))) // TODO favicon

	const puzzlesDir = "puzzles"
	puzzles, err := fs.ReadDir(fsys, puzzlesDir)
	if err != nil {
		panic(fmt.Errorf("server: list puzzles directory: %w", err))
	}

	validPuzzles := make([]string, 0, len(puzzles))

	for _, puzzle := range puzzles {
		if !puzzle.IsDir() {
			slog.Warn("puzzle not a directory, skipping", "name", puzzle.Name())
			continue
		}

		dir := "/"
		if n := puzzle.Name(); n != "index" {
			dir += n + "/"
			validPuzzles = append(validPuzzles, puzzle.Name())
		}

		type file struct {
			src         string
			path        string
			content     []byte
			contentType string
		}
		files := []file{}

		extMap := map[string]string{}
		prefix := path.Join(puzzlesDir, puzzle.Name())
		err := fs.WalkDir(fsys, prefix, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			content, ct := dataFile(fsys, p)

			name := d.Name()
			if name == "index.html" {
				files = append(files, file{
					src:         p,
					path:        dir + "{$}",
					content:     content,
					contentType: ct,
				})
				return nil
			}

			subPath, ok := strings.CutPrefix(p, prefix+"/")
			if !ok {
				return fmt.Errorf("%q is not a subpath of %q", p, prefix+"/")
			}

			h := sha256.New()
			h.Write([]byte(p))
			ep := base64.RawURLEncoding.EncodeToString(h.Sum(nil)) + path.Ext(name)

			extMap[subPath] = dir + ep

			files = append(files, file{
				src:         p,
				path:        dir + ep,
				content:     content,
				contentType: ct,
			})

			return nil
		})
		if err != nil {
			panic(fmt.Errorf("server: walk puzzles directory: %w", err))
		}

		for _, file := range files {
			if shouldRemap(file.src) {
				for orig, new := range extMap {
					file.content = bytes.ReplaceAll(file.content, []byte(orig), []byte(new))
				}
			}

			handler := cachedHandler(file.content, file.contentType)
			if shouldLimit(file.src) {
				handler = antidos.middleware(handler)
			}

			registerHandler("GET", file.path, file.src, handler)
		}
	}

	handler := http.Handler(mux)
	handler = puzzlePathMiddleware(validPuzzles, handler)
	handler = logMiddleware(handler)

	return &Server{
		addr:            fmt.Sprintf("0.0.0.0:%d", config.Port),
		handler:         handler,
		shutdownTimeout: config.ShutdownTimeout,
	}
}

func shouldRemap(file string) bool {
	ext := path.Ext(file)
	return ext == ".html" || ext == ".css" || ext == ".js"
}

func shouldLimit(file string) bool {
	return strings.HasSuffix(file, "index.html")
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
	shutdownErr := srv.Shutdown(stopCtx)
	stopCancel()

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
