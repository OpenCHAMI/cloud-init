package main

import (
	"flag"

	"github.com/OpenCHAMI/cloud-init/internal/memstore"
	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/gin-gonic/gin"
)

var (
	ciEndpoint  = ":27777"
	smdEndpoint = "http://localhost:27779"
)

func main() {
	flag.StringVar(&ciEndpoint, "ci-listen", ciEndpoint, "Server IP and port for cloud-init-server to listen on")
	flag.StringVar(&smdEndpoint, "smd-endpoint", smdEndpoint, "http IP/url and port for running SMD")
	flag.Parse()

	router := gin.Default()
	store := memstore.NewMemStore()
	sm := smdclient.NewSMDClient(smdEndpoint)
	ciHandler := NewCiHandler(store, sm)

	router.GET("/harbor", ciHandler.ListEntries)
	router.POST("/harbor", ciHandler.AddEntry)
	router.GET("/harbor/:id", ciHandler.GetEntry)
	router.GET("/harbor/:id/user-data", ciHandler.GetUserData)
	router.GET("/harbor/:id/meta-data", ciHandler.GetMetaData)
	router.GET("/harbor/:id/vendor-data", ciHandler.GetVendorData)
	router.PUT("/harbor/:id", ciHandler.UpdateEntry)
	router.DELETE("harbor/:id", ciHandler.DeleteEntry)

	router.Run(ciEndpoint)
}
