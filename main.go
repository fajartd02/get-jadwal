package main

import (
	"log"
	"os"
	"runtime"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// Load the environment variables from .env file
	godotenv.Load()

	n := runtime.NumCPU()
	runtime.GOMAXPROCS(n)

	// Open a connection to the MySQL database
	dsn := os.Getenv("MYSQL_USER") + ":" + os.Getenv("MYSQL_PASSWORD") + "@tcp(" +
		os.Getenv("MYSQL_HOST") + ":" + os.Getenv("MYSQL_PORT") + ")/" +
		os.Getenv("MYSQL_DBNAME") + "?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate the database tables
	go func() {
		db.AutoMigrate(&User{}, &Schedule{})
	}()

	// Initialize the Fiber app
	app := fiber.New()

	// Fetch all users and their schedules from the database
	var users []User
	db.Preload("Schedules").Find(&users)

	// Populate the userCache map
	for _, user := range users {
		userCache[user.Email] = UserCache{
			EmailCache:     user.Email,
			UserIDCache:    user.ID,
			ScheduleCache:  user.Schedules,
			CreatedAtCache: user.CreatedAt,
			UpdatedAtCache: user.UpdatedAt,
		}
	}

	// Define the routes
	app.Post("/checkin", CheckinEmail(db))
	app.Post("/schedule", AddSchedule(db))
	app.Get("/schedule", GetSchedules(db))
	app.Patch("/schedule", EditSchedule(db))
	app.Delete("/schedule", DeleteSchedule(db))

	// Start the server
	err = app.Listen(":3030")
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}

}
