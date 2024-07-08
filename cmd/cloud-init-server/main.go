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

	router := chi.NewRouter()
	store := memstore.NewMemStore()
	sm := smdclient.NewSMDClient(smdEndpoint, smdToken)
	ciHandler := NewCiHandler(store, sm)

	router.Get("/cloud-init", ciHandler.ListEntries)
	router.Post("/cloud-init", ciHandler.AddEntry)
	router.Get("/cloud-init/{id}", ciHandler.GetEntry)
	router.Get("/cloud-init/{id}/user-data", ciHandler.GetUserData)
	router.Get("/cloud-init/{id}/meta-data", ciHandler.GetMetaData)
	router.Get("/cloud-init/{id}/vendor-data", ciHandler.GetVendorData)
	router.Put("/cloud-init/{id}", ciHandler.UpdateEntry)
	router.Delete("/cloud-init/{id}", ciHandler.DeleteEntry)

	http.ListenAndServe(ciEndpoint, router)
}
