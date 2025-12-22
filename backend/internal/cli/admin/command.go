package admin

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"strings"

	"github.com/Anvoria/authly/internal/config"
	"github.com/Anvoria/authly/internal/database"
	"github.com/Anvoria/authly/internal/domain/permission"
	"github.com/Anvoria/authly/internal/domain/role"
	svc "github.com/Anvoria/authly/internal/domain/service"
	"github.com/Anvoria/authly/internal/domain/user"
	"github.com/Anvoria/authly/internal/migrations"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Command implements the admin management command
type Command struct{}

func (c *Command) Name() string {
	return "admin"
}

func (c *Command) Description() string {
	return "Administration tasks (init-root)"
}

func (c *Command) Run(args []string) error {
	if len(args) < 1 {
		c.printUsage()
		return fmt.Errorf("subcommand required")
	}

	subcmd := args[0]
	switch subcmd {
	case "init-root":
		return c.runInitRoot(args[1:])
	default:
		c.printUsage()
		return fmt.Errorf("unknown subcommand: %s", subcmd)
	}
}

func (c *Command) printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: authly-cli admin <subcommand> [args]\n\n")
	fmt.Fprintf(os.Stderr, "Subcommands:\n")
	fmt.Fprintf(os.Stderr, "  init-root   Initialize system service, roles and root user\n")
}

func (c *Command) runInitRoot(args []string) error {
	fs := flag.NewFlagSet("init-root", flag.ExitOnError)
	email := fs.String("email", "", "Root user email")
	password := fs.String("password", "", "Root user password")
	username := fs.String("username", "admin", "Root user username")
	domain := fs.String("domain", "", "Service domain (e.g., localhost:3000)")
	redirectURIs := fs.String("redirect-uris", "", "Comma-separated redirect URIs")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *email == "" || *password == "" {
		return fmt.Errorf("email and password are required")
	}

	// Load config
	envConfig := config.LoadEnv()
	cfg, err := config.Load(envConfig.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Connect to database
	if err := database.ConnectDB(cfg); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations
	if err := migrations.RunMigrations(cfg); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	serviceRepo := svc.NewRepository(database.DB)
	systemServiceID, _ := uuid.Parse(svc.DefaultAuthlyServiceID)

	var uris pq.StringArray
	if *redirectURIs != "" {
		uris = strings.Split(*redirectURIs, ",")
		for i := range uris {
			uris[i] = strings.TrimSpace(uris[i])
		}
	}

	systemService, err := serviceRepo.FindByID(svc.DefaultAuthlyServiceID)
	if err != nil {
		slog.Info("Creating system service...")
		systemService = &svc.Service{
			Name:         "Authly Dashboard",
			Description:  "System management dashboard",
			ClientID:     svc.DefaultAuthlyClientID,
			ClientSecret: "",
			AllowedScopes: []string{
				"openid",
				"profile",
				"email",
			},
			Active:       true,
			IsSystem:     true,
			Domain:       *domain,
			RedirectURIs: uris,
		}
		systemService.ID = systemServiceID
		if err := serviceRepo.Create(systemService); err != nil {
			return fmt.Errorf("failed to create system service: %w", err)
		}
	} else {
		slog.Info("System service already exists, updating configuration if provided...")

		updates := false
		if *domain != "" && systemService.Domain != *domain {
			systemService.Domain = *domain
			updates = true
		}
		if len(uris) > 0 {
			systemService.RedirectURIs = uris
			updates = true
		}

		if updates {
			if err := database.DB.Model(systemService).Updates(map[string]any{
				"domain":         systemService.Domain,
				"redirect_uris":  systemService.RedirectURIs,
				"allowed_scopes": systemService.AllowedScopes,
			}).Error; err != nil {
				return fmt.Errorf("failed to update system service: %w", err)
			}
			slog.Info("System service updated")
		}
	}

	permRepo := permission.NewRepository(database.DB)

	// Define system permissions
	systemPerms := []struct {
		Bit  uint8
		Name string
	}{
		{permission.BitRead, permission.PermRead},
		{permission.BitWrite, permission.PermWrite},
		{permission.BitDelete, permission.PermDelete},
		{permission.BitAdmin, permission.PermAdmin},
		{permission.BitManageServices, permission.PermManageServices},
		{permission.BitManagePermissions, permission.PermManagePermissions},
		{permission.BitManageUsers, permission.PermManageUsers},
		{permission.BitManageRoles, permission.PermManageRoles},
		{permission.BitSystemAdmin, permission.PermSystemAdmin},
	}

	var fullBitmask uint64
	for _, p := range systemPerms {
		fullBitmask |= (1 << p.Bit)

		existingPerms, _ := permRepo.FindPermissionsByServiceIDAndResource(systemService.ID.String(), nil)
		found := false
		for _, ep := range existingPerms {
			if ep.Bit == p.Bit {
				found = true
				break
			}
		}

		if !found {
			slog.Info("Creating permission", "name", p.Name)
			newPerm := &permission.Permission{
				ServiceID: systemService.ID,
				Bit:       p.Bit,
				Name:      p.Name,
				Active:    true,
			}
			if err := permRepo.CreatePermission(newPerm); err != nil {
				return fmt.Errorf("failed to create permission %s: %w", p.Name, err)
			}
		}
	}

	roleRepo := role.NewRepository(database.DB)
	roleService := role.NewService(database.DB, roleRepo, permRepo)

	superAdminRole, err := roleRepo.FindByName(systemService.ID.String(), "Super Admin")
	if err != nil {
		slog.Info("Creating Super Admin role...")
		superAdminRole = &role.Role{
			ServiceID:   systemService.ID,
			Name:        "Super Admin",
			Description: "Full system access",
			Bitmask:     fullBitmask,
			IsDefault:   false,
			Priority:    100,
		}
		if err := roleService.CreateRole(superAdminRole); err != nil {
			return fmt.Errorf("failed to create Super Admin role: %w", err)
		}
	} else {
		// Update bitmask in case it changed
		if superAdminRole.Bitmask != fullBitmask {
			superAdminRole.Bitmask = fullBitmask
			if err := roleService.UpdateRole(superAdminRole); err != nil {
				return fmt.Errorf("failed to update Super Admin role bitmask: %w", err)
			}
		}
	}

	userRepo := user.NewRepository(database.DB)
	rootUser, err := userRepo.FindByEmail(*email)
	if err != nil {
		slog.Info("Creating root user...")
		hashedPassword, err := user.HashPassword(*password)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		rootUser = &user.User{
			Username: *username,
			Email:    *email,
			Password: hashedPassword,
			IsActive: true,
		}
		if err := userRepo.Create(rootUser); err != nil {
			return fmt.Errorf("failed to create root user: %w", err)
		}
	} else {
		slog.Info("Root user already exists")
	}

	slog.Info("Assigning Super Admin role to root user...")
	if err := roleService.AssignRole(rootUser.ID.String(), superAdminRole.ID.String()); err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	slog.Info("System initialized successfully!", "admin", *email)
	return nil
}
