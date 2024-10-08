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
)

var (
	ciEndpoint    = ":27777"
	tokenEndpoint = "http://opaal:3333/token" // jwt for smd access obtained from here
	smdEndpoint   = "http://smd:27779"
	jwksUrl       = "http://opaal:3333/keys" // jwt keyserver URL for secure-route token validation
)

func main() {
	flag.StringVar(&ciEndpoint, "listen", ciEndpoint, "Server IP and port for cloud-init-server to listen on")
	flag.StringVar(&tokenEndpoint, "token-url", tokenEndpoint, "OIDC server URL (endpoint) to fetch new tokens from (for SMD access)")
	flag.StringVar(&smdEndpoint, "smd-url", smdEndpoint, "http IP/url and port for running SMD")
	flag.StringVar(&jwksUrl, "jwks-url", jwksUrl, "JWT keyserver URL, for JWT validation")
	flag.Parse()

	// Set up JWT verification via the specified URL, if any
	var keyset *jwtauth.JWTAuth
	var err error
	fmt.Printf("Initializing JWKS from URL: %s\n", jwksUrl)
	keyset, err = fetchPublicKeyFromURL(jwksUrl)
	if err != nil {
		fmt.Printf("JWKS initialization failed: %s\n", err)
		os.Exit(2)
	}

	// Primary router and shared SMD client
	router := chi.NewRouter()
	router.Use(
		middleware.RequestID,
		middleware.RealIP,
		middleware.Logger,
		middleware.Recoverer,
		middleware.StripSlashes,
		middleware.Timeout(60 * time.Second),
	)
	sm := smdclient.NewSMDClient(smdEndpoint, tokenEndpoint)

	// Unsecured datastore and routers
	store := memstore.NewMemStore()
	ciHandler := NewCiHandler(store, sm)
	router_unsec := chi.NewRouter()
	// This "unsecured" router still does security checking, and handles
	// (sensitive) write requests
	router_unsec_writes := chi.NewRouter()
	router_unsec_writes.Use(
		jwtauth.Verifier(keyset),
		jwtauth.Authenticator(keyset),
	)
	initCiRouter(router_unsec, router_unsec_writes, ciHandler)
	router.Mount("/cloud-init", router_unsec)
	router.Mount("/cloud-init", router_unsec_writes)

	// Secured datastore and router
	store_sec := memstore.NewMemStore()
	ciHandler_sec := NewCiHandler(store_sec, sm)
	router_sec := chi.NewRouter()
	router_sec.Use(
		jwtauth.Verifier(keyset),
		jwtauth.Authenticator(keyset),
	)
	initCiRouter(router_sec, router_sec, ciHandler_sec)
	router.Mount("/cloud-init-secure", router_sec)

	// Serve all routes
	http.ListenAndServe(ciEndpoint, router)
}

func initCiRouter(getRouter chi.Router, setRouter chi.Router, handler *CiHandler) {
	// Add cloud-init endpoints to router
	getRouter.Get("/", handler.ListEntries)
	getRouter.Get("/user-data", handler.GetDataByIP(UserData))
	getRouter.Get("/meta-data", handler.GetDataByIP(MetaData))
	getRouter.Get("/vendor-data", handler.GetDataByIP(VendorData))
	getRouter.Get("/{id}", handler.GetEntry)
	getRouter.Get("/{id}/user-data", handler.GetDataByMAC(UserData))
	getRouter.Get("/{id}/meta-data", handler.GetDataByMAC(MetaData))
	getRouter.Get("/{id}/vendor-data", handler.GetDataByMAC(VendorData))
	setRouter.Post("/", handler.AddEntry)
	setRouter.Put("/{id}", handler.UpdateEntry)
	setRouter.Delete("/{id}", handler.DeleteEntry)
}
