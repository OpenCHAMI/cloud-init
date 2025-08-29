package main

//	@Title			OpenCHAMI Cloud-Init Server API
//	@Version		1.0.0
//	@Description	API for cloud-init clients using the OpenCHAMI cloud-init server
//	@License.name	MIT
//	@License.url	https://github.com/OpenCHAMI/.github/blob/main/LICENSE

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/OpenCHAMI/cloud-init/internal/memstore"
	openchami_middleware "github.com/OpenCHAMI/cloud-init/internal/middleware"
	"github.com/OpenCHAMI/cloud-init/internal/quackstore"
	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/OpenCHAMI/cloud-init/pkg/wgtunnel"
	"github.com/OpenCHAMI/jwtauth/v5"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	openchami_authenticator "github.com/openchami/chi-middleware/auth"
	openchami_logger "github.com/openchami/chi-middleware/log"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	ciEndpoint           string
	tokenEndpoint        string
	smdEndpoint          string
	jwksUrl              string
	insecure             bool
	accessToken          string
	certPath             string
	clusterName          string
	region               string
	availabilityZone     string
	cloudProvider        string
	baseUrl              string
	fakeSMDEnabled       bool
	impersonationEnabled bool
	wireguardServer      string
	wireguardOnly        bool
	debug                bool
	wireGuardMiddleware  func(http.Handler) http.Handler
	storageBackend       = "mem"           // Default to memstore
	dbPath               = "cloud-init.db" // Default database path for quackstore
	store                cistore.Store
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "cloud-init-server",
		Short: "Starts the cloud-init server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return startServer()
		},
	}

	// Use Viper to read environment variables.
	// Bind each flag to an env var using Viper conventions.
	// Example: CLI flag --listen → environment var LISTEN
	setupFlags(rootCmd.Flags())
	bindViperToFlags()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// setupFlags defines all CLI flags with defaults reading from environment vars.
func setupFlags(flags *pflag.FlagSet) {
	flags.StringVar(&ciEndpoint, "listen", getEnv("LISTEN", "0.0.0.0:27777"), "Server IP and port for cloud-init-server to listen on")
	flags.StringVar(&tokenEndpoint, "token-url", getEnv("TOKEN_URL", "http://opaal:3333/token"), "OIDC server endpoint to fetch new tokens from (for SMD access)")
	flags.StringVar(&smdEndpoint, "smd-url", getEnv("SMD_URL", "http://smd:27779"), "Server host and port for running SMD (do not include /hsm/v2)")
	flags.StringVar(&jwksUrl, "jwks-url", getEnv("JWKS_URL", ""), "JWT keyserver URL, required to enable secure route")
	flags.StringVar(&accessToken, "access-token", getEnv("ACCESS_TOKEN", ""), "Encoded JWT access token")
	flags.StringVar(&clusterName, "cluster-name", getEnv("CLUSTER_NAME", ""), "Name of the cluster")
	flags.StringVar(&region, "region", getEnv("REGION", ""), "Region of the cluster")
	flags.StringVar(&availabilityZone, "az", getEnv("AZ", ""), "Availability zone of the cluster")
	flags.StringVar(&cloudProvider, "cloud-provider", getEnv("CLOUD_PROVIDER", ""), "Cloud provider of the cluster")
	flags.StringVar(&baseUrl, "base-url", getEnv("BASE_URL", ""), "Base URL for cloud-init-server including protocol and port (e.g. http://localhost:27777)")
	flags.StringVar(&certPath, "cacert", getEnv("CACERT", ""), "Path to CA cert (defaults to system CAs)")
	flags.BoolVar(&insecure, "insecure", parseBool(getEnv("INSECURE", "false")), "Set to bypass TLS verification for requests")
	flags.BoolVar(&impersonationEnabled, "impersonation", parseBool(getEnv("IMPERSONATION", "false")), "Enable impersonation feature")
	flags.StringVar(&wireguardServer, "wireguard-server", getEnv("WIREGUARD_SERVER", ""), "WireGuard server IP address and network (e.g. 100.97.0.1/16)")
	flags.BoolVar(&wireguardOnly, "wireguard-only", parseBool(getEnv("WIREGUARD_ONLY", "false")), "Only allow access to the cloud-init functions from the WireGuard subnet")
	flags.BoolVar(&debug, "debug", parseBool(getEnv("DEBUG", "false")), "Enable debug logging")
	flags.StringVar(&storageBackend, "storage-backend", getEnv("STORAGE_BACKEND", "mem"), "Storage backend to use (mem or quack)")
	flags.StringVar(&dbPath, "db-path", getEnv("DB_PATH", "cloud-init.db"), "Path to the database file for quackstore backend")
}

