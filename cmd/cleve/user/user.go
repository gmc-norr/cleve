package user

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/internal/auth"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func UserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}
	cmd.AddCommand(userListCmd())
	cmd.AddCommand(userCreateCmd())
	cmd.AddCommand(userDeleteCmd())
	return cmd
}

func userListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List users",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := mongo.Connect()
			if err != nil {
				return err
			}
			users, err := db.Users(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to fetch users: %w", err)
			}
			w := tabwriter.NewWriter(os.Stdout, 4, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "id\tdisplay name\temail\troles")
			for _, u := range users {
				var roles []string
				for _, r := range u.Roles {
					roles = append(roles, r.String())
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", u.ID, u.DisplayName, u.Email, strings.Join(roles, ", "))
			}
			_ = w.Flush()
			return nil
		},
	}
}

func userCreateCmd() *cobra.Command {
	var (
		stringRoles []string
		roles       []cleve.UserRole
	)
	cmd := &cobra.Command{
		Use:   "create NAME EMAIL",
		Short: "Create a new user",
		Args:  cobra.ExactArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(stringRoles) == 0 {
				return fmt.Errorf("no role assigned to user")
			}
			var errs []error
			for _, role := range stringRoles {
				if !cleve.ValidRole(cleve.UserRole(role)) {
					errs = append(errs, fmt.Errorf("invalid role: %s", role))
				}
				roles = append(roles, cleve.UserRole(role))
			}
			return errors.Join(errs...)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			pwd, _ := cmd.Flags().GetString("password")
			if pwd == "" {
				var err error
				pwd, err = auth.GenerateTempPassword()
				if err != nil {
					return err
				}
			}
			hashedPwd, err := auth.HashPassword([]byte(pwd))
			if err != nil {
				return err
			}
			db, err := mongo.Connect()
			if err != nil {
				return err
			}
			user := cleve.User{
				DisplayName:  args[0],
				Email:        args[1],
				PasswordHash: string(hashedPwd),
				Roles:        roles,
			}
			if err = db.CreateUser(cmd.Context(), user); err != nil {
				return err
			}
			fmt.Printf("Created user %s (%s) with role(s): %s\n", user.DisplayName, user.Email, strings.Join(stringRoles, ", "))
			fmt.Printf("Temporary password: %s\n", pwd)
			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&stringRoles, "role", "r", []string{"user"}, "Role to assign to user (multiple allowed)")
	cmd.Flags().StringP("password", "p", "", "User password")

	return cmd
}

func userDeleteCmd() *cobra.Command {
	var (
		email string
		id    uuid.UUID
	)
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a user",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			email, _ = cmd.Flags().GetString("email")
			rawID, _ := cmd.Flags().GetString("id")
			if rawID != "" {
				var err error
				id, err = uuid.Parse(rawID)
				if err != nil {
					return err
				}
			}
			if email != "" && id != uuid.Nil {
				return fmt.Errorf("give either an email or ID, not both")
			}
			if email == "" && id == uuid.Nil {
				return fmt.Errorf("supply either email or ID for the user to be deleted")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := mongo.Connect()
			if err != nil {
				return err
			}

			if email != "" {
				user, found, err := db.GetUserByEmail(cmd.Context(), email)
				if err != nil {
					return err
				}
				if !found {
					return fmt.Errorf("user not found")
				}
				id = user.ID
			}

			ok, err := db.DeleteUser(cmd.Context(), id)
			if err != nil {
				return err
			}
			if ok {
				fmt.Printf("user with ID %s successfully deleted\n", id)
			} else {
				fmt.Println("user not found")
			}
			return nil
		},
	}

	cmd.Flags().StringP("email", "m", "", "Email of user to delete")
	cmd.Flags().String("id", "", "ID of user to delete")

	return cmd
}
