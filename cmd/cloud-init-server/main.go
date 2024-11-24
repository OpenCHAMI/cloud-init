package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/OpenCHAMI/cloud-init/internal/memstore"
	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/jwtauth/v5"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	openchami_authenticator "github.com/openchami/chi-middleware/auth"
	openchami_logger "github.com/openchami/chi-middleware/log"
)

var (
	ciEndpoint        = ":27777"
	tokenEndpoint     = "http://opaal:3333/token" // jwt for smd access obtained from here
	smdEndpoint       = "http://smd:27779"
	jwksUrl           = "" // jwt keyserver URL for secure-route token validation
	insecure          = false
	prometheusEnabled = false
)

// Prometheus Metrics
var (
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)
	httpRequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"path", "method", "status"},
	)
)

func main() {
	flag.StringVar(&ciEndpoint, "listen", ciEndpoint, "Server IP and port for cloud-init-server to listen on")
	flag.StringVar(&tokenEndpoint, "token-url", tokenEndpoint, "OIDC server URL (endpoint) to fetch new tokens from (for SMD access)")
	flag.StringVar(&smdEndpoint, "smd-url", smdEndpoint, "http IP/url and port for running SMD")
	flag.StringVar(&jwksUrl, "jwks-url", jwksUrl, "JWT keyserver URL, required to enable secure route")
	flag.BoolVar(&insecure, "insecure", insecure, "Set to bypass TLS verification for requests")
	flag.BoolVar(&prometheusEnabled, "prometheus", prometheusEnabled, "Enable Prometheus metrics")
	flag.Parse()

	// Set up JWT verification via the specified URL, if any
	var keyset *jwtauth.JWTAuth
	secureRouteEnable := false
	if jwksUrl != "" {
		var err error
		keyset, err = fetchPublicKeyFromURL(jwksUrl)
		if err != nil {
			fmt.Printf("JWKS initialization failed: %s\n", err)
		} else {
			// JWKS init SUCCEEDED, secure route supported
			secureRouteEnable = true
		}
	} else {
		fmt.Println("No JWKS URL provided; secure route will be disabled")
	}

	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Set up prometheus exporter
	if prometheusEnabled {
		// Register Prometheus metrics
		prometheus.MustRegister(httpRequestDuration, httpRequestCount)
		// log the Prometheus exporter start
		log.Info().Msg("Prometheus exporter started :2112")
		// Start the Prometheus
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			log.Fatal().Err(http.ListenAndServe(":2112", nil)).Msg("Prometheus exporter failed")
		}()
	}

	// Primary router and shared SMD client
	router := chi.NewRouter()
	router.Use(
		prometheusMiddleware,
		middleware.RequestID,
		middleware.RealIP,
		middleware.Logger,
		middleware.Recoverer,
		middleware.StripSlashes,
		middleware.Timeout(60*time.Second),
		openchami_logger.OpenCHAMILogger(logger),
	)
	sm := smdclient.NewSMDClient(smdEndpoint, tokenEndpoint, insecure)

	// Unsecured datastore and router
	store := memstore.NewMemStore()
	ciHandler := NewCiHandler(store, sm)
	router_unsec := chi.NewRouter()
	initCiRouter(router_unsec, ciHandler)
	router.Mount("/cloud-init", router_unsec)

	if secureRouteEnable {
		// Secured datastore and router
		store_sec := memstore.NewMemStore()
		ciHandler_sec := NewCiHandler(store_sec, sm)
		router_sec := chi.NewRouter()
		router_sec.Use(
			jwtauth.Verifier(keyset),
			openchami_authenticator.AuthenticatorWithRequiredClaims(keyset, []string{"sub", "iss", "aud"}),
		)
		initCiRouter(router_sec, ciHandler_sec)
		router.Mount("/cloud-init-secure", router_sec)
	}

	// Serve all routes
	log.Fatal().Err(http.ListenAndServe(ciEndpoint, router)).Msg("Server closed")

}

func initCiRouter(router chi.Router, handler *CiHandler) {
	// Add cloud-init endpoints to router
	router.Get("/", handler.ListEntries)
	router.Get("/user-data", handler.GetDataByIP(UserData))
	router.Get("/meta-data", handler.GetDataByIP(MetaData))
	router.Get("/vendor-data", handler.GetDataByIP(VendorData))
	router.Get("/{id}", handler.GetEntry)
	router.Get("/{id}/user-data", handler.GetDataByMAC(UserData))
	router.Put("/{id}/user-data", handler.UpdateUserEntry)
	router.Get("/{id}/meta-data", handler.GetDataByMAC(MetaData))
	router.Get("/{id}/vendor-data", handler.GetDataByMAC(VendorData))
	router.Delete("/{id}", handler.DeleteEntry)

	// groups API endpoints
	router.Get("/groups", handler.GetGroups)
	router.Post("/groups/{id}", handler.AddGroupData)
	router.Get("/groups/{id}", handler.GetGroupData)
	router.Put("/groups/{id}", handler.UpdateGroupData)
	router.Delete("/groups/{id}", handler.RemoveGroupData)
}

func prometheusMiddleware(next http.Handler) http.Handler {
	if !prometheusEnabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timer := prometheus.NewTimer(httpRequestDuration.WithLabelValues(r.URL.Path, r.Method))
		defer timer.ObserveDuration()

		rw := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(rw, r)

		status := fmt.Sprintf("%d", rw.Status())
		httpRequestCount.WithLabelValues(r.URL.Path, r.Method, status).Inc()
	})
}
