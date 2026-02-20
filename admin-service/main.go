// =============================================================================
// Civic Connect – Admin Service  (Go + Gin + GORM)
// =============================================================================
// Connects to: PostgreSQL (admin_db), RabbitMQ, Redis
// Port: 8081 (HTTP)
//
// Domains: Authentication, User Management, Government Entities,
//          Officials, Follow System, Departments, Admin Accounts,
//          Notifications, Government Settings, Article Categories
//
// Role Hierarchy:
//   SuperAdmin → Creates municipalities, managers, other super_admins
//   Manager    → Creates departments, dept_managers, manages complaints/articles/posts
//   DeptManager → Manages own department complaints/articles/posts
// =============================================================================

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ── Models ──────────────────────────────────────────────────────────────────

type User struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Name          string    `gorm:"not null" json:"name"`
	Email         string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash  string    `gorm:"column:password_hash;not null" json:"-"`
	Role          string    `gorm:"not null;default:public" json:"role"`
	AadharNo      string    `gorm:"uniqueIndex;not null" json:"aadhar_no"`
	ProofDocument string    `json:"proof_document,omitempty"`
	Location      string    `json:"location,omitempty"`
	ProfilePic    string    `json:"profile_pic,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Notification struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	UserID    uint            `gorm:"index;not null" json:"user_id"`
	Title     string          `gorm:"not null" json:"title"`
	Body      string          `json:"body"`
	Data      json.RawMessage `gorm:"type:jsonb" json:"data,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type Government struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Type      string    `gorm:"not null" json:"type"`
	State     string    `json:"state,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type LocalGovernment struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"not null" json:"name"`
	Jurisdiction string    `json:"jurisdiction,omitempty"`
	State        string    `json:"state,omitempty"`
	Email        string    `json:"email,omitempty"`
	Phone        string    `json:"phone,omitempty"`
	LogoURL      string    `json:"logo_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Department struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"not null" json:"name"`
	Email        string    `json:"email,omitempty"`
	Phone        string    `json:"phone,omitempty"`
	Services     string    `gorm:"type:text" json:"services,omitempty"`
	LogoURL      string    `json:"logo_url,omitempty"`
	GovernmentID uint      `gorm:"index;not null" json:"government_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type GovernmentAdmin struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	GovernmentID uint      `gorm:"index;not null" json:"government_id"`
	Name         string    `gorm:"not null" json:"name"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"column:password_hash;not null" json:"-"`
	Role         string    `gorm:"not null;default:dept_manager" json:"role"`
	DepartmentID *uint     `json:"department_id,omitempty"`
	LastLogin    time.Time `json:"last_login,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type ArticleCategory struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"not null" json:"name"`
	GovernmentID uint   `gorm:"index;not null" json:"government_id"`
}

type GovernmentOfficial struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	UserID     uint   `gorm:"uniqueIndex;not null" json:"user_id"`
	Department string `json:"department,omitempty"`
	Position   string `json:"position,omitempty"`
	User       User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type Follower struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"not null" json:"user_id"`
	OfficialID uint      `gorm:"not null" json:"official_id"`
	FollowedAt time.Time `gorm:"autoCreateTime" json:"followed_at"`
}

type GovernmentFollow struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"not null" json:"user_id"`
	GovernmentID uint      `gorm:"not null" json:"government_id"`
	FollowedAt   time.Time `gorm:"autoCreateTime" json:"followed_at"`
}

// ── Globals ─────────────────────────────────────────────────────────────────

var (
	db       *gorm.DB
	rdb      *redis.Client
	amqpConn *amqp.Connection
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ── Init helpers ────────────────────────────────────────────────────────────

func connectPostgres() {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		env("DB_HOST", "localhost"),
		env("DB_PORT", "5432"),
		env("DB_USER", "civic_admin"),
		env("DB_PASSWORD", "civic_secret_2026"),
		env("DB_NAME", "admin_db"),
	)
	var err error
	for i := 0; i < 30; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("[admin-service] Waiting for PostgreSQL... (%d/30)", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("[admin-service] PostgreSQL connection failed: %v", err)
	}
	log.Println("[admin-service] ✅ PostgreSQL Connected Successfully")

	db.AutoMigrate(
		&User{}, &Notification{},
		&Government{}, &LocalGovernment{},
		&Department{}, &GovernmentAdmin{},
		&GovernmentOfficial{}, &Follower{},
		&GovernmentFollow{}, &ArticleCategory{},
	)

	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_followers_unique ON followers(user_id, official_id)")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_gov_follow_unique ON government_follows(user_id, government_id)")
	db.Exec("UPDATE government_admins SET role = 'dept_manager' WHERE role = 'department_admin'")
}

func connectRabbitMQ() {
	url := env("RABBITMQ_URL",
		fmt.Sprintf("amqp://%s:%s@localhost:5672/",
			env("RABBITMQ_USER", "civic_rabbit"),
			env("RABBITMQ_PASS", "rabbit_secret_2026")))
	var err error
	for i := 0; i < 30; i++ {
		amqpConn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("[admin-service] Waiting for RabbitMQ... (%d/30)", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("[admin-service] RabbitMQ connection failed: %v", err)
	}
	log.Println("[admin-service] ✅ RabbitMQ Connected Successfully")
}

func connectRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     env("REDIS_ADDR", "localhost:6379"),
		Password: env("REDIS_PASSWORD", "redis_secret_2026"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for i := 0; i < 30; i++ {
		if err := rdb.Ping(ctx).Err(); err == nil {
			break
		}
		log.Printf("[admin-service] Waiting for Redis... (%d/30)", i+1)
		time.Sleep(2 * time.Second)
	}
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("[admin-service] Redis connection failed: %v", err)
	}
	log.Println("[admin-service] ✅ Redis Connected Successfully")
}

