package server

import (
	"net/http"
	"sync"
)

type reloadingHandler struct {
	mx         *sync.RWMutex
	handler    http.Handler
	reloadFunc func(*reloadingHandler) (http.Handler, error)
}

func newReloadingHandler(reload func(*reloadingHandler) (http.Handler, error)) *reloadingHandler {
	r := &reloadingHandler{
		mx:         &sync.RWMutex{},
		reloadFunc: reload,
	}

	h, err := reload(r)
	if err != nil {
		panic(err)
	}

	r.handler = h
	return r
}

func (r *reloadingHandler) load() http.Handler {
	r.mx.RLock()
	defer r.mx.RUnlock()
	return r.handler
}

func (r *reloadingHandler) store(h http.Handler) {
	r.mx.Lock()
	defer r.mx.Unlock()
	r.handler = h
}

func (r *reloadingHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.load().ServeHTTP(w, req)
}

func (r *reloadingHandler) reload() error {
	h, err := r.reloadFunc(r)
	if err != nil {
		return err
	}

	r.store(h)
	return nil
}
