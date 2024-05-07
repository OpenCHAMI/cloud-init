package main

import (
	"flag"

	"github.com/OpenCHAMI/cloud-init/internal/memstore"
	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/gin-gonic/gin"
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

	router := gin.Default()
	store := memstore.NewMemStore()
	sm := smdclient.NewSMDClient(smdEndpoint, smdToken)
	ciHandler := NewCiHandler(store, sm)

	router.GET("/cloud-init", ciHandler.ListEntries)
	router.POST("/cloud-init", ciHandler.AddEntry)
	router.GET("/cloud-init/:id", ciHandler.GetEntry)
	router.GET("/cloud-init/:id/user-data", ciHandler.GetUserData)
	router.GET("/cloud-init/:id/meta-data", ciHandler.GetMetaData)
	router.GET("/cloud-init/:id/vendor-data", ciHandler.GetVendorData)
	router.PUT("/cloud-init/:id", ciHandler.UpdateEntry)
	router.DELETE("cloud-init/:id", ciHandler.DeleteEntry)

	router.Run(ciEndpoint)
}