// ── JWT ─────────────────────────────────────────────────────────────────────

var jwtSecret = []byte(env("JWT_SECRET", "civic_jwt_secret_2026"))

func generateToken(userID uint, email, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
}

func generateAdminToken(admin GovernmentAdmin) (string, error) {
	claims := jwt.MapClaims{
		"admin_id":      admin.ID,
		"government_id": admin.GovernmentID,
		"email":         admin.Email,
		"role":          admin.Role,
		"department_id": admin.DepartmentID,
		"exp":           time.Now().Add(8 * time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization")
		if len(tokenStr) < 8 || tokenStr[:7] != "Bearer " {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		token, err := jwt.Parse(tokenStr[7:], func(t *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		claims := token.Claims.(jwt.MapClaims)
		c.Set("user_id", claims["user_id"])
		c.Set("role", claims["role"])
		c.Next()
	}
}

func adminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization")
		if len(tokenStr) < 8 || tokenStr[:7] != "Bearer " {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		token, err := jwt.Parse(tokenStr[7:], func(t *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		claims := token.Claims.(jwt.MapClaims)
		if _, ok := claims["admin_id"]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not an admin token"})
			return
		}
		c.Set("admin_id", claims["admin_id"])
		c.Set("government_id", claims["government_id"])
		c.Set("admin_role", claims["role"])
		c.Set("department_id", claims["department_id"])
		c.Next()
	}
}

func roleRequired(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		for _, r := range roles {
			if r == role {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}

func adminRoleRequired(roles ...string) gin.HandlerFunc {
	// Role hierarchy: super_admin > manager > dept_manager
	hierarchy := map[string]int{"super_admin": 3, "manager": 2, "dept_manager": 1}
	return func(c *gin.Context) {
		role, _ := c.Get("admin_role")
		roleStr, _ := role.(string)
		callerLevel := hierarchy[roleStr]
		for _, r := range roles {
			if callerLevel >= hierarchy[r] {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}

func getUserID(c *gin.Context) uint {
	if v, ok := c.Get("user_id"); ok {
		switch id := v.(type) {
		case float64:
			return uint(id)
		case uint:
			return id
		}
	}
	return 0
}

func getAdminID(c *gin.Context) uint {
	if v, ok := c.Get("admin_id"); ok {
		switch id := v.(type) {
		case float64:
			return uint(id)
		case uint:
			return id
		}
	}
	return 0
}

func getGovID(c *gin.Context) uint {
	if v, ok := c.Get("government_id"); ok {
		switch id := v.(type) {
		case float64:
			return uint(id)
		case uint:
			return id
		}
	}
	return 0
}

func getDeptID(c *gin.Context) *uint {
	if v, ok := c.Get("department_id"); ok {
		switch id := v.(type) {
		case float64:
			u := uint(id)
			if u > 0 {
				return &u
			}
		case *uint:
			return id
		}
	}
	return nil
}

// ── Auth Handlers ───────────────────────────────────────────────────────────

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "admin-service"})
}

func registerHandler(c *gin.Context) {
	var body struct {
		Name          string `json:"name" binding:"required"`
		Email         string `json:"email" binding:"required,email"`
		Password      string `json:"password" binding:"required,min=6"`
		AadharNo      string `json:"aadhar_no" binding:"required"`
		Role          string `json:"role"`
		ProofDocument string `json:"proof_document"`
		Location      string `json:"location"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := "public"
	if body.Role == "government" {
		role = "government"
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	user := User{
		Name: body.Name, Email: body.Email, PasswordHash: string(hash),
		Role: role, AadharNo: body.AadharNo,
		ProofDocument: body.ProofDocument, Location: body.Location,
	}
	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email or aadhar already registered"})
		return
	}
	token, _ := generateToken(user.ID, user.Email, user.Role)
	c.JSON(http.StatusCreated, gin.H{"token": token, "user": user})
}

func loginHandler(c *gin.Context) {
	var body struct {
		AadharNo string `json:"aadhar_no" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var user User
	if err := db.Where("aadhar_no = ?", body.AadharNo).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	token, _ := generateToken(user.ID, user.Email, user.Role)
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func checkAuthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"authenticated": true, "user_id": c.GetFloat64("user_id")})
}

// ── Profile Handlers ────────────────────────────────────────────────────────

func getProfileHandler(c *gin.Context) {
	var user User
	if err := db.First(&user, getUserID(c)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func updateProfileHandler(c *gin.Context) {
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if body.Name != "" {
		updates["name"] = body.Name
	}
	if body.Email != "" {
		updates["email"] = body.Email
	}
	db.Model(&User{}).Where("id = ?", getUserID(c)).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}

func changePasswordHandler(c *gin.Context) {
	var body struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var user User
	if err := db.First(&user, getUserID(c)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect current password"})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	db.Model(&user).Update("password_hash", string(hash))
	c.JSON(http.StatusOK, gin.H{"message": "password changed"})
}

// ── Notification Handlers ───────────────────────────────────────────────────

func getNotificationsHandler(c *gin.Context) {
	uid := getUserID(c)
	var notifications []Notification
	db.Where("user_id = ?", uid).Order("created_at DESC").Find(&notifications)
	db.Where("user_id = ?", uid).Delete(&Notification{})
	c.JSON(http.StatusOK, notifications)
}

// ── Government Entity Handlers ──────────────────────────────────────────────

func listGovernmentsHandler(c *gin.Context) {
	var govs []Government
	db.Find(&govs)
	c.JSON(http.StatusOK, govs)
}

func getGovernmentHandler(c *gin.Context) {
	var gov Government
	if err := db.First(&gov, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "government not found"})
		return
	}
	c.JSON(http.StatusOK, gov)
}

func createGovernmentHandler(c *gin.Context) {
	var gov Government
	if err := c.ShouldBindJSON(&gov); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db.Create(&gov)
	c.JSON(http.StatusCreated, gov)
}

func updateGovernmentHandler(c *gin.Context) {
	var gov Government
	if err := db.First(&gov, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "government not found"})
		return
	}
	c.ShouldBindJSON(&gov)
	db.Save(&gov)
	c.JSON(http.StatusOK, gov)
}

func deleteGovernmentHandler(c *gin.Context) {
	db.Delete(&Government{}, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ── Local Government Handlers ───────────────────────────────────────────────

func listLocalGovernmentsHandler(c *gin.Context) {
	var govs []LocalGovernment
	db.Find(&govs)
	c.JSON(http.StatusOK, govs)
}

func getLocalGovernmentHandler(c *gin.Context) {
	var gov LocalGovernment
	if err := db.First(&gov, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gov)
}

func updateLocalGovernmentHandler(c *gin.Context) {
	var gov LocalGovernment
	if err := db.First(&gov, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.ShouldBindJSON(&gov)
	db.Save(&gov)
	c.JSON(http.StatusOK, gov)
}

// ── Municipality CRUD (SuperAdmin only) ─────────────────────────────────────

func createMunicipalityHandler(c *gin.Context) {
	var body struct {
		Name         string `json:"name" binding:"required"`
		Jurisdiction string `json:"jurisdiction"`
		State        string `json:"state"`
		Email        string `json:"email"`
		Phone        string `json:"phone"`
		LogoURL      string `json:"logo_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gov := LocalGovernment{
		Name: body.Name, Jurisdiction: body.Jurisdiction, State: body.State,
		Email: body.Email, Phone: body.Phone, LogoURL: body.LogoURL,
	}
	if err := db.Create(&gov).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "municipality already exists"})
		return
	}
	c.JSON(http.StatusCreated, gov)
}

func listMunicipalitiesHandler(c *gin.Context) {
	var govs []LocalGovernment
	db.Order("name ASC").Find(&govs)
	type MuniWithCounts struct {
		LocalGovernment
		ManagerCount int64 `json:"manager_count"`
		DeptCount    int64 `json:"department_count"`
	}
	var result []MuniWithCounts
	for _, g := range govs {
		var mgrCount, deptCount int64
		db.Model(&GovernmentAdmin{}).Where("government_id = ? AND role = ?", g.ID, "manager").Count(&mgrCount)
		db.Model(&Department{}).Where("government_id = ?", g.ID).Count(&deptCount)
		result = append(result, MuniWithCounts{LocalGovernment: g, ManagerCount: mgrCount, DeptCount: deptCount})
	}
	c.JSON(http.StatusOK, result)
}

func updateMunicipalityHandler(c *gin.Context) {
	var gov LocalGovernment
	if err := db.First(&gov, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "municipality not found"})
		return
	}
	var body struct {
		Name         string `json:"name"`
		Jurisdiction string `json:"jurisdiction"`
		State        string `json:"state"`
		Email        string `json:"email"`
		Phone        string `json:"phone"`
		LogoURL      string `json:"logo_url"`
	}
	c.ShouldBindJSON(&body)
	if body.Name != "" {
		gov.Name = body.Name
	}
	if body.Jurisdiction != "" {
		gov.Jurisdiction = body.Jurisdiction
	}
	if body.State != "" {
		gov.State = body.State
	}
	if body.Email != "" {
		gov.Email = body.Email
	}
	if body.Phone != "" {
		gov.Phone = body.Phone
	}
	if body.LogoURL != "" {
		gov.LogoURL = body.LogoURL
	}
	db.Save(&gov)
	c.JSON(http.StatusOK, gov)
}

func deleteMunicipalityHandler(c *gin.Context) {
	id := c.Param("id")
	var deptCount int64
	db.Model(&Department{}).Where("government_id = ?", id).Count(&deptCount)
	if deptCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "cannot delete municipality with existing departments"})
		return
	}
	db.Delete(&LocalGovernment{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ── Department Handlers ─────────────────────────────────────────────────────

func listDepartmentsHandler(c *gin.Context) {
	govID := c.Query("government_id")
	role, _ := c.Get("admin_role")
	var depts []Department
	q := db
	if govID != "" {
		q = q.Where("government_id = ?", govID)
	} else if role != "super_admin" {
		q = q.Where("government_id = ?", getGovID(c))
	}
	if role == "dept_manager" {
		deptID := getDeptID(c)
		if deptID != nil {
			q = q.Where("id = ?", *deptID)
		}
	}
	q.Order("name ASC").Find(&depts)

	type DeptWithManagers struct {
		Department
		ManagerCount int64  `json:"manager_count"`
		GovName      string `json:"government_name,omitempty"`
	}
	var result []DeptWithManagers
	for _, d := range depts {
		var mgrCount int64
		db.Model(&GovernmentAdmin{}).Where("department_id = ? AND role = ?", d.ID, "dept_manager").Count(&mgrCount)
		item := DeptWithManagers{Department: d, ManagerCount: mgrCount}
		var gov LocalGovernment
		if db.First(&gov, d.GovernmentID).Error == nil {
			item.GovName = gov.Name
		}
		result = append(result, item)
	}
	c.JSON(http.StatusOK, result)
}

func getDepartmentHandler(c *gin.Context) {
	var dept Department
	if err := db.First(&dept, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "department not found"})
		return
	}
	c.JSON(http.StatusOK, dept)
}

func createDepartmentHandler(c *gin.Context) {
	govID := getGovID(c)
	var body struct {
		Name     string `json:"name" binding:"required"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Services string `json:"services"`
		LogoURL  string `json:"logo_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dept := Department{
		Name: body.Name, Email: body.Email, Phone: body.Phone,
		Services: body.Services, LogoURL: body.LogoURL, GovernmentID: govID,
	}
	if err := db.Create(&dept).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "department already exists"})
		return
	}
	c.JSON(http.StatusCreated, dept)
}

func updateDepartmentHandler(c *gin.Context) {
	var dept Department
	if err := db.First(&dept, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "department not found"})
		return
	}
	if dept.GovernmentID != getGovID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify departments in other municipalities"})
		return
	}
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Services string `json:"services"`
		LogoURL  string `json:"logo_url"`
	}
	c.ShouldBindJSON(&body)
	if body.Name != "" {
		dept.Name = body.Name
	}
	if body.Email != "" {
		dept.Email = body.Email
	}
	if body.Phone != "" {
		dept.Phone = body.Phone
	}
	if body.Services != "" {
		dept.Services = body.Services
	}
	if body.LogoURL != "" {
		dept.LogoURL = body.LogoURL
	}
	db.Save(&dept)
	c.JSON(http.StatusOK, dept)
}

func deleteDepartmentHandler(c *gin.Context) {
	var dept Department
	if err := db.First(&dept, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "department not found"})
		return
	}
	if dept.GovernmentID != getGovID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete departments in other municipalities"})
		return
	}
	db.Delete(&dept)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ── Government Official Handlers ────────────────────────────────────────────

func listOfficialsHandler(c *gin.Context) {
	var officials []GovernmentOfficial
	db.Preload("User").Find(&officials)
	c.JSON(http.StatusOK, officials)
}

func getOfficialHandler(c *gin.Context) {
	id := c.Param("id")
	var official GovernmentOfficial
	if err := db.Preload("User").First(&official, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "official not found"})
		return
	}
	var followerCount int64
	db.Model(&Follower{}).Where("official_id = ?", id).Count(&followerCount)
	c.JSON(http.StatusOK, gin.H{"official": official, "follower_count": followerCount})
}

func createOfficialHandler(c *gin.Context) {
	var official GovernmentOfficial
	if err := c.ShouldBindJSON(&official); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db.Create(&official)
	c.JSON(http.StatusCreated, official)
}

func updateOfficialHandler(c *gin.Context) {
	id := c.Param("id")
	var official GovernmentOfficial
	if err := db.First(&official, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "official not found"})
		return
	}
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db.Model(&official).Updates(body)
	c.JSON(http.StatusOK, official)
}

func deleteOfficialHandler(c *gin.Context) {
	id := c.Param("id")
	if err := db.Delete(&GovernmentOfficial{}, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "official not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ── Follow System Handlers ──────────────────────────────────────────────────

func followOfficialHandler(c *gin.Context) {
	officialID, _ := strconv.Atoi(c.Param("official_id"))
	follow := Follower{UserID: getUserID(c), OfficialID: uint(officialID)}
	if err := db.Create(&follow).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "already following"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "followed"})
}

func unfollowOfficialHandler(c *gin.Context) {
	officialID := c.Param("official_id")
	db.Where("user_id = ? AND official_id = ?", getUserID(c), officialID).Delete(&Follower{})
	c.JSON(http.StatusOK, gin.H{"message": "unfollowed"})
}

func checkFollowHandler(c *gin.Context) {
	officialID := c.Param("official_id")
	var count int64
	db.Model(&Follower{}).Where("user_id = ? AND official_id = ?", getUserID(c), officialID).Count(&count)
	c.JSON(http.StatusOK, gin.H{"following": count > 0})
}

func getFollowersHandler(c *gin.Context) {
	officialID := c.Param("id")
	var followers []Follower
	db.Where("official_id = ?", officialID).Find(&followers)
	c.JSON(http.StatusOK, followers)
}

func getUserFollowingHandler(c *gin.Context) {
	userID := c.Param("user_id")
	var followers []Follower
	db.Where("user_id = ?", userID).Find(&followers)
	var officialIDs []uint
	for _, f := range followers {
		officialIDs = append(officialIDs, f.OfficialID)
	}
	var officials []GovernmentOfficial
	if len(officialIDs) > 0 {
		db.Preload("User").Where("id IN ?", officialIDs).Find(&officials)
	}
	c.JSON(http.StatusOK, officials)
}

// ── Government Admin Portal Handlers ────────────────────────────────────────

func adminLoginHandler(c *gin.Context) {
	var body struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var admin GovernmentAdmin
	if err := db.Where("email = ?", body.Email).First(&admin).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(body.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	db.Model(&admin).Update("last_login", time.Now())
	token, _ := generateAdminToken(admin)

	var gov LocalGovernment
	db.First(&gov, admin.GovernmentID)
	var deptName string
	if admin.DepartmentID != nil {
		var dept Department
		if db.First(&dept, *admin.DepartmentID).Error == nil {
			deptName = dept.Name
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token, "admin": admin,
		"government_name": gov.Name, "government_logo": gov.LogoURL,
		"department_name": deptName,
	})
}

func adminMeHandler(c *gin.Context) {
	adminID := getAdminID(c)
	var admin GovernmentAdmin
	if err := db.First(&admin, adminID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}
	var gov LocalGovernment
	db.First(&gov, admin.GovernmentID)
	var deptName string
	if admin.DepartmentID != nil {
		var dept Department
		if db.First(&dept, *admin.DepartmentID).Error == nil {
			deptName = dept.Name
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"id": admin.ID, "government_id": admin.GovernmentID,
		"name": admin.Name, "email": admin.Email, "role": admin.Role,
		"department_id": admin.DepartmentID, "department_name": deptName,
		"last_login": admin.LastLogin, "created_at": admin.CreatedAt,
		"government_name": gov.Name, "government_logo": gov.LogoURL,
	})
}

func dashboardHandler(c *gin.Context) {
	govID := getGovID(c)
	role, _ := c.Get("admin_role")
	deptID := getDeptID(c)
	ctx := context.Background()
	result := gin.H{"role": role}

	switch role {
	case "super_admin":
		var govCount, mgrCount, deptCount, userCount int64
		db.Model(&LocalGovernment{}).Count(&govCount)
		db.Model(&GovernmentAdmin{}).Where("role = ?", "manager").Count(&mgrCount)
		db.Model(&Department{}).Count(&deptCount)
		db.Model(&User{}).Count(&userCount)
		result["municipalities"] = govCount
		result["managers"] = mgrCount
		result["departments"] = deptCount
		result["users"] = userCount

	case "manager":
		var deptCount, deptMgrCount int64
		db.Model(&Department{}).Where("government_id = ?", govID).Count(&deptCount)
		db.Model(&GovernmentAdmin{}).Where("government_id = ? AND role = ?", govID, "dept_manager").Count(&deptMgrCount)
		result["departments"] = deptCount
		result["dept_managers"] = deptMgrCount
		pendingStr, _ := rdb.Get(ctx, fmt.Sprintf("dashboard:%v:pending", govID)).Result()
		inProgressStr, _ := rdb.Get(ctx, fmt.Sprintf("dashboard:%v:in_progress", govID)).Result()
		resolvedStr, _ := rdb.Get(ctx, fmt.Sprintf("dashboard:%v:resolved", govID)).Result()
		pending, _ := strconv.ParseInt(pendingStr, 10, 64)
		inProgress, _ := strconv.ParseInt(inProgressStr, 10, 64)
		resolved, _ := strconv.ParseInt(resolvedStr, 10, 64)
		result["pending_complaints"] = pending
		result["in_progress_complaints"] = inProgress
		result["resolved_complaints"] = resolved

	case "dept_manager":
		if deptID != nil {
			var dept Department
			if db.First(&dept, *deptID).Error == nil {
				result["department_name"] = dept.Name
			}
			result["department_id"] = *deptID
		}
		pendingStr, _ := rdb.Get(ctx, fmt.Sprintf("dashboard:%v:dept:%v:pending", govID, deptID)).Result()
		resolvedStr, _ := rdb.Get(ctx, fmt.Sprintf("dashboard:%v:dept:%v:resolved", govID, deptID)).Result()
		pending, _ := strconv.ParseInt(pendingStr, 10, 64)
		resolved, _ := strconv.ParseInt(resolvedStr, 10, 64)
		result["pending_complaints"] = pending
		result["resolved_complaints"] = resolved
	}
	c.JSON(http.StatusOK, result)
}

func listAdminsHandler(c *gin.Context) {
	govID := getGovID(c)
	role, _ := c.Get("admin_role")
	var admins []GovernmentAdmin
	switch role {
	case "super_admin":
		db.Where("role IN ?", []string{"manager", "super_admin"}).Order("role ASC, name ASC").Find(&admins)
	case "manager":
		db.Where("government_id = ? AND role = ?", govID, "dept_manager").Order("name ASC").Find(&admins)
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}
	type AdminWithDetails struct {
		GovernmentAdmin
		GovernmentName string `json:"government_name"`
		DepartmentName string `json:"department_name,omitempty"`
	}
	var result []AdminWithDetails
	for _, a := range admins {
		detail := AdminWithDetails{GovernmentAdmin: a}
		var gov LocalGovernment
		if db.First(&gov, a.GovernmentID).Error == nil {
			detail.GovernmentName = gov.Name
		}
		if a.DepartmentID != nil {
			var dept Department
			if db.First(&dept, *a.DepartmentID).Error == nil {
				detail.DepartmentName = dept.Name
			}
		}
		result = append(result, detail)
	}
	c.JSON(http.StatusOK, result)
}

func createAdminHandler(c *gin.Context) {
	callerRole, _ := c.Get("admin_role")
	callerGovID := getGovID(c)
	var body struct {
		Name         string `json:"name" binding:"required"`
		Email        string `json:"email" binding:"required,email"`
		Password     string `json:"password" binding:"required,min=6"`
		Role         string `json:"role" binding:"required"`
		GovernmentID uint   `json:"government_id"`
		DepartmentID *uint  `json:"department_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	validRoles := map[string]bool{"super_admin": true, "manager": true, "dept_manager": true}
	if !validRoles[body.Role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}
	switch callerRole {
	case "super_admin":
		if body.Role == "dept_manager" {
			c.JSON(http.StatusForbidden, gin.H{"error": "super admins cannot create department managers; managers should do this"})
			return
		}
		if body.Role == "manager" && body.GovernmentID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "government_id is required when creating a manager"})
			return
		}
	case "manager":
		if body.Role != "dept_manager" {
			c.JSON(http.StatusForbidden, gin.H{"error": "managers can only create department manager accounts"})
			return
		}
		if body.DepartmentID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "department_id is required for dept_manager role"})
			return
		}
		var dept Department
		if err := db.First(&dept, *body.DepartmentID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "department not found"})
			return
		}
		if dept.GovernmentID != callerGovID {
			c.JSON(http.StatusForbidden, gin.H{"error": "department does not belong to your municipality"})
			return
		}
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	govID := body.GovernmentID
	if callerRole.(string) == "manager" {
		govID = callerGovID
	}
	if callerRole.(string) == "super_admin" && body.Role == "super_admin" {
		govID = callerGovID
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	admin := GovernmentAdmin{
		GovernmentID: govID, Name: body.Name, Email: body.Email,
		PasswordHash: string(hash), Role: body.Role, DepartmentID: body.DepartmentID,
	}
	if err := db.Create(&admin).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "admin email already exists"})
		return
	}
	c.JSON(http.StatusCreated, admin)
}

