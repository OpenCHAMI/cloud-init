package main

import (
	"flag"
	"net/http"

	"github.com/OpenCHAMI/cloud-init/internal/memstore"
	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/go-chi/chi/v5"
)

var (
	ciEndpoint  = ":27777"
	smdEndpoint = "http://smd:27779"
	smdToken    = "" // jwt for access to smd
)

func main() {
	flag.StringVar(&ciEndpoint, "listen", ciEndpoint, "Server IP and port for cloud-init-server to listen on")
	flag.StringVar(&smdEndpoint, "smd-url", smdEndpoint, "http IP/url and port for running SMD")
	flag.StringVar(&smdToken, "smd-token", smdToken, "JWT token for SMD access")
	flag.Parse()

	// Primary router and shared SMD client
	router := chi.NewRouter()
	sm := smdclient.NewSMDClient(smdEndpoint, smdToken)

	// Unsecured datastore and router
	store := memstore.NewMemStore()
	ciHandler := NewCiHandler(store, sm)
	router_unsec := newCiRouter(ciHandler)
	router.Mount("/cloud-init", router_unsec)

	// Secured datastore and router
	store_sec := memstore.NewMemStore()
	ciHandler_sec := NewCiHandler(store_sec, sm)
	router_sec := newCiRouter(ciHandler_sec)
	router.Mount("/cloud-init-secure", router_sec)

	// Serve all routes
	http.ListenAndServe(ciEndpoint, router)
}

func newCiRouter(handler *CiHandler) chi.Router {
	// Create a fresh Router with cloud-init endpoints
	router := chi.NewRouter()
	router.Get("/", handler.ListEntries)
	router.Post("/", handler.AddEntry)
	router.Get("/{id}", handler.GetEntry)
	router.Get("/{id}/user-data", handler.GetUserData)
	router.Get("/{id}/meta-data", handler.GetMetaData)
	router.Get("/{id}/vendor-data", handler.GetVendorData)
	router.Put("/{id}", handler.UpdateEntry)
	router.Delete("/{id}", handler.DeleteEntry)
	return router
}
