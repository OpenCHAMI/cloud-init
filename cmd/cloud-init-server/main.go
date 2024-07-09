package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/OpenCHAMI/cloud-init/internal/memstore"
	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/jwtauth/v5"
	"github.com/go-chi/chi/v5"
)

var (
	ciEndpoint  = ":27777"
	smdEndpoint = "http://smd:27779"
	smdToken    = "" // jwt for access to smd
	jwksUrl     = "" // jwt keyserver URL for secure-route token validation
)

func main() {
	flag.StringVar(&ciEndpoint, "listen", ciEndpoint, "Server IP and port for cloud-init-server to listen on")
	flag.StringVar(&smdEndpoint, "smd-url", smdEndpoint, "http IP/url and port for running SMD")
	flag.StringVar(&smdToken, "smd-token", smdToken, "JWT token for SMD access")
	flag.StringVar(&jwksUrl, "jwks-url", jwksUrl, "JWT keyserver URL, required to enable secure route")
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

	// Primary router and shared SMD client
	router := chi.NewRouter()
	sm := smdclient.NewSMDClient(smdEndpoint, smdToken)

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
			jwtauth.Authenticator(keyset),
		)
		initCiRouter(router_sec, ciHandler_sec)
		router.Mount("/cloud-init-secure", router_sec)
	}

	// Serve all routes
	http.ListenAndServe(ciEndpoint, router)
}

func initCiRouter(router chi.Router, handler *CiHandler) {
	// Add cloud-init endpoints to router
	router.Get("/", handler.ListEntries)
	router.Post("/", handler.AddEntry)
	router.Get("/{id}", handler.GetEntry)
	router.Get("/{id}/user-data", handler.GetUserData)
	router.Get("/{id}/meta-data", handler.GetMetaData)
	router.Get("/{id}/vendor-data", handler.GetVendorData)
	router.Put("/{id}", handler.UpdateEntry)
	router.Delete("/{id}", handler.DeleteEntry)
}