// bindViperToFlags binds each flag to Viper so environment variables work seamlessly.
func bindViperToFlags() {
	viper.AutomaticEnv()
	_ = viper.BindEnv("listen")
	_ = viper.BindEnv("token_url")
	_ = viper.BindEnv("smd_url")
	_ = viper.BindEnv("jwks_url")
	_ = viper.BindEnv("access_token")
	_ = viper.BindEnv("cluster_name")
	_ = viper.BindEnv("region")
	_ = viper.BindEnv("az")
	_ = viper.BindEnv("cloud_provider")
	_ = viper.BindEnv("base_url")
	_ = viper.BindEnv("cacert")
	_ = viper.BindEnv("insecure")
	_ = viper.BindEnv("impersonation")
	_ = viper.BindEnv("wireguard_server")
	_ = viper.BindEnv("wireguard_only")
	_ = viper.BindEnv("debug")
	_ = viper.BindEnv("storage_backend")
	_ = viper.BindEnv("db_path")
}

// startServer is where we run our main program logic
func startServer() error {
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().
			Str("listen", ciEndpoint).
			Str("token-url", tokenEndpoint).
			Str("smd-url", smdEndpoint).
			Str("jwks-url", jwksUrl).
			Str("access-token", accessToken).
			Str("cluster-name", clusterName).
			Str("region", region).
			Str("az", availabilityZone).
			Str("cloud-provider", cloudProvider).
			Str("base-url", baseUrl).
			Str("cacert", certPath).
			Bool("insecure", insecure).
			Bool("impersonation", impersonationEnabled).
			Str("wireguard-server", wireguardServer).
			Bool("wireguard-only", wireguardOnly).
			Bool("debug", debug).
			Str("storage-backend", storageBackend).
			Str("db-path", dbPath).
			Msg("Resolved configuration")
	}

	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Initialize storage backend
	var err error
	switch storageBackend {
	case "mem":
		store = memstore.NewMemStore()
	case "quack":
		store, err = quackstore.NewQuackStore(dbPath)
		if err != nil {
			return fmt.Errorf("failed to initialize quackstore: %w", err)
		}
	default:
		return fmt.Errorf("unsupported storage backend: %s", storageBackend)
	}

	// Setup JWKS if provided
	var keyset *jwtauth.JWTAuth
	secureRouteEnable := false
	if jwksUrl != "" {
		var err error
		keyset, err = fetchPublicKeyFromURL(jwksUrl)
		if err != nil {
			fmt.Printf("JWKS initialization failed: %s\n", err)
		} else {
			secureRouteEnable = true
		}
	} else {
		fmt.Println("No JWKS URL provided; secure route will be disabled")
	}

	// Create SMD client
	var sm smdclient.SMDClientInterface
	if os.Getenv("CLOUD_INIT_SMD_SIMULATOR") == "true" {
		fmt.Printf("\n\n**********\n\n\tCLOUD_INIT_SMD_SIMULATOR is set to true in your environment.\n\n\tUsing the FakeSMDClient\n\n**********\n\n\n")
		sm = smdclient.NewFakeSMDClient(clusterName, 500)
	} else {
		sm, err = smdclient.NewSMDClient(clusterName, smdEndpoint, tokenEndpoint, accessToken, certPath, insecure)
		if err != nil {
			return fmt.Errorf("failed to create SMD client: %w", err)
		}
	}

	// Create CI handler
	handler := &CiHandler{
		sm:    sm,
		store: store,
	}

	// Setup WireGuard middleware if enabled
	if wireguardOnly && wireguardServer != "" {
		wireGuardMiddleware = openchami_middleware.WireGuardMiddlewareWithInterface("wg0", wireguardServer)
	}

	// Create router
	router := chi.NewRouter()

	// Add middleware
	router.Use(
		middleware.RequestID,
		middleware.RealIP,
		middleware.Logger,
		middleware.Recoverer,
		middleware.StripSlashes,
		middleware.Timeout(60*time.Second),
		openchami_logger.OpenCHAMILogger(log.Logger),
	)

	// Add WireGuard middleware if enabled
	if wireguardOnly && wireGuardMiddleware != nil {
		router.Use(wireGuardMiddleware)
	}

	// Setup routes
	initCiClientRouter(router, handler, nil)
	initCiAdminRouter(router, handler)

	// Add secure routes if JWKS is configured
	if secureRouteEnable && keyset != nil {
		secureRouter := chi.NewRouter()
		secureRouter.Use(
			jwtauth.Verifier(keyset),
			openchami_authenticator.AuthenticatorWithRequiredClaims(keyset, []string{"sub", "iss", "aud"}),
		)

		// Add secure routes here if needed
		router.Mount("/secure", secureRouter)
	}

	// Start server
	fmt.Printf("Starting cloud-init server on %s\n", ciEndpoint)
	return http.ListenAndServe(ciEndpoint, router)
}

