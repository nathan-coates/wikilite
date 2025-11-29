package commands

import (
	"context"
	"fmt"
	"log"
	"strings"
	"wikilite/pkg/models"
	"wikilite/pkg/utils"

	"github.com/spf13/cobra"
)

// newAddUserCmd creates the "add-user" command to directly add a user to the database.
func newAddUserCmd(state *cliState) *cobra.Command {
	var (
		email    string
		password string
		name     string
		role     string
		external bool
	)

	cmd := &cobra.Command{
		Use:   "add-user",
		Short: "Directly add a user to the database",
		Run: func(cmd *cobra.Command, args []string) {
			if email == "" || name == "" {
				log.Fatal("Error: --email and --name are required")
			}
			if !external && password == "" {
				log.Fatal("Error: --password is required for local users")
			}

			var userRole models.UserRole
			switch strings.ToLower(role) {
			case "admin":
				userRole = models.ADMIN
			case "write", "editor":
				userRole = models.WRITE
			case "read", "viewer":
				userRole = models.READ
			default:
				log.Fatalf("Invalid role: %s. Allowed: admin, write, read", role)
			}

			hash, err := utils.HashPassword(password)
			if err != nil {
				log.Fatalf("Failed to hash password: %v", err)
			}

			newUser := &models.User{
				Name:       name,
				Email:      email,
				Hash:       hash,
				Role:       userRole,
				IsExternal: external,
			}

			ctx := models.NewContextWithLogger(context.Background(), state.DB.CreateLogEntry)

			err = state.DB.CreateUser(ctx, newUser)
			if err != nil {
				log.Fatalf("Failed to create user: %v", err)
			}

			fmt.Printf(
				"User '%s' (%s) created successfully with role [%s].\n",
				name,
				email,
				strings.ToUpper(role),
			)
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "User email address (required)")
	cmd.Flags().StringVar(&name, "name", "", "Display name (required)")
	cmd.Flags().StringVar(&password, "password", "", "Password (required for local users)")
	cmd.Flags().StringVar(&role, "role", "read", "Role (read, write, admin)")
	cmd.Flags().BoolVar(&external, "external", false, "Is this user managed by an external IDP?")

	return cmd
}

// newRemoveUserCmd creates the "remove-user" command.
func newRemoveUserCmd(state *cliState) *cobra.Command {
	var email string

	cmd := &cobra.Command{
		Use:   "remove-user",
		Short: "Remove a user from the database",
		Run: func(cmd *cobra.Command, args []string) {
			if email == "" {
				log.Fatal("Error: --email is required")
			}

			ctx := models.NewContextWithLogger(context.Background(), state.DB.CreateLogEntry)

			user, err := state.DB.GetUserByEmail(ctx, email)
			if err != nil {
				log.Fatalf("Database error: %v", err)
			}
			if user == nil {
				log.Fatalf("User with email '%s' not found", email)
			}

			err = state.DB.DeleteUser(ctx, user.Id)
			if err != nil {
				log.Fatalf("Failed to delete user: %v", err)
			}

			fmt.Printf("User '%s' (ID: %d) has been removed.\n", email, user.Id)
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "Email of the user to remove (required)")

	return cmd
}

// newUpdateUserCmd creates the "update-user" command.
func newUpdateUserCmd(state *cliState) *cobra.Command {
	var (
		email    string
		password string
		role     string
		disable  bool
		enable   bool
	)

	cmd := &cobra.Command{
		Use:   "update-user",
		Short: "Update user role, password, or status",
		Run: func(cmd *cobra.Command, args []string) {
			if email == "" {
				log.Fatal("Error: --email is required")
			}

			ctx := models.NewContextWithLogger(context.Background(), state.DB.CreateLogEntry)

			user, err := state.DB.GetUserByEmail(ctx, email)
			if err != nil {
				log.Fatalf("Database error: %v", err)
			}
			if user == nil {
				log.Fatalf("User with email '%s' not found", email)
			}

			var columns []string

			if cmd.Flags().Changed("role") {
				switch strings.ToLower(role) {
				case "admin":
					user.Role = models.ADMIN
				case "write", "editor":
					user.Role = models.WRITE
				case "read", "viewer":
					user.Role = models.READ
				default:
					log.Fatalf("Invalid role: %s. Allowed: admin, write, read", role)
				}
				columns = append(columns, "role")
			}

			if cmd.Flags().Changed("password") {
				if password == "" {
					log.Fatal("Error: Password cannot be empty if flag is provided")
				}

				h, err := utils.HashPassword(password)
				if err != nil {
					log.Fatalf("Failed to hash password: %v", err)
				}

				user.Hash = h
				columns = append(columns, "hash")
			}

			if disable && enable {
				log.Fatal("Error: Cannot specify both --enable and --disable")
			}
			if disable {
				user.Disabled = true
				columns = append(columns, "disabled")
			} else if enable {
				user.Disabled = false
				columns = append(columns, "disabled")
			}

			if len(columns) == 0 {
				fmt.Println("No changes specified.")

				return
			}

			err = state.DB.UpdateUser(ctx, user, columns...)
			if err != nil {
				log.Fatalf("Failed to update user: %v", err)
			}

			fmt.Printf("User '%s' updated successfully.\n", email)
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "Email of the user to update (required)")
	cmd.Flags().StringVar(&role, "role", "", "New role (read, write, admin)")
	cmd.Flags().StringVar(&password, "password", "", "New password")
	cmd.Flags().BoolVar(&disable, "disable", false, "Disable the user account")
	cmd.Flags().BoolVar(&enable, "enable", false, "Enable the user account")

	return cmd
}
