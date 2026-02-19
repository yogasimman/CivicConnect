// =============================================================================
// Civic Connect – Complaint Service  (Go + Gin + GORM + PostGIS + MinIO)
// =============================================================================
// Connects to: PostgreSQL (complaint_db + PostGIS), RabbitMQ, Redis, MinIO
// Port: 8083
//
// Domains: Complaints (geo-tagged, multi-image), Upvote/Downvote,
//          Comments, Actions Taken (completion tracking, auto-resolve),
//          Priority Scoring, Nearby Search
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
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ── Models ──────────────────────────────────────────────────────────────────

type Complaint struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	GovernmentID   uint      `gorm:"index;not null" json:"government_id"`
	DepartmentID   *uint     `gorm:"index" json:"department_id,omitempty"`
	UserID         uint      `gorm:"not null" json:"user_id"`
	Category       string    `gorm:"not null" json:"category"`
	Description    string    `gorm:"type:text;not null" json:"description"`
	MultimediaURLs string    `gorm:"type:text" json:"multimedia_urls,omitempty"` // JSON array of image URLs
	Status         string    `gorm:"default:pending" json:"status"`             // pending | in_progress | resolved | rejected
	Upvotes        int       `gorm:"default:0" json:"upvotes"`
	Downvotes      int       `gorm:"default:0" json:"downvotes"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	ManualLocation string    `json:"manual_location,omitempty"`
	Version        int       `gorm:"default:1" json:"version"`
	AIAnalysis     string    `gorm:"type:text" json:"ai_analysis,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Priority score = upvotes - (downvotes * 2)
func (c *Complaint) PriorityScore() int {
	return c.Upvotes - (c.Downvotes * 2)
}

type ComplaintUpvote struct {
	ID          uint `gorm:"primaryKey" json:"id"`
	ComplaintID uint `gorm:"not null" json:"complaint_id"`
	UserID      uint `gorm:"not null" json:"user_id"`
}

type ComplaintDownvote struct {
	ID          uint `gorm:"primaryKey" json:"id"`
	ComplaintID uint `gorm:"not null" json:"complaint_id"`
	UserID      uint `gorm:"not null" json:"user_id"`
}

type ComplaintComment struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ComplaintID uint      `gorm:"index;not null" json:"complaint_id"`
	UserID      uint      `gorm:"not null" json:"user_id"`
	Content     string    `gorm:"type:text;not null" json:"content"`
	CreatedAt   time.Time `json:"created_at"`
}

// ActionTaken — government response to a complaint with completion %
type ActionTaken struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	ComplaintID         uint      `gorm:"index;not null" json:"complaint_id"`
	GovernmentID        uint      `gorm:"not null" json:"government_id"`
	AdminID             uint      `gorm:"not null" json:"admin_id"`
	ActionDetails       string    `gorm:"type:text;not null" json:"action_details"`
	ActionMultimediaURLs string   `gorm:"type:text" json:"action_multimedia_urls,omitempty"`
	CompletionPercent   int       `gorm:"default:0" json:"completion_percentage"`
	CreatedAt           time.Time `json:"created_at"`
}

// ── Globals ─────────────────────────────────────────────────────────────────

var (
	db          *gorm.DB
	rdb         *redis.Client
	amqpConn    *amqp.Connection
	minioClient *minio.Client
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
		env("DB_NAME", "complaint_db"),
	)

	var err error
	for i := 0; i < 30; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("[complaint-service] Waiting for PostgreSQL... (%d/30)", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("[complaint-service] PostgreSQL connection failed: %v", err)
	}

	// Enable PostGIS
	sqlDB, _ := db.DB()
	sqlDB.Exec("CREATE EXTENSION IF NOT EXISTS postgis")

	db.AutoMigrate(
		&Complaint{}, &ComplaintUpvote{}, &ComplaintDownvote{},
		&ComplaintComment{}, &ActionTaken{},
	)

	// Unique constraints
	sqlDB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_upvote_unique ON complaint_upvotes(complaint_id, user_id)")
	sqlDB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_downvote_unique ON complaint_downvotes(complaint_id, user_id)")

	log.Println("[complaint-service] ✅ PostgreSQL Connected Successfully (PostGIS enabled)")
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
		log.Printf("[complaint-service] Waiting for RabbitMQ... (%d/30)", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("[complaint-service] RabbitMQ connection failed: %v", err)
	}
	log.Println("[complaint-service] ✅ RabbitMQ Connected Successfully")
}

func connectRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     env("REDIS_ADDR", "localhost:6379"),
		Password: env("REDIS_PASSWORD", "redis_secret_2026"),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for i := 0; i < 30; i++ {
		if err := rdb.Ping(ctx).Err(); err == nil {
			break
		}
		log.Printf("[complaint-service] Waiting for Redis... (%d/30)", i+1)
		time.Sleep(2 * time.Second)
	}
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("[complaint-service] Redis connection failed: %v", err)
	}
	log.Println("[complaint-service] ✅ Redis Connected Successfully")
}

