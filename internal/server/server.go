// Package server provides HTTP server setup and lifecycle management for the Sphinx application.
package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
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
	"sphinx/internal/rec"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

type Server struct {
	addr            string
	handler         *reloadingHandler
	tlsLoader       *tlsLoader
	httpsRedirect   bool
	shutdownTimeout time.Duration
	deadline        time.Time
}

func New(config Config) *Server {
	if config.Port == 0 {
		panic("server: port is required")
	}
	if config.AntiCheatBuckets == 0 {
		panic("server: antiCheatBuckets is required")
	}
	if config.AntiCheatPeriod == 0 {
		panic("server: antiCheatPeriod is required")
	}
	if config.AntiCheatMaxConcurrent == 0 {
		panic("server: antiCheatMaxConcurrent is required")
	}
	if config.DataDir == "" {
		panic("server: dataDir is required")
	}
	if config.ShutdownTimeout == 0 {
		panic("server: shutdownTimeout is required")
	}
	if config.AdminKey == "" {
		panic("server: adminKey is required")
	}

	handler := newReloadingHandler(func(r *reloadingHandler) (http.Handler, error) {
		return newHandler(config, r)
	})

	var loader *tlsLoader
	if config.TLSCertFile != "" && config.TLSKeyFile != "" {
		if config.TLSReloadInterval <= 0 {
			config.TLSReloadInterval = 24 * time.Hour
		}

		loader = newTLSLoader(config.TLSCertFile, config.TLSKeyFile, config.TLSReloadInterval)
	} else if config.TLSCertFile != "" || config.TLSKeyFile != "" {
		panic("server: both tlsCertFile and tlsKeyFile must be set to enable TLS")
	}

	return &Server{
		addr:            fmt.Sprintf("%s:%d", config.Host, config.Port),
		handler:         handler,
		tlsLoader:       loader,
		httpsRedirect:   config.HTTPSRedirect,
		shutdownTimeout: config.ShutdownTimeout,
		deadline:        config.Deadline,
	}
}

func newHandler(config Config, rh *reloadingHandler) (h http.Handler, err error) {
	defer rec.Error(&err)

	fsys := os.DirFS(filepath.FromSlash(config.DataDir))

	// middlewares
	antiCheat := newAntiCheat(
		config.AntiCheatBuckets, config.AntiCheatPeriod, config.AntiCheatMaxConcurrent,
		tooManyRequestsHandler(dataFile(fsys, "static/429.html")),
	)

	teams := newTeams(dataFile(fsys, "static/team.html"))

	notFound := antiCheat.middleware(notFoundHandler(dataFile(fsys, "static/404.html")))

	admin := newAdmin(config.AdminKey, notFound)

	// handlers
	mux := http.NewServeMux()

	registerHandler := func(method, path, src string, handler http.Handler) {
		slog.Info("registering handler", "method", method, "path", path, "src", src)
		mux.Handle(method+" "+path, handler)
	}

	// static
	registerHandler("GET", "/", "static/404.html", notFound)

	// TODO favicon
	const wwwDir = "static/www"
	err = fs.WalkDir(fsys, wwwDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		subPath, ok := strings.CutPrefix(p, wwwDir)
		if !ok {
			return fmt.Errorf("%q is not a subpath of %q", p, wwwDir)
		}

		registerHandler("GET", subPath, p, cachedHandler(dataFile(fsys, p)))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("server: walk www directory: %w", err)
	}

	registerHandler("GET", "/admin/{$}", "static/admin.html", admin.middleware(cachedHandler(dataFile(fsys, "static/admin.html"))))
	registerHandler("POST", "/admin/reload", "reloadHandler()", admin.middleware(reloadHandler(rh)))
	registerHandler("GET", "/admin/teams/progress", "progressHandler()", admin.middleware(progressHandler(config.PuzzleOrder)))
	registerHandler("POST", "/admin/teams/hide", "hideTeamHandler()", admin.middleware(http.HandlerFunc(hideTeamHandler)))
	registerHandler("POST", "/admin/teams/rename", "renameTeamHandler()", admin.middleware(http.HandlerFunc(renameTeamHandler)))

	// puzzles
	const puzzlesDir = "puzzles"
	puzzles, err := fs.ReadDir(fsys, puzzlesDir)
	if err != nil {
		return nil, fmt.Errorf("server: list puzzles directory: %w", err)
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

			name := d.Name()
			if strings.HasPrefix(name, "!") {
				return nil
			}

			content, ct := dataFile(fsys, p)

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
			h.Write([]byte("VGhpc0lzQVN1cGVyU2VjcmV0U2FsdA"))
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
			return nil, fmt.Errorf("server: walk puzzles directory: %w", err)
		}

		for _, file := range files {
			if shouldRemap(file.src) {
				for orig, new := range extMap {
					file.content = bytes.ReplaceAll(file.content, []byte(orig), []byte(new))
				}
			}

			handler := cachedHandler(file.content, file.contentType)
			if shouldTeam(file.src, puzzle.Name()) {
				handler = teams.middleware(handler)
			}
			if shouldLimit(file.src) {
				handler = antiCheat.middleware(handler)
			}

			registerHandler("GET", file.path, file.src, handler)
			if shouldTeam(file.src, puzzle.Name()) {
				registerHandler("POST", file.path, file.src, handler)
			}
		}
	}

	handler := http.Handler(mux)
	handler = puzzlePathMiddleware(validPuzzles, handler)
	handler = newRecover(handler, internalServerErrorHandler(dataFile(fsys, "static/500.html")))
	handler = robotsMiddleware(handler)
	handler = hostMiddleware(config.Host, handler)
	handler = logMiddleware(handler)

	return handler, nil
}

