package database

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"main/model"

	_ "github.com/lib/pq"
)

var DB *gorm.DB

// функция подключения к базе данных
func Connect() error {
	connStr := "user=username dbname=wishListDatabase sslmode=disable password=password"
	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("ошибка подключения к БД: %v", err)
	}

	DB = db
	err = db.AutoMigrate(&model.User{}, &model.Present{}, &model.Wishlist{})
	if err != nil {
		return fmt.Errorf("ошибка миграции: %v", err)
	}
	log.Println("Успешно подключились к базе данных")
	return nil
}