func connectMinIO() {
	endpoint := env("MINIO_ENDPOINT", "localhost:9000")
	accessKey := env("MINIO_ACCESS_KEY", "civic_minio")
	secretKey := env("MINIO_SECRET_KEY", "minio_secret_2026")
	bucket := env("MINIO_BUCKET", "civic-complaints")

	var err error
	for i := 0; i < 30; i++ {
		minioClient, err = minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure: false,
		})
		if err == nil {
			ctx := context.Background()
			exists, errBucket := minioClient.BucketExists(ctx, bucket)
			if errBucket == nil {
				if !exists {
					if mkErr := minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); mkErr != nil {
						log.Printf("[complaint-service] Bucket creation warning: %v", mkErr)
					}
				}
				break
			}
			err = errBucket
		}
		log.Printf("[complaint-service] Waiting for MinIO... (%d/30)", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("[complaint-service] MinIO connection failed: %v", err)
	}
	log.Println("[complaint-service] ✅ MinIO Connected Successfully")
}

// ── Handlers ────────────────────────────────────────────────────────────────

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "complaint-service"})
}

// List complaints for a government (priority-sorted)
func listComplaintsHandler(c *gin.Context) {
	govID := c.Query("government_id")
	status := c.Query("status")
	deptID := c.Query("department_id")

	var complaints []Complaint
	query := db
	if govID != "" {
		query = query.Where("government_id = ?", govID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if deptID != "" {
		query = query.Where("department_id = ?", deptID)
	}
	query.Order("(upvotes - downvotes * 2) DESC, created_at DESC").Find(&complaints)
	c.JSON(http.StatusOK, complaints)
}

func getComplaintHandler(c *gin.Context) {
	id := c.Param("id")
	var complaint Complaint
	if err := db.First(&complaint, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "complaint not found"})
		return
	}
	c.JSON(http.StatusOK, complaint)
}

func createComplaintHandler(c *gin.Context) {
	var complaint Complaint
	if err := c.ShouldBindJSON(&complaint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	complaint.Status = "pending"
	complaint.Version = 1
	if err := db.Create(&complaint).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Publish to RabbitMQ for AI analysis
	if amqpConn != nil {
		ch, err := amqpConn.Channel()
		if err == nil {
			defer ch.Close()
			body, _ := json.Marshal(map[string]interface{}{
				"complaint_id":  complaint.ID,
				"description":   complaint.Description,
				"category":      complaint.Category,
				"latitude":      complaint.Latitude,
				"longitude":     complaint.Longitude,
				"government_id": complaint.GovernmentID,
			})
			ch.Publish("", "complaint_analysis", false, false, amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			})
		}
	}

	c.JSON(http.StatusCreated, complaint)
}

func updateComplaintHandler(c *gin.Context) {
	id := c.Param("id")
	var complaint Complaint
	if err := db.First(&complaint, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "complaint not found"})
		return
	}

	var update struct {
		Description    string `json:"description"`
		MultimediaURLs string `json:"multimedia_urls"`
		Status         string `json:"status"`
	}
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if update.Description != "" {
		complaint.Description = update.Description
	}
	if update.MultimediaURLs != "" {
		complaint.MultimediaURLs = update.MultimediaURLs
	}
	if update.Status != "" {
		complaint.Status = update.Status
	}
	complaint.Version++
	db.Save(&complaint)

	c.JSON(http.StatusOK, complaint)
}

// ── Upvote / Downvote ───────────────────────────────────────────────────────

func upvoteHandler(c *gin.Context) {
	complaintID, _ := strconv.Atoi(c.Param("id"))
	var body struct {
		UserID uint `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vote := ComplaintUpvote{ComplaintID: uint(complaintID), UserID: body.UserID}
	if err := db.Create(&vote).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "already upvoted"})
		return
	}
	db.Model(&Complaint{}).Where("id = ?", complaintID).Update("upvotes", gorm.Expr("upvotes + 1"))
	c.JSON(http.StatusOK, gin.H{"message": "upvoted"})
}

func downvoteHandler(c *gin.Context) {
	complaintID, _ := strconv.Atoi(c.Param("id"))
	var body struct {
		UserID uint `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vote := ComplaintDownvote{ComplaintID: uint(complaintID), UserID: body.UserID}
	if err := db.Create(&vote).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "already downvoted"})
		return
	}
	db.Model(&Complaint{}).Where("id = ?", complaintID).Update("downvotes", gorm.Expr("downvotes + 1"))
	c.JSON(http.StatusOK, gin.H{"message": "downvoted"})
}

// ── Comments ────────────────────────────────────────────────────────────────

func getCommentsHandler(c *gin.Context) {
	complaintID := c.Param("id")
	var comments []ComplaintComment
	db.Where("complaint_id = ?", complaintID).Order("created_at DESC").Find(&comments)
	c.JSON(http.StatusOK, comments)
}