func shouldRemap(file string) bool {
	ext := path.Ext(file)
	return ext == ".html" || ext == ".css" || ext == ".js"
}

func shouldLimit(file string) bool {
	return strings.HasSuffix(file, "index.html")
}

func shouldTeam(file, puzzle string) bool {
	return strings.HasSuffix(file, "index.html") && puzzle != "index"
}

func (s *Server) runServer(ctx context.Context, cancel context.CancelFunc, srv *http.Server) error {
	logger := ctxlog.Get(ctx)

	serveErrCh := make(chan error, 1)
	go func() {
		defer cancel()
		logger.Info("server is running", "addr", srv.Addr)
		if srv.TLSConfig != nil {
			serveErrCh <- srv.ListenAndServeTLS("", "")
		} else {
			serveErrCh <- srv.ListenAndServe()
		}
	}()

	<-ctx.Done()

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

func (s *Server) Run(ctx context.Context) error {
	logger := ctxlog.Get(ctx)

	// setup
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if !s.deadline.IsZero() {
		now := time.Now()
		logger.Info("deadline set", "deadline", s.deadline, "duration", s.deadline.Sub(now).String())

		var dcancel context.CancelFunc
		ctx, dcancel = context.WithDeadline(ctx, s.deadline)
		defer dcancel()
	}

	group, ctx := errgroup.WithContext(ctx)

	// main server
	srv := &http.Server{
		Addr:        s.addr,
		Handler:     s.handler,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	if s.tlsLoader != nil {
		srv.TLSConfig = &tls.Config{
			GetCertificate: s.tlsLoader.getCertificate,
		}
		go s.tlsLoader.reloadLoop(ctx)
	}
	group.Go(func() error {
		return s.runServer(ctx, cancel, srv)
	})

	// redirect server
	if s.httpsRedirect {
		redirectToHTTPS := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target := "https://" + r.Host + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		})
		httpsRedirectSrv := &http.Server{
			Addr:        ":80",
			Handler:     redirectToHTTPS,
			BaseContext: func(net.Listener) context.Context { return ctx },
		}
		group.Go(func() error {
			return s.runServer(ctx, cancel, httpsRedirectSrv)
		})
	}

	<-ctx.Done()

	logger.Info("server is shutting down")

	return group.Wait()
}
