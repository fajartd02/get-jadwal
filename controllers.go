package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type UserCache struct {
	UserIDCache    uint64
	EmailCache     string
	ScheduleCache  []Schedule
	CreatedAtCache time.Time
	UpdatedAtCache time.Time
}

var userCache = make(map[string]UserCache)

func ContainsAtSymbol(s string) bool {
	return strings.IndexByte(s, '@') != -1
}

func CheckinEmail(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Parse the request body
		var requestBody struct {
			Email string `json:"email"`
		}

		if err := c.BodyParser(&requestBody); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Invalid request body",
			})
		}

		// Check if the email is empty
		if requestBody.Email == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Email is required",
			})
		}

		if !ContainsAtSymbol(requestBody.Email) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Invalid email",
			})
		}

		cacheUser, ok := userCache[requestBody.Email]
		if ok {
			return c.JSON(fiber.Map{
				"status":  "Success",
				"message": "Success",
				"data": fiber.Map{
					"id":        cacheUser.UserIDCache,
					"email":     cacheUser.EmailCache,
					"updatedAt": cacheUser.UpdatedAtCache,
					"createdAt": cacheUser.CreatedAtCache,
				},
			})
		}

		// Create a new user record in the database
		user := User{
			Email:     requestBody.Email,
			Schedules: nil,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		result := db.Create(&user)
		if result.Error != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "Error",
				"message": "Failed to create user record",
			})
		}

		userCache[requestBody.Email] = UserCache{
			EmailCache:     requestBody.Email,
			UserIDCache:    user.ID,
			ScheduleCache:  []Schedule{},
			CreatedAtCache: user.CreatedAt,
			UpdatedAtCache: user.UpdatedAt,
		}

		// Return the response with the new user record
		return c.JSON(fiber.Map{
			"status":  "Success",
			"message": "Success",
			"data": fiber.Map{
				"id":        user.ID,
				"email":     requestBody.Email,
				"updatedAt": user.UpdatedAt,
				"createdAt": user.CreatedAt,
			},
		})
	}
}

func AddSchedule(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get the email parameter from the query string
		email := c.Query("email")

		if email == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Email is required",
			})
		}

		if !ContainsAtSymbol(email) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Invalid email",
			})
		}

		cacheUser, ok := userCache[email]
		if !ok {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "Not Found",
				"message": "Email is not found",
			})
		}

		// Check if the title is empty in the request body
		var requestBody struct {
			Title string `json:"title"`
			Day   string `json:"day"`
		}

		if err := c.BodyParser(&requestBody); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Invalid request body",
			})
		}

		if requestBody.Title == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Title is required",
			})
		}

		if requestBody.Day == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Day is required",
			})
		}

		// Check if the day parameter is a valid day of the week
		validDays := map[string]bool{
			"monday":    true,
			"tuesday":   true,
			"wednesday": true,
			"thursday":  true,
			"friday":    true,
			"saturday":  true,
			"sunday":    true,
		}

		if !validDays[requestBody.Day] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Day is invalid",
			})
		}

		// Create a new schedule record for the user
		schedule := Schedule{
			Title:     requestBody.Title,
			UserID:    cacheUser.UserIDCache,
			Day:       requestBody.Day,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		db.Create(&schedule)

		// Return the response
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"status":  "Success",
			"message": "Success",
			"data": fiber.Map{
				"id":        schedule.ScheduleID,
				"title":     schedule.Title,
				"user_id":   schedule.UserID,
				"day":       schedule.Day,
				"updatedAt": schedule.UpdatedAt,
				"createdAt": schedule.CreatedAt,
			},
		})
	}
}

