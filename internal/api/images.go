package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/m18h/kanea/internal/imagegc"
)

// PathImages is the node's image view; PathImagesGC runs one sweep now
// (PRD §5.2.4/§16.1, v1.111).
const (
	PathImages   = "/v1/images"
	PathImagesGC = "/v1/images/gc"
)

// ImageCollector is the slice of the image GC the API needs (interfaces at
// the consumer, the HostInspector shape): the read half works on any node,
// the sweep half answers 409 wherever the collector refuses - a disabled
// collector, or a node whose default pull policy is "never". Nil answers
// 503, like every optional daemon dependency.
type ImageCollector interface {
	List(ctx context.Context) ([]imagegc.Image, error)
	Once(ctx context.Context) (imagegc.Summary, error)
	Refusal() string
}

// ImagesResponse is GET /v1/images.
type ImagesResponse struct {
	Images []imagegc.Image `json:"images"`
}

// handleListImages reports every containerd image the node holds, with the
// collector's in-use verdict.
func (s *Server) handleListImages(w http.ResponseWriter, r *http.Request) {
	if s.imageGC == nil {
		writeError(w, http.StatusServiceUnavailable, errNoImageCollector)
		return
	}
	images, err := s.imageGC.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, ImagesResponse{Images: images})
}

// handleImagesGC runs one sweep now. A refused collector answers 409 naming
// the reason rather than sweeping past the node's own configuration.
func (s *Server) handleImagesGC(w http.ResponseWriter, r *http.Request) {
	if s.imageGC == nil {
		writeError(w, http.StatusServiceUnavailable, errNoImageCollector)
		return
	}
	if reason := s.imageGC.Refusal(); reason != "" {
		writeError(w, http.StatusConflict, errors.New(reason))
		return
	}
	summary, err := s.imageGC.Once(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

var errNoImageCollector = errors.New("api: this daemon has no image collector wired")