func updateAdminHandler(c *gin.Context) {
	var admin GovernmentAdmin
	if err := db.First(&admin, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}
	var body struct {
		Name         string `json:"name"`
		Role         string `json:"role"`
		DepartmentID *uint  `json:"department_id"`
	}
	c.ShouldBindJSON(&body)
	if body.Name != "" {
		admin.Name = body.Name
	}
	if body.Role != "" {
		admin.Role = body.Role
	}
	if body.DepartmentID != nil {
		admin.DepartmentID = body.DepartmentID
	}
	db.Save(&admin)
	c.JSON(http.StatusOK, admin)
}

func deleteAdminHandler(c *gin.Context) {
	callerRole, _ := c.Get("admin_role")
	callerGovID := getGovID(c)
	var admin GovernmentAdmin
	if err := db.First(&admin, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}
	switch callerRole {
	case "super_admin":
		if admin.ID == getAdminID(c) {
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete yourself"})
			return
		}
	case "manager":
		if admin.Role != "dept_manager" {
			c.JSON(http.StatusForbidden, gin.H{"error": "managers can only delete department managers"})
			return
		}
		if admin.GovernmentID != callerGovID {
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete admins from other municipalities"})
			return
		}
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}
	db.Delete(&admin)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func adminChangePasswordHandler(c *gin.Context) {
	adminID := getAdminID(c)
	var body struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var admin GovernmentAdmin
	if err := db.First(&admin, adminID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(body.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect current password"})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	db.Model(&admin).Update("password_hash", string(hash))
	c.JSON(http.StatusOK, gin.H{"message": "password changed"})
}

// ── Article Category Handlers ───────────────────────────────────────────────

func listArticleCategoriesHandler(c *gin.Context) {
	govID := getGovID(c)
	var cats []ArticleCategory
	db.Where("government_id = ?", govID).Order("name ASC").Find(&cats)
	c.JSON(http.StatusOK, cats)
}

func createArticleCategoryHandler(c *gin.Context) {
	govID := getGovID(c)
	var body struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cat := ArticleCategory{Name: body.Name, GovernmentID: govID}
	if err := db.Create(&cat).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "category already exists"})
		return
	}
	c.JSON(http.StatusCreated, cat)
}

