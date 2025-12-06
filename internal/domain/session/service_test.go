package session

import (
	"errors"
	"testing"
	"time"

	"github.com/Anvoria/authly/internal/domain/user"
	"github.com/Anvoria/authly/internal/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db := utils.SetupTestDB(t, &user.User{}, &Session{})
	db.Exec("DELETE FROM sessions")
	db.Exec("DELETE FROM users")
	return db
}

func createTestUser(t *testing.T, db *gorm.DB, userID uuid.UUID) *user.User {
	testUser := &user.User{
		Username:  "testuser_" + userID.String()[:8],
		FirstName: "Test",
		LastName:  "User",
		Email:     "test_" + userID.String()[:8] + "@example.com",
		Password:  "hashedpassword",
		IsActive:  true,
	}
	// Use GORM to create with specific ID
	if err := db.Table("users").Create(map[string]interface{}{
		"id":         userID,
		"username":   testUser.Username,
		"first_name": testUser.FirstName,
		"last_name":  testUser.LastName,
		"email":      testUser.Email,
		"password":   testUser.Password,
		"is_active":  testUser.IsActive,
	}).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	// Fetch the created user to return proper struct
	userRepo := user.NewRepository(db)
	createdUser, err := userRepo.FindByID(userID.String())
	if err != nil {
		t.Fatalf("Failed to find created user: %v", err)
	}
	return createdUser
}

func TestService_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	userID := uuid.New()
	createTestUser(t, db, userID)
	userAgent := "Mozilla/5.0"
	ip := "192.168.1.1"
	ttl := 24 * time.Hour

	sessionID, secret, err := service.Create(userID, userAgent, ip, ttl)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	if sessionID == uuid.Nil {
		t.Errorf("Create() sessionID should not be nil")
	}

	if secret == "" {
		t.Errorf("Create() secret should not be empty")
	}

	sess, err := repo.FindByID(sessionID)
	if err != nil {
		t.Fatalf("Create() session should exist in database: %v", err)
	}

	if sess.UserID != userID.String() {
		t.Errorf("Create() userID = %v, want %v", sess.UserID, userID.String())
	}

	if sess.UserAgent != userAgent {
		t.Errorf("Create() userAgent = %v, want %v", sess.UserAgent, userAgent)
	}

	if sess.IPAddress != ip {
		t.Errorf("Create() ipAddress = %v, want %v", sess.IPAddress, ip)
	}

	if sess.Revoked {
		t.Errorf("Create() revoked should be false")
	}

	// Verify secret hash matches
	expectedHash := hashSecret(secret)
	if sess.RefreshHash != expectedHash {
		t.Errorf("Create() refreshHash does not match secret")
	}
}

func TestService_Validate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	userID := uuid.New()
	createTestUser(t, db, userID)
	userAgent := "Mozilla/5.0"
	ip := "192.168.1.1"
	ttl := 24 * time.Hour

	sessionID, secret, err := service.Create(userID, userAgent, ip, ttl)
	if err != nil {
		t.Fatalf("Failed to create session for validation test: %v", err)
	}

	tests := []struct {
		name      string
		sessionID uuid.UUID
		secret    string
		wantErr   error
	}{
		{
			name:      "valid session",
			sessionID: sessionID,
			secret:    secret,
			wantErr:   nil,
		},
		{
			name:      "invalid secret",
			sessionID: sessionID,
			secret:    "wrong-secret",
			wantErr:   ErrInvalidSecret,
		},
		{
			name:      "non-existent session",
			sessionID: uuid.New(),
			secret:    secret,
			wantErr:   ErrInvalidSession,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sess, err := service.Validate(tt.sessionID, tt.secret)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Validate() expected error but got none")
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
				}
				if sess != nil {
					t.Errorf("Validate() expected nil session on error, got %v", sess)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
					return
				}
				if sess == nil {
					t.Errorf("Validate() expected session but got nil")
					return
				}
				if sess.ID != tt.sessionID {
					t.Errorf("Validate() sessionID = %v, want %v", sess.ID, tt.sessionID)
				}
			}
		})
	}
}

func TestService_Validate_ExpiredSession(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	userID := uuid.New()
	createTestUser(t, db, userID)
	userAgent := "Mozilla/5.0"
	ip := "192.168.1.1"
	ttl := -1 * time.Hour

	sessionID, secret, err := service.Create(userID, userAgent, ip, ttl)
	if err != nil {
		t.Fatalf("Failed to create expired session: %v", err)
	}

	sess, err := service.Validate(sessionID, secret)
	if err == nil {
		t.Errorf("Validate() expected error for expired session but got none")
	}
	if !errors.Is(err, ErrExpiredSession) {
		t.Errorf("Validate() error = %v, want %v", err, ErrExpiredSession)
	}
	if sess != nil {
		t.Errorf("Validate() expected nil session for expired session, got %v", sess)
	}
}

