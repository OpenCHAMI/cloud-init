package bss

import (
	"net/http"

	"github.com/OpenCHAMI/cloud-init/pkg/smdclient"
	"github.com/go-chi/chi/v5"
)

// NewRouter creates a new router with all the BSS endpoints
func NewBSSRouter(store Store, smd *smdclient.SMDClient) http.Handler {
	r := chi.NewRouter()

	// Boot parameters endpoints
	r.Post("/bootparams", CreateBootParamsHandler(store))
	r.Get("/bootparams/{id}", GetBootParamsHandler(store))
	r.Put("/bootparams/{id}", UpdateBootParamsHandler(store))
	r.Get("/bootparams/{id}", GetBootParamsHandler(store))

	r.Put("/bootparameters", V1AddBootParamsHandler(store, smd))
	r.Post("/bootparameters", V1AddBootParamsHandler(store, smd))

	// Boot script endpoint
	r.Get("/bootscript", GenerateBootScriptHandler(store, smd))

	return r
}