func deleteArticleCategoryHandler(c *gin.Context) {
	govID := getGovID(c)
	var cat ArticleCategory
	if err := db.First(&cat, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
		return
	}
	if cat.GovernmentID != govID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete categories from other municipalities"})
		return
	}
	db.Delete(&cat)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ── Session Verify ──────────────────────────────────────────────────────────

func sessionVerifyHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"valid": true, "user_id": c.GetFloat64("user_id"), "role": c.GetString("role"),
	})
}

// ── Government Follow Handlers ──────────────────────────────────────────────

func followGovernmentHandler(c *gin.Context) {
	govID, _ := strconv.Atoi(c.Param("gov_id"))
	var gov LocalGovernment
	if err := db.First(&gov, govID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "government not found"})
		return
	}
	follow := GovernmentFollow{UserID: getUserID(c), GovernmentID: uint(govID)}
	if err := db.Create(&follow).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "already following"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "followed"})
}

func unfollowGovernmentHandler(c *gin.Context) {
	govID := c.Param("gov_id")
	db.Where("user_id = ? AND government_id = ?", getUserID(c), govID).Delete(&GovernmentFollow{})
	c.JSON(http.StatusOK, gin.H{"message": "unfollowed"})
}

func checkGovernmentFollowHandler(c *gin.Context) {
	govID := c.Param("gov_id")
	var count int64
	db.Model(&GovernmentFollow{}).Where("user_id = ? AND government_id = ?", getUserID(c), govID).Count(&count)
	c.JSON(http.StatusOK, gin.H{"following": count > 0})
}

