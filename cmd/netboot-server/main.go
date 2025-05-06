package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/OpenCHAMI/cloud-init/pkg/bss"
	"github.com/OpenCHAMI/cloud-init/pkg/smdclient"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	openchami_logger "github.com/openchami/chi-middleware/log"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"

	"github.com/spf13/cobra"
)

var (
	clusterName    string
	jwksUrl        string
	smdEndpoint    string
	tokenEndpoint  string
	accessToken    string
	certPath       string
	insecure       bool
	fakeSMDEnabled bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "netboot-server",
		Short: "Starts the netboot server for OpenCHAMI",
		Long:  `This command starts the netboot server for OpenCHAMI, providing boot parameter management and boot script generation.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Validate flags
			if smdEndpoint == "" && !fakeSMDEnabled {
				//cmd.Usage() // Print usage if validation fails
				return errors.New("either --smd-endpoint or --fake-smd must be set for inventory details")
			}
			if clusterName == "" {
				//cmd.Usage() // Print usage if validation fails
				return errors.New("cluster name must be set via --cluster-name or CLUSTER_NAME")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return startServer()
		},
	}

	// Define CLI flags
	rootCmd.Flags().StringVar(&clusterName, "cluster-name", os.Getenv("CLUSTER_NAME"), "Cluster name for SMD (default from CLUSTER_NAME env)")
	rootCmd.Flags().StringVar(&jwksUrl, "jwks-url", os.Getenv("JWKS_URL"), "JWKS URL for JWT validation (default from JWKS_URL env)")
	rootCmd.Flags().StringVar(&smdEndpoint, "smd-endpoint", os.Getenv("SMD_ENDPOINT"), "SMD endpoint (default from SMD_ENDPOINT env)")
	rootCmd.Flags().StringVar(&tokenEndpoint, "token-endpoint", os.Getenv("TOKEN_ENDPOINT"), "Token endpoint (default from TOKEN_ENDPOINT env)")
	rootCmd.Flags().StringVar(&accessToken, "access-token", os.Getenv("ACCESS_TOKEN"), "Access token (default from ACCESS_TOKEN env)")
	rootCmd.Flags().StringVar(&certPath, "cert-path", os.Getenv("CERT_PATH"), "Path to certificate for secure connections (default from CERT_PATH env)")
	rootCmd.Flags().BoolVar(&insecure, "insecure", os.Getenv("INSECURE") == "true", "Disable TLS verification (default from INSECURE env)")
	rootCmd.Flags().BoolVar(&fakeSMDEnabled, "fake-smd", os.Getenv("CLOUD_INIT_SMD_SIMULATOR") == "true", "Enable FakeSMDClient simulation (default from CLOUD_INIT_SMD_SIMULATOR env)")

	// Execute Cobra command
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// startServer encapsulates the existing main logic
func startServer() error {
	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Create SMD client
	var sm smdclient.SMDClientInterface
	if fakeSMDEnabled {
		fmt.Printf("\n\n**********\n\n\tCLOUD_INIT_SMD_SIMULATOR is set to true.\n\tUsing the FakeSMDClient\n\n**********\n\n\n")
		fakeSm := smdclient.NewFakeSMDClient(clusterName, 500)
		fakeSm.Summary()
		sm = fakeSm
	} else {
		var err error
		sm, err = smdclient.NewSMDClient(clusterName, smdEndpoint, tokenEndpoint, accessToken, certPath, insecure)
		if err != nil {
			log.Fatal().Err(err)
		}
	}

	// Initialize BSS storage
	bssStorage := bss.NewMemoryStore()

	// Set up router
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

	initBssRouter(router, sm, bssStorage)

	// Start the HTTP server
	log.Info().
		Str("cluster", clusterName).
		Str("smdEndpoint", smdEndpoint).
		Str("fakeSMDEnabled", fmt.Sprintf("%t", fakeSMDEnabled)).
		Str("tokenEndpoint", tokenEndpoint).
		Str("jwksUrl", jwksUrl).
		Str("accessToken", accessToken).
		Str("certPath", certPath).
		Bool("insecure", insecure).
		Msgf("Starting netboot server on port 8080")

	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal().Msgf("Failed to start server: %v", err)
	}
	return nil
}

// initBssRouter sets up bootparam routes
func initBssRouter(router chi.Router, sm smdclient.SMDClientInterface, bssStorage bss.Store) {
	router.Route("/boot/v1/", func(r chi.Router) {
		r.Post("/bootparams", bss.CreateBootParamsHandler(bssStorage))
		r.Get("/bootparams/{id}", bss.GetBootParamsHandler(bssStorage))
		r.Put("/bootparams/{id}", bss.UpdateBootParamsHandler(bssStorage))
		r.Get("/bootparams/{id}", bss.GetBootParamsHandler(bssStorage))
		r.Get("/bootscript", bss.GenerateBootScriptHandler(bssStorage, sm))
	})
	router.Route("/boot/v2/", func(r chi.Router) {
		r.Get("/bootparams/{id}", bss.GetBootParamsHandler(bssStorage))
		r.Put("/bootparams/{id}", bss.UpdateBootParamsHandler(bssStorage))
		r.Get("/bootparams/{id}", bss.GetBootParamsHandler(bssStorage))
		r.Get("/bootscript", bss.GenerateBootScriptHandler(bssStorage, sm))
	})
}
