package postgresql

import (
	"net/http"

	"github.com/drycc/drycc/discoverd/client"
	"github.com/drycc/drycc/pkg/httphelper"
	"github.com/drycc/drycc/pkg/sirenia/client"
	"github.com/drycc/drycc/pkg/sirenia/state"
	"github.com/drycc/drycc/pkg/status"
	"github.com/julienschmidt/httprouter"
	"github.com/inconshreveable/log15"
)

// Handler represents an HTTP API handler for the process.
type Handler struct {
	router *httprouter.Router

	Process     *Process
	Peer        *state.Peer
	Heartbeater discoverd.Heartbeater
	Logger      log15.Logger
}

// NewHandler returns a new instance of Handler.
func NewHandler() *Handler {
	h := &Handler{
		router: httprouter.New(),
		Logger: log15.New(),
	}
	h.router.Handler("GET", status.Path, status.Handler(h.healthStatus))
	h.router.GET("/status", h.handleGetStatus)
	h.router.POST("/stop", h.handlePostStop)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) { h.router.ServeHTTP(w, req) }

func (h *Handler) healthStatus() status.Status {
	info := h.Peer.Info()
	if info.State == nil || info.RetryPending != nil ||
		(info.Role != state.RolePrimary && info.Role != state.RoleSync && info.Role != state.RoleAsync) {
		return status.Unhealthy
	}

	process, err := h.Process.Info()
	if err != nil || !process.Running || !process.UserExists {
		return status.Unhealthy
	}
	if info.Role == state.RolePrimary {
		if !process.ReadWrite {
			return status.Unhealthy
		}
		if !info.State.Singleton && (process.SyncedDownstream == nil || info.State.Sync == nil || info.State.Sync.ID != process.SyncedDownstream.ID) {
			return status.Unhealthy
		}
	}

	return status.Healthy
}

func (h *Handler) handleGetStatus(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	logger := h.Logger.New("fn", "handleGetStatus")

	status := &client.Status{
		Peer: h.Peer.Info(),
	}
	var err error
	status.Database, err = h.Process.Info()
	if err != nil {
		// Log the error, but don't return a 500. We will always have some
		// information to return, but postgres may not be online.
		logger.Error("error getting postgres info", "err", err)
	}
	httphelper.JSON(w, 200, status)
}

func (h *Handler) handlePostStop(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	if err := h.Peer.Stop(); err != nil {
		httphelper.Error(w, err)
		return
	}
	if err := h.Heartbeater.Close(); err != nil {
		httphelper.Error(w, err)
		return
	}
	w.WriteHeader(200)
}