func getUserFollowedGovernmentsHandler(c *gin.Context) {
	userID := c.Param("user_id")
	var follows []GovernmentFollow
	db.Where("user_id = ?", userID).Find(&follows)
	var govIDs []uint
	for _, f := range follows {
		govIDs = append(govIDs, f.GovernmentID)
	}
	var governments []LocalGovernment
	if len(govIDs) > 0 {
		db.Where("id IN ?", govIDs).Find(&governments)
	}
	c.JSON(http.StatusOK, governments)
}

func searchLocalGovernmentsHandler(c *gin.Context) {
	q := c.Query("q")
	var govs []LocalGovernment
	if q != "" {
		db.Where("name ILIKE ? OR jurisdiction ILIKE ?", "%"+q+"%", "%"+q+"%").Find(&govs)
	} else {
		db.Find(&govs)
	}
	uid := getUserID(c)
	type GovWithMeta struct {
		LocalGovernment
		FollowerCount int  `json:"follower_count"`
		IsFollowing   bool `json:"is_following"`
	}
	var result []GovWithMeta
	for _, g := range govs {
		var count int64
		db.Model(&GovernmentFollow{}).Where("government_id = ?", g.ID).Count(&count)
		var followingCount int64
		if uid > 0 {
			db.Model(&GovernmentFollow{}).Where("user_id = ? AND government_id = ?", uid, g.ID).Count(&followingCount)
		}
		result = append(result, GovWithMeta{
			LocalGovernment: g, FollowerCount: int(count), IsFollowing: followingCount > 0,
		})
	}
	c.JSON(http.StatusOK, result)
}

