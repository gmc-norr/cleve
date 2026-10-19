package user

import (
	"errors"
	"fmt"
	"os"
	"slices"
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
	cmd.AddCommand(userUpdateCmd())
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

func userUpdateCmd() *cobra.Command {
	var (
		email  string
		userID uuid.UUID
		roles  []cleve.UserRole
	)
	cmd := &cobra.Command{
		Use:   "update EMAIL|ID",
		Short: "Update user details",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			rawRoles, _ := cmd.Flags().GetStringArray("role")
			roles = make([]cleve.UserRole, 0, len(rawRoles))
			for _, r := range rawRoles {
				role := cleve.UserRole(r)
				if !cleve.ValidRole(role) {
					return fmt.Errorf("invalid role: %s", r)
				}
				if !slices.Contains(roles, role) {
					roles = append(roles, role)
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			userID, err = uuid.Parse(args[0])
			if err != nil {
				email = args[0]
			}

			db, err := mongo.Connect()
			if err != nil {
				return nil
			}

			var user cleve.User
			if userID != uuid.Nil {
				user, err = db.GetUserByID(cmd.Context(), userID)
				if err != nil {
					return err
				}
			} else {
				var found bool
				user, found, err = db.GetUserByEmail(cmd.Context(), email)
				if err != nil {
					return err
				}
				if !found {
					return fmt.Errorf("no user with email %s found", email)
				}
			}

			newPassword, _ := cmd.Flags().GetString("password")
			newEmail, _ := cmd.Flags().GetString("email")
			newName, _ := cmd.Flags().GetString("name")
			appendRoles, _ := cmd.Flags().GetBool("append")

			updatedFields := false

			if newPassword != "" {
				newPasswordHash, err := auth.HashPassword([]byte(newPassword))
				if err != nil {
					return err
				}
				user.PasswordHash = string(newPasswordHash)
				updatedFields = true
			}

			if newEmail != "" {
				user.Email = newEmail
				updatedFields = true
			}

			if newName != "" {
				user.DisplayName = newName
				updatedFields = true
			}

			if len(roles) > 0 {
				if appendRoles {
					for _, r := range roles {
						user.AddRole(r)
					}
				} else {
					user.Roles = roles
				}
				updatedFields = true
			}

			if !updatedFields {
				return fmt.Errorf("no fields are being updated")
			}

			updated, err := db.UpdateUser(cmd.Context(), user)
			if err != nil {
				return err
			}
			if !updated {
				fmt.Println("user not updated")
			} else {
				fmt.Println("user updated")
			}
			return nil
		},
	}

	cmd.Flags().StringP("password", "p", "", "User password")
	cmd.Flags().StringP("email", "m", "", "User email")
	cmd.Flags().StringP("name", "n", "", "User display name")
	cmd.Flags().StringArrayP("role", "r", nil, "User role, existing roles will be replaced (multiple allowed)")
	cmd.Flags().BoolP("append", "a", false, "Use to append role to user instead of replacing")

	return cmd
}
