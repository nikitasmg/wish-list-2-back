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

// Connect функция подключения к базе данных
func Connect() error {
	connStr := "host=postgres port=5432 user=postgres password=postgres dbname=wishlist_db sslmode=disable"

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
