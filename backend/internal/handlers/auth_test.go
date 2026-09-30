package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/shopverse/backend/internal/database"
	"github.com/shopverse/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbUser := getTestEnv("TEST_DB_USER", "shopverse")
	dbPassword := getTestEnv("TEST_DB_PASSWORD", "shopverse123")
	dbHost := getTestEnv("TEST_DB_HOST", "localhost")
	dbPort := getTestEnv("TEST_DB_PORT", "3306")
	dbName := getTestEnv("TEST_DB_NAME", "shopverse_test")

	// Connect to MySQL server first.
	serverDSN := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser,
		dbPassword,
		dbHost,
		dbPort,
	)

	serverDB, err := gorm.Open(mysql.Open(serverDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to MySQL: %v", err)
	}

	// Create a separate database for tests.
	err = serverDB.Exec(
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", dbName),
	).Error
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// Connect to the test database.
	testDSN := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser,
		dbPassword,
		dbHost,
		dbPort,
		dbName,
	)

	testDB, err := gorm.Open(mysql.Open(testDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// Create the users table.
	err = testDB.AutoMigrate(&models.User{})
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	// Login() uses database.DB, so point the application's
	// global database connection to our test database.
	database.DB = testDB

	return testDB
}

func getTestEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return fallback
}

func createTestUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()

	password := "password123"

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := models.User{
		Name:         "Test User",
		Email:        "test@example.com",
		PasswordHash: string(passwordHash),
	}

	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return user
}

func TestLogin_Success(t *testing.T) {
	db := setupTestDB(t)

	// Remove users from previous test runs.
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.User{})

	user := createTestUser(t, db)

	app := fiber.New()
	app.Post("/login", Login)

	requestBody := map[string]string{
		"email":    user.Email,
		"password": "password123",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("failed to create request body: %v", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"/login",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	defer resp.Body.Close()

	// Login should return HTTP 200.
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	// Decode response.
	var response models.AuthResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// JWT token should exist.
	if response.Token == "" {
		t.Error("expected JWT token, got empty token")
	}

	// Returned user should be correct.
	if response.User.Email != user.Email {
		t.Errorf(
			"expected email %q, got %q",
			user.Email,
			response.User.Email,
		)
	}

	if response.User.Name != user.Name {
		t.Errorf(
			"expected name %q, got %q",
			user.Name,
			response.User.Name,
		)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	db := setupTestDB(t)

	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.User{})

	user := createTestUser(t, db)

	app := fiber.New()
	app.Post("/login", Login)

	requestBody := map[string]string{
		"email":    user.Email,
		"password": "wrong-password",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("failed to create request body: %v", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"/login",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	defer resp.Body.Close()

	// Wrong password should return 401.
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf(
			"expected status 401, got %d",
			resp.StatusCode,
		)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	db := setupTestDB(t)

	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.User{})

	app := fiber.New()
	app.Post("/login", Login)

	requestBody := map[string]string{
		"email":    "doesnotexist@example.com",
		"password": "password123",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("failed to create request body: %v", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"/login",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	defer resp.Body.Close()

	// Unknown user should return 401.
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf(
			"expected status 401, got %d",
			resp.StatusCode,
		)
	}
}

func TestLogin_MissingCredentials(t *testing.T) {
	setupTestDB(t)

	app := fiber.New()
	app.Post("/login", Login)

	requestBody := map[string]string{
		"email":    "",
		"password": "",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("failed to create request body: %v", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"/login",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	defer resp.Body.Close()

	// Missing credentials should return 400.
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"expected status 400, got %d",
			resp.StatusCode,
		)
	}
}