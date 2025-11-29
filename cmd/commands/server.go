package commands

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wikilite/internal/api"
	"wikilite/pkg/models"
	"wikilite/pkg/utils"

	"github.com/spf13/cobra"
)

// newServerCmd creates the "serve" command to start the WikiLite server.
func newServerCmd(state *cliState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the WikiLite",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()

			seeded, err := state.DB.IsSeeded(ctx)
			if err != nil {
				log.Printf("Warning: Failed to check seed status: %v", err)
			}

			if !seeded {
				log.Println("Seeding database with default Admin user...")
				hash, err := utils.HashPassword("admin")
				if err != nil {
					log.Fatalf("Failed to hash seed password: %v", err)
				}

				adminUser := &models.User{
					Name:       "System Admin",
					Email:      "admin@example.com",
					Hash:       hash,
					Role:       models.ADMIN,
					IsExternal: false,
				}

				err = state.DB.Seed(ctx, adminUser, "Home")
				if err != nil {
					log.Fatalf("Failed to seed database: %v", err)
				}
				log.Println("Seeding complete. Login with admin@example.com / admin")
			}

			wikiName := api.DefaultWikiName
			if state.Config.WikiName != "" {
				wikiName = state.Config.WikiName
			}

			server, err := api.NewServer(
				state.DB,
				state.Config.JWTSecret,
				state.Config.JWKSURL,
				state.Config.JWTIssuer,
				state.Config.JWTEmailClaim,
				wikiName,
				state.Config.PluginPath,
				state.Config.PluginStoragePath,
				state.Config.JSPkgsPath,
			)
			if err != nil {
				log.Fatalf("Failed to create server: %v", err)
			}

			port := ":8080"
			log.Printf("Starting %s on %s", wikiName, port)

			if state.Config.JWKSURL != "" {
				log.Printf("Auth Mode: External IDP (JWKS: %s)", state.Config.JWKSURL)
				logAuthModeDetails(&state.Config)
			} else {
				log.Printf("Auth Mode: Local HMAC")
			}

			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

			go func() {
				err := server.Start(port)
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Printf("Server failed: %v", err)
					close(stop)
				}
			}()

			<-stop
			log.Println("Shutdown signal received...")

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err = server.Shutdown(ctx)
			if err != nil {
				log.Printf("Server forced to shutdown: %v", err)
			}

			err = server.Close()
			if err != nil {
				log.Printf("Error cleaning up server resources: %v", err)
			}

			log.Println("Server exited gracefully.")
		},
	}

	return cmd
}

// logAuthModeDetails logs authentication mode details.
func logAuthModeDetails(config *config) {
	if config.JWTEmailClaim != "" {
		log.Printf("Email Claim: Explicitly set to '%s'", config.JWTEmailClaim)
		return
	}
	log.Println("Email Claim: Auto-discovery mode")
}