func addCommentHandler(c *gin.Context) {
	var comment ComplaintComment
	if err := c.ShouldBindJSON(&comment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db.Create(&comment)
	c.JSON(http.StatusCreated, comment)
}

// ── Actions Taken ───────────────────────────────────────────────────────────

func getActionsHandler(c *gin.Context) {
	complaintID := c.Param("id")
	var actions []ActionTaken
	db.Where("complaint_id = ?", complaintID).Order("created_at DESC").Find(&actions)
	c.JSON(http.StatusOK, actions)
}

func addActionHandler(c *gin.Context) {
	complaintID, _ := strconv.Atoi(c.Param("id"))
	var action ActionTaken
	if err := c.ShouldBindJSON(&action); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	action.ComplaintID = uint(complaintID)
	db.Create(&action)

	// Auto-resolve at 100% completion
	if action.CompletionPercent >= 100 {
		db.Model(&Complaint{}).Where("id = ?", complaintID).Update("status", "resolved")
	} else if action.CompletionPercent > 0 {
		db.Model(&Complaint{}).Where("id = ?", complaintID).Update("status", "in_progress")
	}

	c.JSON(http.StatusCreated, action)
}

// ── Nearby Search (PostGIS) ─────────────────────────────────────────────────

func nearbyComplaintsHandler(c *gin.Context) {
	lat, _ := strconv.ParseFloat(c.Query("lat"), 64)
	lng, _ := strconv.ParseFloat(c.Query("lng"), 64)
	radius := c.DefaultQuery("radius", "5000") // meters

	var complaints []Complaint
	db.Raw(`
		SELECT * FROM complaints
		WHERE ST_DWithin(
			ST_MakePoint(longitude, latitude)::geography,
			ST_MakePoint(?, ?)::geography,
			?
		)
		ORDER BY (upvotes - downvotes * 2) DESC, created_at DESC
	`, lng, lat, radius).Scan(&complaints)

	c.JSON(http.StatusOK, complaints)
}

// ── Image Upload (MinIO) ────────────────────────────────────────────────────

func uploadImageHandler(c *gin.Context) {
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no image file provided"})
		return
	}
	defer file.Close()

	bucket := env("MINIO_BUCKET", "civic-complaints")
	objectName := fmt.Sprintf("complaints/%d_%s", time.Now().UnixNano(), header.Filename)

	_, err = minioClient.PutObject(
		context.Background(),
		bucket,
		objectName,
		file,
		header.Size,
		minio.PutObjectOptions{ContentType: header.Header.Get("Content-Type")},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	imageURL := fmt.Sprintf("http://%s/%s/%s", env("MINIO_ENDPOINT", "localhost:9000"), bucket, objectName)
	c.JSON(http.StatusOK, gin.H{"image_url": imageURL})
}

// Upload action image
func uploadActionImageHandler(c *gin.Context) {
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no image file provided"})
		return
	}
	defer file.Close()

	bucket := env("MINIO_BUCKET", "civic-complaints")
	objectName := fmt.Sprintf("actions/%d_%s", time.Now().UnixNano(), header.Filename)

	_, err = minioClient.PutObject(
		context.Background(),
		bucket,
		objectName,
		file,
		header.Size,
		minio.PutObjectOptions{ContentType: header.Header.Get("Content-Type")},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	imageURL := fmt.Sprintf("http://%s/%s/%s", env("MINIO_ENDPOINT", "localhost:9000"), bucket, objectName)
	c.JSON(http.StatusOK, gin.H{"image_url": imageURL})
}

// ── Main ────────────────────────────────────────────────────────────────────

func main() {
	log.Println("[complaint-service] Starting Civic Connect Complaint Service...")

	connectPostgres()
	connectRabbitMQ()
	connectRedis()
	connectMinIO()

	log.Println("[complaint-service] ✅ All connections established – Connected Successfully")

	r := gin.Default()

	r.GET("/health", healthHandler)

	// Complaints CRUD
	r.GET("/complaints", listComplaintsHandler)
	r.GET("/complaints/:id", getComplaintHandler)
	r.POST("/complaints", createComplaintHandler)
	r.PUT("/complaints/:id", updateComplaintHandler)

	// Voting
	r.POST("/complaints/:id/upvote", upvoteHandler)
	r.POST("/complaints/:id/downvote", downvoteHandler)

	// Comments
	r.GET("/complaints/:id/comments", getCommentsHandler)
	r.POST("/complaints/comments", addCommentHandler)

	// Actions Taken
	r.GET("/complaints/:id/actions", getActionsHandler)
	r.POST("/complaints/:id/actions", addActionHandler)

	// Nearby Search
	r.GET("/complaints/nearby", nearbyComplaintsHandler)

	// Image Upload
	r.POST("/complaints/upload", uploadImageHandler)
	r.POST("/complaints/upload/action", uploadActionImageHandler)

	port := env("PORT", "8083")
	log.Printf("[complaint-service] Listening on :%s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("[complaint-service] Server failed: %v", err)
	}
}
