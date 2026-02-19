// =============================================================================
// Civic Connect – Admin Service  (Go + Gin + GORM)
// =============================================================================
// Connects to: PostgreSQL (admin_db), RabbitMQ, Redis
// Port: 8081 (HTTP)
//
// Domains: Authentication, User Management, Government Entities,
//          Officials, Follow System, Departments, Admin Accounts,
//          Notifications, Government Settings
// =============================================================================

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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

// User — citizen or government role, Aadhar-based identity
type User struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Name          string    `gorm:"not null" json:"name"`
	Email         string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash  string    `gorm:"column:password_hash;not null" json:"-"`
	Role          string    `gorm:"not null;default:public" json:"role"` // public | government
	AadharNo      string    `gorm:"uniqueIndex;not null" json:"aadhar_no"`
	ProofDocument string    `json:"proof_document,omitempty"`
	Location      string    `json:"location,omitempty"`
	ProfilePic    string    `json:"profile_pic,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Notification — fetch-and-delete pattern
type Notification struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	UserID    uint            `gorm:"index;not null" json:"user_id"`
	Title     string          `gorm:"not null" json:"title"`
	Body      string          `json:"body"`
	Data      json.RawMessage `gorm:"type:jsonb" json:"data,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// Government — central or state level entity
type Government struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Type      string    `gorm:"not null" json:"type"` // central | state
	State     string    `json:"state,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// LocalGovernment — municipality / corporation
type LocalGovernment struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"not null" json:"name"`
	Jurisdiction string `json:"jurisdiction,omitempty"`
	Email        string `json:"email,omitempty"`
	Phone        string `json:"phone,omitempty"`
}

// Department — under a local government
type Department struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"not null" json:"name"`
	Email        string `json:"email,omitempty"`
	Phone        string `json:"phone,omitempty"`
	Services     string `gorm:"type:text" json:"services,omitempty"`
	GovernmentID uint   `gorm:"index;not null" json:"government_id"`
}

// GovernmentAdmin — admin user for the government web portal
type GovernmentAdmin struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	GovernmentID uint      `gorm:"index;not null" json:"government_id"`
	Name         string    `gorm:"not null" json:"name"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"column:password_hash;not null" json:"-"`
	Role         string    `gorm:"not null;default:dept_manager" json:"role"` // super_admin | manager | dept_manager
	DepartmentID *uint     `json:"department_id,omitempty"`
	LastLogin    time.Time `json:"last_login,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// GovernmentOfficial — a government officer linked to a user account
type GovernmentOfficial struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	UserID     uint   `gorm:"uniqueIndex;not null" json:"user_id"`
	Department string `json:"department,omitempty"`
	Position   string `json:"position,omitempty"`
	User       User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// Follower — citizen follows a government official
type Follower struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"not null" json:"user_id"`
	OfficialID uint      `gorm:"not null" json:"official_id"`
	FollowedAt time.Time `gorm:"autoCreateTime" json:"followed_at"`
}

// GovernmentFollow — citizen follows a local government
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
		&GovernmentFollow{},
	)

	// Unique constraint: one follow per user-official pair
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_followers_unique ON followers(user_id, official_id)")
	// Unique constraint: one follow per user-government pair
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_gov_follow_unique ON government_follows(user_id, government_id)")

	// Migrate existing department_admin roles to dept_manager
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

// authMiddleware — validates Bearer token for citizen/government users
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

// adminAuthMiddleware — validates Bearer token for government admin portal
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
	return func(c *gin.Context) {
		role, _ := c.Get("admin_role")
		for _, r := range roles {
			if r == role {
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
		Name:          body.Name,
		Email:         body.Email,
		PasswordHash:  string(hash),
		Role:          role,
		AadharNo:      body.AadharNo,
		ProofDocument: body.ProofDocument,
		Location:      body.Location,
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
	// Fetch-and-delete pattern
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

// ── Department Handlers ─────────────────────────────────────────────────────

func listDepartmentsHandler(c *gin.Context) {
	govID := c.Query("government_id")
	var depts []Department
	q := db
	if govID != "" {
		q = q.Where("government_id = ?", govID)
	}
	q.Find(&depts)
	c.JSON(http.StatusOK, depts)
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
	var dept Department
	if err := c.ShouldBindJSON(&dept); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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
	c.ShouldBindJSON(&dept)
	db.Save(&dept)
	c.JSON(http.StatusOK, dept)
}

func deleteDepartmentHandler(c *gin.Context) {
	db.Delete(&Department{}, c.Param("id"))
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
	c.JSON(http.StatusOK, gin.H{"token": token, "admin": admin})
}

func adminMeHandler(c *gin.Context) {
	adminID, _ := c.Get("admin_id")
	var admin GovernmentAdmin
	if err := db.First(&admin, adminID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}
	c.JSON(http.StatusOK, admin)
}

func dashboardHandler(c *gin.Context) {
	govID, _ := c.Get("government_id")
	role, _ := c.Get("admin_role")
	deptID, _ := c.Get("department_id")

	ctx := context.Background()

	result := gin.H{
		"role": role,
	}

	switch role {
	case "super_admin":
		var govCount, deptCount, adminCount, userCount int64
		db.Model(&LocalGovernment{}).Count(&govCount)
		db.Model(&Department{}).Where("government_id = ?", govID).Count(&deptCount)
		db.Model(&GovernmentAdmin{}).Where("government_id = ?", govID).Count(&adminCount)
		db.Model(&User{}).Count(&userCount)
		result["governments"] = govCount
		result["departments"] = deptCount
		result["admins"] = adminCount
		result["users"] = userCount

	case "manager":
		var deptCount int64
		db.Model(&Department{}).Where("government_id = ?", govID).Count(&deptCount)
		result["departments"] = deptCount
		// Complaint stats from Redis cache
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
		result["department_id"] = deptID
		// Department-specific complaint stats from Redis
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
	govID, _ := c.Get("government_id")
	var admins []GovernmentAdmin
	db.Where("government_id = ?", govID).Find(&admins)
	c.JSON(http.StatusOK, admins)
}

func createAdminHandler(c *gin.Context) {
	govID, _ := c.Get("government_id")
	callerRole, _ := c.Get("admin_role")
	var body struct {
		Name         string `json:"name" binding:"required"`
		Email        string `json:"email" binding:"required,email"`
		Password     string `json:"password" binding:"required,min=6"`
		Role         string `json:"role" binding:"required"`
		DepartmentID *uint  `json:"department_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Validate role
	validRoles := map[string]bool{"super_admin": true, "manager": true, "dept_manager": true}
	if !validRoles[body.Role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role, must be: super_admin, manager, or dept_manager"})
		return
	}
	// Manager can only create dept_manager accounts
	if callerRole == "manager" && body.Role != "dept_manager" {
		c.JSON(http.StatusForbidden, gin.H{"error": "managers can only create department manager accounts"})
		return
	}
	// dept_manager must have a department
	if body.Role == "dept_manager" && body.DepartmentID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "department_id is required for dept_manager role"})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	admin := GovernmentAdmin{
		GovernmentID: uint(govID.(float64)),
		Name:         body.Name,
		Email:        body.Email,
		PasswordHash: string(hash),
		Role:         body.Role,
		DepartmentID: body.DepartmentID,
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
	db.Delete(&GovernmentAdmin{}, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func adminChangePasswordHandler(c *gin.Context) {
	adminID, _ := c.Get("admin_id")
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

// ── Session Verify (inter-service) ──────────────────────────────────────────

func sessionVerifyHandler(c *gin.Context) {
	// Called by other services to validate a user token
	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"user_id": c.GetFloat64("user_id"),
		"role":    c.GetString("role"),
	})
}

// ── Government Follow Handlers (citizens follow local governments) ──────────

func followGovernmentHandler(c *gin.Context) {
	govID, _ := strconv.Atoi(c.Param("gov_id"))
	// Verify government exists
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
	// Add follower count and follow status for current user
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
			LocalGovernment: g,
			FollowerCount:   int(count),
			IsFollowing:     followingCount > 0,
		})
	}
	c.JSON(http.StatusOK, result)
}