func TestService_Validate_RevokedSession(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	userID := uuid.New()
	createTestUser(t, db, userID)
	userAgent := "Mozilla/5.0"
	ip := "192.168.1.1"
	ttl := 24 * time.Hour

	sessionID, secret, err := service.Create(userID, userAgent, ip, ttl)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if err := service.Revoke(sessionID); err != nil {
		t.Fatalf("Failed to revoke session: %v", err)
	}

	sess, err := service.Validate(sessionID, secret)
	if err == nil {
		t.Errorf("Validate() expected error for revoked session but got none")
	}
	if !errors.Is(err, ErrInvalidSession) {
		t.Errorf("Validate() error = %v, want %v", err, ErrInvalidSession)
	}
	if sess != nil {
		t.Errorf("Validate() expected nil session for revoked session, got %v", sess)
	}
}

func TestService_Rotate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	userID := uuid.New()
	createTestUser(t, db, userID)
	userAgent := "Mozilla/5.0"
	ip := "192.168.1.1"
	ttl := 24 * time.Hour

	sessionID, oldSecret, err := service.Create(userID, userAgent, ip, ttl)
	if err != nil {
		t.Fatalf("Failed to create session for rotation test: %v", err)
	}

	newTTL := 48 * time.Hour
	newSecret, err := service.Rotate(sessionID, oldSecret, newTTL)
	if err != nil {
		t.Fatalf("Rotate() unexpected error: %v", err)
	}

	if newSecret == "" {
		t.Errorf("Rotate() newSecret should not be empty")
	}

	if newSecret == oldSecret {
		t.Errorf("Rotate() newSecret should be different from oldSecret")
	}

	_, err = service.Validate(sessionID, oldSecret)
	if err == nil {
		t.Errorf("Validate() old secret should not work after rotation")
	}
	if !errors.Is(err, ErrInvalidSecret) {
		t.Errorf("Validate() error = %v, want %v", err, ErrInvalidSecret)
	}

	sess, err := service.Validate(sessionID, newSecret)
	if err != nil {
		t.Errorf("Validate() new secret should work after rotation: %v", err)
	}
	if sess == nil {
		t.Errorf("Validate() expected session with new secret")
	}
}

func TestService_Rotate_InvalidSecret(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	userID := uuid.New()
	createTestUser(t, db, userID)
	userAgent := "Mozilla/5.0"
	ip := "192.168.1.1"
	ttl := 24 * time.Hour

	sessionID, _, err := service.Create(userID, userAgent, ip, ttl)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	newSecret, err := service.Rotate(sessionID, "wrong-secret", ttl)
	if err == nil {
		t.Errorf("Rotate() expected error for invalid secret but got none")
	}
	if !errors.Is(err, ErrInvalidSecret) {
		t.Errorf("Rotate() error = %v, want %v", err, ErrInvalidSecret)
	}
	if newSecret != "" {
		t.Errorf("Rotate() newSecret should be empty on error")
	}
}

func TestService_Rotate_ReplayDetection(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	userID := uuid.New()
	createTestUser(t, db, userID)
	userAgent := "Mozilla/5.0"
	ip := "192.168.1.1"
	ttl := 24 * time.Hour

	sessionID, oldSecret, err := service.Create(userID, userAgent, ip, ttl)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	newSecret1, err := service.Rotate(sessionID, oldSecret, ttl)
	if err != nil {
		t.Fatalf("First rotation should succeed: %v", err)
	}

	newSecret2, err := service.Rotate(sessionID, oldSecret, ttl)
	if err == nil {
		t.Errorf("Rotate() expected error for old secret but got none")
	}
	if !errors.Is(err, ErrInvalidSecret) {
		t.Errorf("Rotate() error = %v, want %v", err, ErrInvalidSecret)
	}
	if newSecret2 != "" {
		t.Errorf("Rotate() newSecret should be empty on error")
	}

	newSecret3, err := service.Rotate(sessionID, newSecret1, ttl)
	if err != nil {
		t.Errorf("Rotate() with new secret should work: %v", err)
	}
	if newSecret3 == "" {
		t.Errorf("Rotate() newSecret should not be empty")
	}
}

func TestService_Revoke(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	userID := uuid.New()
	createTestUser(t, db, userID)
	userAgent := "Mozilla/5.0"
	ip := "192.168.1.1"
	ttl := 24 * time.Hour

	sessionID, secret, err := service.Create(userID, userAgent, ip, ttl)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	_, err = service.Validate(sessionID, secret)
	if err != nil {
		t.Fatalf("Session should be valid before revoking: %v", err)
	}

	err = service.Revoke(sessionID)
	if err != nil {
		t.Fatalf("Revoke() unexpected error: %v", err)
	}

	_, err = service.Validate(sessionID, secret)
	if err == nil {
		t.Errorf("Validate() expected error for revoked session but got none")
	}
	if !errors.Is(err, ErrInvalidSession) {
		t.Errorf("Validate() error = %v, want %v", err, ErrInvalidSession)
	}
}

func TestService_Revoke_NonExistentSession(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	nonExistentID := uuid.New()

	err := service.Revoke(nonExistentID)
	if err != nil {
		t.Errorf("Revoke() should not error for non-existent session: %v", err)
	}
}