func GetSchedules(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get the email and day parameters from the query string
		email := c.Query("email")
		day := c.Query("day")

		// Check if the email is empty
		if email == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Email is required",
			})
		}

		if !ContainsAtSymbol(email) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Invalid email",
			})
		}

		// Retrieve the user based on the given email
		var user User
		result := db.Preload("Schedules").Where("email = ?", email).First(&user)
		if result.Error != nil {
			// Handle the error if any
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "Not Found",
				"message": "Email is not found",
			})
		}

		// If the day parameter is empty, return all schedules
		if day == "" {
			// Organize the schedules by day
			scheduleByDay := make(map[string][]Schedule)

			// Initialize all days of the week
			daysOfWeek := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
			for _, day := range daysOfWeek {
				scheduleByDay[day] = []Schedule{}
			}

			// Group the schedules by day
			for _, schedule := range user.Schedules {
				scheduleByDay[schedule.Day] = append(scheduleByDay[schedule.Day], schedule)
			}

			// Return the organized schedules as the response
			return c.JSON(fiber.Map{
				"status":  "Success",
				"message": "Success",
				"data":    scheduleByDay,
			})
		}

		// Check if the day parameter is a valid day of the week
		validDays := map[string]bool{
			"monday":    true,
			"tuesday":   true,
			"wednesday": true,
			"thursday":  true,
			"friday":    true,
			"saturday":  true,
			"sunday":    true,
		}

		if !validDays[day] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Day is invalid",
			})
		}

		// Filter the schedules based on the given day
		var schedules []Schedule
		for _, schedule := range user.Schedules {
			if schedule.Day == day {
				schedules = append(schedules, schedule)
			}
		}

		// Return the filtered schedules as the response
		return c.JSON(fiber.Map{
			"status":  "Success",
			"message": "Success",
			"data":    schedules,
		})
	}
}

func EditSchedule(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get the email and id parameters from the query string
		email := c.Query("email")
		id := c.Query("id")

		// Check if the email is empty
		if email == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Email is required",
			})
		}

		if !ContainsAtSymbol(email) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Invalid email",
			})
		}

		// Check if the id is empty
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "ID is required",
			})
		}

		// Convert the id to uint64
		scheduleID, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Invalid ID",
			})
		}

		// Find the schedule with the given ID
		var schedule Schedule
		result := db.Where("schedule_id = ?", scheduleID).First(&schedule)
		if result.Error != nil {
			// Schedule not found
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "Not Found",
				"message": fmt.Sprintf("Schedule with ID %d Not Found", scheduleID),
			})
		}

		// Retrieve the user based on the email
		var user User
		result = db.Where("email = ?", email).First(&user)
		if result.Error != nil {
			// User not found
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "Not Found",
				"message": "Email is not found",
			})
		}

		// Check if the schedule belongs to the user
		if schedule.UserID != user.ID {
			// Access denied
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "Forbidden",
				"message": "Access denied!",
			})
		}

		// Parse the request body
		var requestBody struct {
			Title string `json:"title"`
		}
		if err := c.BodyParser(&requestBody); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Invalid request body",
			})
		}

		if requestBody.Title == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Title is required",
			})
		}

		// Update the schedule title
		schedule.Title = requestBody.Title

		// Save the updated schedule to the database
		db.Save(&schedule)

		// Return the updated schedule as the response
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"status":  "Success",
			"message": "Success",
			"data":    schedule,
		})
	}
}

// DeleteSchedule handles the deletion of a schedule
func DeleteSchedule(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get the email and id parameters from the query string
		email := c.Query("email")
		id := c.Query("id")

		// Check if the email is empty
		if email == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Email is required",
			})
		}

		if !ContainsAtSymbol(email) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Invalid email",
			})
		}

		// Check if the id is empty
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "ID is required",
			})
		}

		// Convert the id string to uint64
		scheduleID, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "Bad Request",
				"message": "Invalid ID",
			})
		}

		// Retrieve the schedule from the database
		var schedule Schedule
		result := db.First(&schedule, scheduleID)
		if result.Error != nil {
			// Handle the error if the schedule is not found
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"status":  "Not Found",
					"message": fmt.Sprintf("Schedule with ID %d Not Found", scheduleID),
				})
			}
			// Handle other database errors
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "Internal Server Error",
				"message": "Failed to delete schedule",
			})
		}

		// Retrieve the user based on the email
		var user User
		result = db.Where("email = ?", email).First(&user)
		if result.Error != nil {
			// User not found
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "Not Found",
				"message": "Email is not found",
			})
		}

		// Check if the schedule belongs to the user
		if schedule.UserID != user.ID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "Forbidden",
				"message": "Access denied!",
			})
		}

		// Delete the schedule from the database
		db.Delete(&schedule)

		// Return a success response
		return c.JSON(fiber.Map{
			"status":  "Success",
			"message": "Success",
			"data":    fiber.Map{},
		})
	}
}