// ── Seed Initial SuperAdmin ─────────────────────────────────────────────────

func seedSuperAdminHandler(c *gin.Context) {
	var count int64
	db.Model(&GovernmentAdmin{}).Where("role = ?", "super_admin").Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "super admin already exists"})
		return
	}
	var body struct {
		GovernmentName string `json:"government_name" binding:"required"`
		Jurisdiction   string `json:"jurisdiction"`
		Name           string `json:"name" binding:"required"`
		Email          string `json:"email" binding:"required,email"`
		Password       string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gov := LocalGovernment{Name: body.GovernmentName, Jurisdiction: body.Jurisdiction}
	db.Create(&gov)
	hash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	admin := GovernmentAdmin{
		GovernmentID: gov.ID, Name: body.Name, Email: body.Email,
		PasswordHash: string(hash), Role: "super_admin",
	}
	db.Create(&admin)
	token, _ := generateAdminToken(admin)
	c.JSON(http.StatusCreated, gin.H{"token": token, "admin": admin, "government": gov})
}

// ── User Listing with search, filter, pagination ────────────────────────────

func listUsersHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	searchQ := c.Query("search")
	location := c.Query("location")
	sortBy := c.DefaultQuery("sort", "created_at")
	order := c.DefaultQuery("order", "desc")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	validSortFields := map[string]bool{"created_at": true, "name": true, "email": true, "location": true}
	if !validSortFields[sortBy] {
		sortBy = "created_at"
	}
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	query := db.Model(&User{})
	if searchQ != "" {
		searchQ = strings.TrimSpace(searchQ)
		query = query.Where("name ILIKE ? OR email ILIKE ? OR RIGHT(aadhar_no, 4) = ?",
			"%"+searchQ+"%", "%"+searchQ+"%", searchQ)
	}
	if location != "" {
		query = query.Where("location ILIKE ?", "%"+location+"%")
	}

	var total int64
	query.Count(&total)
	var users []User
	offset := (page - 1) * limit
	query.Order(fmt.Sprintf("%s %s", sortBy, order)).Offset(offset).Limit(limit).Find(&users)
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"users": users, "total": total, "page": page,
		"limit": limit, "total_pages": totalPages,
	})
}

