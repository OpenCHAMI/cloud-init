package bss

import (
	"github.com/OpenCHAMI/cloud-init/pkg/smdclient"
	"github.com/go-chi/chi/v5"
)

// NewBSSRouter creates a new router with all the BSS endpoints
// This is an example router which will work in some circumstances.
// It is not intended to be used in production.
func NewBSSRouter(store Store, smd *smdclient.SMDClient) chi.Router {
	r := chi.NewRouter()
	r.Route("/boot/v1", func(r chi.Router) {

		r.Put("/bootparameters", V1AddBootParamsHandler(store, smd))
		r.Post("/bootparameters", V1AddBootParamsHandler(store, smd))

		// Boot script endpoint
		r.Get("/bootscript", GenerateBootScriptHandler(store, smd))
	})
	r.Route("/boot/v2", func(r chi.Router) {
		// Boot parameters endpoints
		r.Post("/bootparams", CreateBootParamsHandler(store))
		r.Get("/bootparams/{id}", GetBootParamsHandler(store))
		r.Put("/bootparams/{id}", UpdateBootParamsHandler(store))
		r.Get("/bootparams/{id}", GetBootParamsHandler(store))
		// Boot script endpoint
		r.Get("/bootscript", GenerateBootScriptHandler(store, smd))
	})

	return r
}
