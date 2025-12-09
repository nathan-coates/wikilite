package commands

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"wikilite/internal/api"
	"wikilite/internal/db"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// cliState holds the shared runtime state for the application.
type cliState struct {
	DB     *db.DB
	Config config
}

// config holds the environment configuration.
type config struct {
	DBPath            string
	LogDBPath         string
	JWTSecret         string
	JWKSURL           string
	JWTIssuer         string
	JWTEmailClaim     string
	WikiName          string
	PluginPath        string
	PluginStoragePath string
	JSPkgsPath        string
	Production        bool
	TrustProxyHeaders bool
	InsecureCookies   bool
	Port              int
}

// NewRootCmd creates the entire command tree and returns the root command.
func NewRootCmd() *cobra.Command {
	state := &cliState{}

	var portNumber int
	port := os.Getenv("PORT")
	if port != "" {
		cnvPort, err := strconv.Atoi(port)
		if err != nil {
			log.Fatalf("Invalid PORT value: %v", err)
		}

		portNumber = cnvPort
	} else {
		portNumber = api.DefaultPort
	}

	rootCmd := &cobra.Command{
		Use:   "wikilite",
		Short: "WikiLite CLI",
		Long:  `CLI for managing the WikiLite application.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			_ = godotenv.Load()

			state.Config = config{
				DBPath:            os.Getenv("DB_PATH"),
				LogDBPath:         os.Getenv("LOG_DB_PATH"),
				JWTSecret:         os.Getenv("JWT_SECRET"),
				JWKSURL:           os.Getenv("JWKS_URL"),
				JWTIssuer:         os.Getenv("JWT_ISSUER"),
				JWTEmailClaim:     os.Getenv("JWT_EMAIL_CLAIM"),
				WikiName:          os.Getenv("WIKI_NAME"),
				PluginPath:        os.Getenv("PLUGIN_PATH"),
				PluginStoragePath: os.Getenv("PLUGIN_STORAGE_PATH"),
				JSPkgsPath:        os.Getenv("JSPKGS_PATH"),
				Production:        !(os.Getenv("IS_DEVELOPMENT") == "true"),
				TrustProxyHeaders: os.Getenv("TRUST_PROXY_HEADERS") == "true",
				InsecureCookies:   os.Getenv("INSECURE_COOKIES") == "true",
				Port:              portNumber,
			}

			if state.Config.JWTSecret == "" && state.Config.JWKSURL == "" {
				return fmt.Errorf(
					"missing authentication configuration. Set either JWT_SECRET (for local auth) or JWKS_URL (for external IDP)",
				)
			}

			wikiDbPath := db.DefaultWikiDb
			if state.Config.DBPath != "" {
				wikiDbPath = state.Config.DBPath
			}

			logDbPath := db.DefaultLogDb
			if state.Config.LogDBPath != "" {
				logDbPath = state.Config.LogDBPath
			}

			var err error
			state.DB, err = db.New(
				"file:"+wikiDbPath+"?cache=shared",
				"file:"+logDbPath+"?cache=shared",
			)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}

			return nil
		},
		// PersistentPostRun ensures the DB is closed after the command finishes.
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if state.DB != nil {
				err := state.DB.Close()
				if err != nil {
					log.Printf("Error closing database: %v", err)
				}
			}
		},
	}

	rootCmd.AddCommand(newServerCmd(state))
	rootCmd.AddCommand(newPruneLogsCmd(state))
	rootCmd.AddCommand(newAddUserCmd(state))
	rootCmd.AddCommand(newRemoveUserCmd(state))
	rootCmd.AddCommand(newUpdateUserCmd(state))

	return rootCmd
}
