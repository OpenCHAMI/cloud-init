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
	// Parse command-line flags for configuration
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
		middleware.RequestID,               // Adds a request ID to each request
		middleware.RealIP,                  // Sets the RemoteAddr to the client's IP address
		middleware.Logger,                  // Logs the start and end of each request
		middleware.Recoverer,               // Recovers from panics and returns a 500 error
		middleware.StripSlashes,            // Strips slashes from the URL
		middleware.Timeout(60*time.Second), // Sets a timeout for each request
	)
	sm := smdclient.NewSMDClient(smdEndpoint, tokenEndpoint)

	// datastore and handlers
	store := memstore.NewMemStore()
	ciHandler := NewCiHandler(store, sm)

	// Unsecured router for GET requests
	router_unsec := chi.NewRouter()
	initCiRouter(router_unsec, nil, ciHandler)
	router.Mount("/cloud-init", router_unsec)

	// Secured router for POST/PUT/DELETE requests
	router_sec := chi.NewRouter()
	router_sec.Use(
		jwtauth.Verifier(keyset),      // Verifies JWT tokens
		jwtauth.Authenticator(keyset), // Authenticates requests using JWT tokens
	)
	initCiRouter(nil, router_sec, ciHandler)
	router.Mount("/cloud-init", router_sec)

	// Secure router for all operations
	store_sec := memstore.NewMemStore()
	ciHandler_sec := NewCiHandler(store_sec, sm)
	router_secure := chi.NewRouter()
	router_secure.Use(
		jwtauth.Verifier(keyset),      // Verifies JWT tokens
		jwtauth.Authenticator(keyset), // Authenticates requests using JWT tokens
	)
	initCiRouter(router_secure, router_secure, ciHandler_sec)
	router.Mount("/cloud-init-secure", router_secure)

	// Serve all routes
	http.ListenAndServe(ciEndpoint, router)
}

func initCiRouter(getRouter chi.Router, setRouter chi.Router, handler *CiHandler) {
	if getRouter != nil {
		// Add cloud-init GET endpoints to router
		getRouter.Get("/", handler.ListEntries)
		getRouter.Get("/user-data", handler.GetDataByIP(UserData))
		getRouter.Get("/meta-data", handler.GetDataByIP(MetaData))
		getRouter.Get("/vendor-data", handler.GetDataByIP(VendorData))
		getRouter.Get("/{id}", handler.GetEntry)
		getRouter.Get("/{id}/user-data", handler.GetDataByMAC(UserData))
		getRouter.Get("/{id}/meta-data", handler.GetDataByMAC(MetaData))
		getRouter.Get("/{id}/vendor-data", handler.GetDataByMAC(VendorData))
	}
	if setRouter != nil {
		// Add cloud-init POST/PUT/DELETE endpoints to router
		setRouter.Post("/", handler.AddEntry)
		setRouter.Put("/{id}", handler.UpdateEntry)
		setRouter.Delete("/{id}", handler.DeleteEntry)
	}
}