// Utility to read optional environment variables
func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

// parseBool is a helper to convert string "true" or "false" to bool
func parseBool(str string) bool {
	return strings.EqualFold(str, "true") || str == "1"
}

func initCiClientRouter(router chi.Router, handler *CiHandler, wgInterfaceManager *wgtunnel.InterfaceManager) {
	// Add cloud-init endpoints to router
	router.Get("/openapi.json", DocsHandler)
	router.Get("/version", VersionHandler)
	if wireGuardMiddleware != nil {
		router.With(wireGuardMiddleware).Get("/user-data", UserDataHandler)
		router.With(wireGuardMiddleware).Get("/meta-data", MetaDataHandler(handler.sm, handler.store))
		router.With(wireGuardMiddleware).Get("/vendor-data", VendorDataHandler(handler.sm, handler.store))
		router.With(wireGuardMiddleware).Get("/{group}.yaml", GroupUserDataHandler(handler.sm, handler.store))
	} else {
		router.Get("/user-data", UserDataHandler)
		router.Get("/meta-data", MetaDataHandler(handler.sm, handler.store))
		router.Get("/vendor-data", VendorDataHandler(handler.sm, handler.store))
		router.Get("/{group}.yaml", GroupUserDataHandler(handler.sm, handler.store))
	}
	router.Post("/phone-home/{id}", PhoneHomeHandler(wgInterfaceManager, handler.sm))
	router.Post("/wg-init", wgtunnel.AddClientHandler(wgInterfaceManager, handler.sm))
}

func initCiAdminRouter(router chi.Router, handler *CiHandler) {
	// admin API subrouter
	router.Route("/admin/", func(r chi.Router) {

		// Cluster Defaults
		r.Get("/cluster-defaults", GetClusterDataHandler(handler.store))
		r.Post("/cluster-defaults", SetClusterDataHandler(handler.store))
		// r.Put("/cluster-defaults", SetClusterDataHandler(handler.store)) // Should we support PUT and POST or just one of them?

		r.Put("/instance-info/{id}", InstanceInfoHandler(handler.sm, handler.store))

		// groups API endpoints
		r.Get("/groups", handler.GetGroups)
		r.Post("/groups", handler.AddGroupHandler)
		r.Get("/groups/{id}", handler.GetGroupHandler)
		r.Put("/groups/{name}", handler.UpdateGroupHandler)
		r.Delete("/groups/{id}", handler.RemoveGroupHandler)

		if impersonationEnabled {
			// impersonation API endpoints
			r.Get("/impersonation/{id}/user-data", UserDataHandler)
			r.Get("/impersonation/{id}/meta-data", MetaDataHandler(handler.sm, handler.store))
			r.Get("/impersonation/{id}/vendor-data", VendorDataHandler(handler.sm, handler.store))
			r.Get("/impersonation/{id}/{group}.yaml", GroupUserDataHandler(handler.sm, handler.store))
		}

		if fakeSMDEnabled {
			r.Post("/fake-sm/nodes", smdclient.AddNodeToInventoryHandler(handler.sm.(*smdclient.FakeSMDClient)))
			r.Get("/fake-sm/nodes", smdclient.ListNodesHandler(handler.sm.(*smdclient.FakeSMDClient)))
			r.Put("/fake-sm/nodes/{id}", smdclient.UpdateNodeHandler(handler.sm.(*smdclient.FakeSMDClient)))
		}

	})
}