// ── Admin Government Info ───────────────────────────────────────────────────

func adminGetGovernmentHandler(c *gin.Context) {
	govID := getGovID(c)
	var gov LocalGovernment
	if err := db.First(&gov, govID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "government not found"})
		return
	}
	var followerCount int64
	db.Model(&GovernmentFollow{}).Where("government_id = ?", govID).Count(&followerCount)
	c.JSON(http.StatusOK, gin.H{
		"id": gov.ID, "name": gov.Name, "jurisdiction": gov.Jurisdiction,
		"state": gov.State, "email": gov.Email, "phone": gov.Phone,
		"logo_url": gov.LogoURL, "created_at": gov.CreatedAt,
		"follower_count": followerCount,
	})
}

func adminUpdateGovernmentHandler(c *gin.Context) {
	govID := getGovID(c)
	var gov LocalGovernment
	if err := db.First(&gov, govID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "government not found"})
		return
	}
	var body struct {
		Name         string `json:"name"`
		Jurisdiction string `json:"jurisdiction"`
		State        string `json:"state"`
		Email        string `json:"email"`
		Phone        string `json:"phone"`
		LogoURL      string `json:"logo_url"`
	}
	c.ShouldBindJSON(&body)
	if body.Name != "" {
		gov.Name = body.Name
	}
	if body.Jurisdiction != "" {
		gov.Jurisdiction = body.Jurisdiction
	}
	if body.State != "" {
		gov.State = body.State
	}
	if body.Email != "" {
		gov.Email = body.Email
	}
	if body.Phone != "" {
		gov.Phone = body.Phone
	}
	if body.LogoURL != "" {
		gov.LogoURL = body.LogoURL
	}
	db.Save(&gov)
	c.JSON(http.StatusOK, gov)
}

// ── Main ────────────────────────────────────────────────────────────────────

