package config

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() *gorm.DB {
	// dsn := "root:Ap@123456@tcp(192.168.3.19:3306)/hope?charset=utf8mb4&parseTime=True&loc=Local"
	dsn := "root:Ap@123456@tcp(127.0.0.1:3306)/hope?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	DB = db
	fmt.Println("Database connected successfully!")
	return DB
}
