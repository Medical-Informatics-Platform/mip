package spawnapi

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	requestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mip_spawn_api_requests_total",
		Help: "Spawn API requests by method and response code.",
	}, []string{"method", "code"})
	deniedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mip_spawn_api_denied_total",
		Help: "Spawn API requests denied, by reason.",
	}, []string{"reason"})
)

func init() {
	metrics.Registry.MustRegister(requestsTotal, deniedTotal)
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.code = code
	s.ResponseWriter.WriteHeader(code)
}

// instrument counts requests; labels carry no names, so cardinality is fixed.
func instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rec, r)
		method := r.Method
		switch method {
		case http.MethodGet, http.MethodPut, http.MethodDelete:
		default:
			method = "other"
		}
		requestsTotal.WithLabelValues(method, strconv.Itoa(rec.code)).Inc()
	})
}

// Listener serves a handler over TLS as a manager Runnable.
type Listener struct {
	Addr    string
	Handler http.Handler
	// GetCertificate comes from a certwatcher, so a renewed certificate is
	// picked up without a restart.
	GetCertificate func(*tls.ClientHelloInfo) (*tls.Certificate, error)

	ready atomic.Bool
}

func (l *Listener) Start(ctx context.Context) error {
	srv := &http.Server{
		Handler: l.Handler,
		TLSConfig: &tls.Config{
			MinVersion:     tls.VersionTLS12,
			GetCertificate: l.GetCertificate,
		},
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    32 << 10,
	}
	ln, err := net.Listen("tcp", l.Addr)
	if err != nil {
		return err
	}
	l.ready.Store(true)
	done := make(chan struct{})
	go func() {
		defer close(done)
		<-ctx.Done()
		l.ready.Store(false)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	err = srv.ServeTLS(ln, "", "")
	if errors.Is(err, http.ErrServerClosed) {
		<-done
		return nil
	}
	return err
}

// NeedLeaderElection is false: every replica serves.
func (l *Listener) NeedLeaderElection() bool { return false }

// ReadyCheck is a readyz check: ready while the listener is open.
func (l *Listener) ReadyCheck(_ *http.Request) error {
	if !l.ready.Load() {
		return errors.New("spawn API listener is not serving")
	}
	return nil
}