// ── Seed Initial SuperAdmin ─────────────────────────────────────────────────

func seedSuperAdminHandler(c *gin.Context) {
	// Only allow if no super_admin exists
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
	// Create local government
	gov := LocalGovernment{Name: body.GovernmentName, Jurisdiction: body.Jurisdiction}
	db.Create(&gov)
	// Create super admin
	hash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	admin := GovernmentAdmin{
		GovernmentID: gov.ID,
		Name:         body.Name,
		Email:        body.Email,
		PasswordHash: string(hash),
		Role:         "super_admin",
	}
	db.Create(&admin)
	token, _ := generateAdminToken(admin)
	c.JSON(http.StatusCreated, gin.H{"token": token, "admin": admin, "government": gov})
}

// ── User Listing (admin) ────────────────────────────────────────────────────

func listUsersHandler(c *gin.Context) {
	var users []User
	db.Find(&users)
	c.JSON(http.StatusOK, users)
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

		// Government entity management (requires government role)
		gov := auth.Group("/", roleRequired("government"))
		{
			gov.POST("/governments", createGovernmentHandler)
			gov.PUT("/governments/:id", updateGovernmentHandler)
			gov.DELETE("/governments/:id", deleteGovernmentHandler)
			gov.POST("/officials", createOfficialHandler)
		}

		// Follow officials
		auth.POST("/follow/:official_id", followOfficialHandler)
		auth.DELETE("/follow/:official_id", unfollowOfficialHandler)
		auth.GET("/follow/:official_id", checkFollowHandler)

		// Follow local governments
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

		// Local government settings
		admin.GET("/government", getLocalGovernmentHandler)
		admin.PUT("/government", updateLocalGovernmentHandler)
		admin.GET("/governments", listLocalGovernmentsHandler)

		// Departments — SuperAdmin & Manager can manage
		admin.GET("/departments", listDepartmentsHandler)
		admin.GET("/departments/:id", getDepartmentHandler)
		admin.POST("/departments", adminRoleRequired("super_admin", "manager"), createDepartmentHandler)
		admin.PUT("/departments/:id", adminRoleRequired("super_admin", "manager"), updateDepartmentHandler)
		admin.DELETE("/departments/:id", adminRoleRequired("super_admin"), deleteDepartmentHandler)

		// Officials management
		admin.POST("/officials", adminRoleRequired("super_admin", "manager"), createOfficialHandler)
		admin.PUT("/officials/:id", adminRoleRequired("super_admin", "manager"), updateOfficialHandler)
		admin.DELETE("/officials/:id", adminRoleRequired("super_admin"), deleteOfficialHandler)
		admin.GET("/officials", listOfficialsHandler)

		// Admin management — SuperAdmin manages all, Manager manages dept_managers
		admin.GET("/admins", adminRoleRequired("super_admin", "manager"), listAdminsHandler)
		admin.POST("/admins", adminRoleRequired("super_admin", "manager"), createAdminHandler)
		admin.PUT("/admins/:id", adminRoleRequired("super_admin", "manager"), updateAdminHandler)
		admin.DELETE("/admins/:id", adminRoleRequired("super_admin"), deleteAdminHandler)

		// Users (admin view)
		admin.GET("/users", listUsersHandler)
	}

	port := env("PORT", "8081")
	log.Printf("[admin-service] Listening on :%s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("[admin-service] Server failed: %v", err)
	}
}