func main() {
	log.Println("[admin-service] Starting Civic Connect Admin Service...")
	connectPostgres()
	connectRabbitMQ()
	connectRedis()
	log.Println("[admin-service] ✅ All connections established – Connected Successfully")

	r := gin.Default()

	// ── Public Routes ────────────────────────────────────────────────────
	r.GET("/health", healthHandler)
	r.POST("/register", registerHandler)
	r.POST("/login", loginHandler)
	r.GET("/governments", listGovernmentsHandler)
	r.GET("/governments/:id", getGovernmentHandler)
	r.GET("/localgovernments", listLocalGovernmentsHandler)
	r.GET("/localgovernments/:id", getLocalGovernmentHandler)
	r.GET("/localgovernments/search", searchLocalGovernmentsHandler)
	r.GET("/officials", listOfficialsHandler)
	r.GET("/officials/:id", getOfficialHandler)
	r.GET("/officials/:id/followers", getFollowersHandler)
	r.GET("/users/:user_id/following", getUserFollowingHandler)
	r.GET("/users/:user_id/governments", getUserFollowedGovernmentsHandler)

	// ── Citizen/Government Protected Routes ──────────────────────────────
	auth := r.Group("/", authMiddleware())
	{
		auth.GET("/check-auth", checkAuthHandler)
		auth.GET("/session/verify", sessionVerifyHandler)
		auth.GET("/profile", getProfileHandler)
		auth.PUT("/profile", updateProfileHandler)
		auth.PUT("/change-password", changePasswordHandler)
		auth.GET("/notifications", getNotificationsHandler)
		auth.POST("/logout", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "logged out"})
		})
		gov := auth.Group("/", roleRequired("government"))
		{
			gov.POST("/governments", createGovernmentHandler)
			gov.PUT("/governments/:id", updateGovernmentHandler)
			gov.DELETE("/governments/:id", deleteGovernmentHandler)
			gov.POST("/officials", createOfficialHandler)
		}
		auth.POST("/follow/:official_id", followOfficialHandler)
		auth.DELETE("/follow/:official_id", unfollowOfficialHandler)
		auth.GET("/follow/:official_id", checkFollowHandler)
		auth.POST("/follow/government/:gov_id", followGovernmentHandler)
		auth.DELETE("/follow/government/:gov_id", unfollowGovernmentHandler)
		auth.GET("/follow/government/:gov_id", checkGovernmentFollowHandler)
	}

	// ── Government Admin Portal Routes ───────────────────────────────────
	r.POST("/api/admin/login", adminLoginHandler)
	r.POST("/api/admin/seed", seedSuperAdminHandler)

	admin := r.Group("/api/admin", adminAuthMiddleware())
	{
		admin.GET("/me", adminMeHandler)
		admin.GET("/dashboard", dashboardHandler)
		admin.POST("/change-password", adminChangePasswordHandler)

		// Municipality management (SuperAdmin only)
		admin.GET("/municipalities", adminRoleRequired("super_admin"), listMunicipalitiesHandler)
		admin.POST("/municipalities", adminRoleRequired("super_admin"), createMunicipalityHandler)
		admin.PUT("/municipalities/:id", adminRoleRequired("super_admin"), updateMunicipalityHandler)
		admin.DELETE("/municipalities/:id", adminRoleRequired("super_admin"), deleteMunicipalityHandler)

		// Government info for current admin
		admin.GET("/government", adminGetGovernmentHandler)
		admin.PUT("/government", adminRoleRequired("manager"), adminUpdateGovernmentHandler)

		// Departments — Manager only creates/updates/deletes
		admin.GET("/departments", adminRoleRequired("manager", "dept_manager"), listDepartmentsHandler)
		admin.GET("/departments/:id", adminRoleRequired("manager", "dept_manager"), getDepartmentHandler)
		admin.POST("/departments", adminRoleRequired("manager"), createDepartmentHandler)
		admin.PUT("/departments/:id", adminRoleRequired("manager"), updateDepartmentHandler)
		admin.DELETE("/departments/:id", adminRoleRequired("manager"), deleteDepartmentHandler)

		// Officials management
		admin.POST("/officials", adminRoleRequired("manager"), createOfficialHandler)
		admin.PUT("/officials/:id", adminRoleRequired("manager"), updateOfficialHandler)
		admin.DELETE("/officials/:id", adminRoleRequired("manager"), deleteOfficialHandler)
		admin.GET("/officials", listOfficialsHandler)

		// Admin/Staff management
		admin.GET("/admins", adminRoleRequired("super_admin", "manager"), listAdminsHandler)
		admin.POST("/admins", adminRoleRequired("super_admin", "manager"), createAdminHandler)
		admin.PUT("/admins/:id", adminRoleRequired("super_admin", "manager"), updateAdminHandler)
		admin.DELETE("/admins/:id", adminRoleRequired("super_admin", "manager"), deleteAdminHandler)

		// Article categories
		admin.GET("/article-categories", adminRoleRequired("manager", "dept_manager"), listArticleCategoriesHandler)
		admin.POST("/article-categories", adminRoleRequired("manager"), createArticleCategoryHandler)
		admin.DELETE("/article-categories/:id", adminRoleRequired("manager"), deleteArticleCategoryHandler)

		// Users
		admin.GET("/users", adminRoleRequired("manager", "dept_manager"), listUsersHandler)

		// Local government listing (for super_admin)
		admin.GET("/governments", listLocalGovernmentsHandler)
	}

	port := env("PORT", "8081")
	log.Printf("[admin-service] Listening on :%s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("[admin-service] Server failed: %v", err)
	}
}
