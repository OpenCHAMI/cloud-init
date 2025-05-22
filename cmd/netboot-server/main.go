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
	storageBackend string
	dbPath         string
)

func main() {
	// Create root command
	rootCmd := &cobra.Command{
		Use:   "netboot-server",
		Short: "OpenCHAMI netboot server",
		Long: `OpenCHAMI netboot server provides boot parameter management and boot script generation.
It supports both in-memory and persistent storage backends, and can integrate with SMD for inventory details.`,
	}

	// Create serve command
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the netboot server",
		Long:  `Start the netboot server with the specified configuration.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if smdEndpoint == "" && !fakeSMDEnabled {
				return errors.New("either --smd-endpoint or --fake-smd must be set for inventory details")
			}
			if clusterName == "" {
				return errors.New("cluster name must be set via --cluster-name or CLUSTER_NAME")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return startServer()
		},
	}

	// Add flags to serve command
	serveCmd.Flags().StringVar(&clusterName, "cluster-name", os.Getenv("CLUSTER_NAME"), "Cluster name for SMD (default from CLUSTER_NAME env)")
	serveCmd.Flags().StringVar(&jwksUrl, "jwks-url", os.Getenv("JWKS_URL"), "JWKS URL for JWT validation (default from JWKS_URL env)")
	serveCmd.Flags().StringVar(&smdEndpoint, "smd-endpoint", os.Getenv("SMD_ENDPOINT"), "SMD endpoint (default from SMD_ENDPOINT env)")
	serveCmd.Flags().StringVar(&tokenEndpoint, "token-endpoint", os.Getenv("TOKEN_ENDPOINT"), "Token endpoint (default from TOKEN_ENDPOINT env)")
	serveCmd.Flags().StringVar(&accessToken, "access-token", os.Getenv("ACCESS_TOKEN"), "Access token (default from ACCESS_TOKEN env)")
	serveCmd.Flags().StringVar(&certPath, "cert-path", os.Getenv("CERT_PATH"), "Path to certificate for secure connections (default from CERT_PATH env)")
	serveCmd.Flags().BoolVar(&insecure, "insecure", os.Getenv("INSECURE") == "true", "Disable TLS verification (default from INSECURE env)")
	serveCmd.Flags().BoolVar(&fakeSMDEnabled, "fake-smd", os.Getenv("CLOUD_INIT_SMD_SIMULATOR") == "true", "Enable FakeSMDClient simulation (default from CLOUD_INIT_SMD_SIMULATOR env)")
	serveCmd.Flags().StringVar(&storageBackend, "storage-backend", "mem", "Storage backend to use (mem or quack)")
	serveCmd.Flags().StringVar(&dbPath, "db-path", "netboot.db", "Path to the database file for quackstore backend")

	// Create version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			PrintVersionInfo()
		},
	}

	// Add commands to root
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(versionCmd)

	// Execute root command
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

	// Initialize BSS storage based on the selected backend
	var bssStorage bss.Store
	var err error
	switch storageBackend {
	case "mem":
		bssStorage = bss.NewMemoryStore()
	case "quack":
		bssStorage, err = bss.NewQuackStore(dbPath)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize QuackStore")
		}
		defer bssStorage.(*bss.QuackStore).Close()
	default:
		log.Fatal().Msgf("Unsupported storage backend: %s", storageBackend)
	}

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
		Str("storageBackend", storageBackend).
		Str("dbPath", dbPath).
		Str("version", Version).
		Str("commit", GitCommit).
		Msgf("Starting netboot server on port 8080")

	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal().Msgf("Failed to start server: %v", err)
	}
	return nil
}

// initBssRouter sets up bootparam routes
func initBssRouter(router chi.Router, sm smdclient.SMDClientInterface, bssStorage bss.Store) {
	router.Get("/version", VersionHandler)
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
