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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"

	openchami_authenticator "github.com/openchami/chi-middleware/auth"
	openchami_logger "github.com/openchami/chi-middleware/log"
)

var (
	ciEndpoint    = ":27777"
	tokenEndpoint = "http://opaal:3333/token" // jwt for smd access obtained from here
	smdEndpoint   = "http://smd:27779"
	jwksUrl       = "" // jwt keyserver URL for secure-route token validation
	insecure      = false
	accessToken   = ""
	certPath      = ""
	store         ciStore
	clusterName   string
)

func main() {
	flag.StringVar(&ciEndpoint, "listen", ciEndpoint, "Server IP and port for cloud-init-server to listen on")
	flag.StringVar(&tokenEndpoint, "token-url", tokenEndpoint, "OIDC server URL (endpoint) to fetch new tokens from (for SMD access)")
	flag.StringVar(&smdEndpoint, "smd-url", smdEndpoint, "Server host and port only for running SMD (do not include /hsm/v2)")
	flag.StringVar(&jwksUrl, "jwks-url", jwksUrl, "JWT keyserver URL, required to enable secure route")
	flag.StringVar(&accessToken, "access-token", accessToken, "encoded JWT access token")
	flag.StringVar(&clusterName, "cluster-name", clusterName, "Name of the cluster")
	flag.StringVar(&certPath, "cacert", certPath, "Path to CA cert. (defaults to system CAs)")
	flag.BoolVar(&insecure, "insecure", insecure, "Set to bypass TLS verification for requests")
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
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Primary router and shared SMD client
	router := chi.NewRouter()
	router.Use(
		middleware.RequestID,
		middleware.RealIP,
		middleware.Logger,
		middleware.Recoverer,
		middleware.StripSlashes,
		middleware.Timeout(60*time.Second),
		openchami_logger.OpenCHAMILogger(log.Logger),
	)

	var sm smdclient.SMDClientInterface
	// if the CLOUD-INIT_SMD_SIMULATOR environment variable is set, use the simulator
	if os.Getenv("CLOUD_INIT_SMD_SIMULATOR") == "true" {
		fmt.Printf("\n\n**********\n\n\tCLOUD_INIT_SMD_SIMULATOR is set to true in your environment.\n\n\tUsing the FakeSMDClient to simulate SMD\n\n**********\n\n\n")
		fakeSm := smdclient.NewFakeSMDClient(clusterName, 500)
		fakeSm.Summary()
		sm = fakeSm
	} else {
		var err error
		sm, err = smdclient.NewSMDClient(clusterName, smdEndpoint, tokenEndpoint, accessToken, certPath, insecure)
		if err != nil {
			// Could not create SMD client, so exit with error saying why
			log.Fatal().Err(err)
		}
	}

	// Unsecured datastore and router
	store = memstore.NewMemStore()
	ciHandler := NewCiHandler(store, sm, clusterName)
	router_unsec := chi.NewRouter()
	initCiRouter(router_unsec, ciHandler)
	router.Mount("/cloud-init", router_unsec)

	if secureRouteEnable {
		// Secured datastore and router
		store_sec := memstore.NewMemStore()
		ciHandler_sec := NewCiHandler(store_sec, sm, clusterName)
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
	router.Get("/user-data", UserDataHandler)
	router.Get("/meta-data", MetaDataHandler(handler.sm, handler.store, clusterName))
	router.Get("/vendor-data", VendorDataHandler)
	router.Get("/{group}.yaml", GroupUserDataHandler(handler.sm, handler.store))
	router.Get("/instance-data", InstanceDataHandler(handler.sm, handler.store, clusterName))

	// admin API subrouter
	router.Route("/admin", func(r chi.Router) {

		// groups API endpoints
		r.Get("/groups", handler.GetGroups)
		r.Post("/groups", handler.AddGroupHandler)
		r.Get("/groups/{id}", handler.GetGroupHandler)
		r.Put("/groups/{name}", handler.UpdateGroupHandler)
		r.Delete("/groups/{id}", handler.RemoveGroupHandler)

		// impersonation API endpoints
		r.Get("/user-data/{id}", UserDataHandler)
		r.Get("/meta-data/{id}", MetaDataHandler(handler.sm, handler.store, clusterName))
		r.Get("/vendor-data/{id}", VendorDataHandler)
		r.Get("/instance-data/{id}", InstanceDataHandler(handler.sm, handler.store, clusterName))
		r.Get("/{group}.yaml/{id}", GroupUserDataHandler(handler.sm, handler.store))
	})
}
