package app

import (
    "fmt"
    "gorm.io/gorm"	
)


func InitializeDBAll() (*gorm.DB,error){
	var err error
	
	cfg, err := LoadConfig("configs/config.yaml")
	// DB
	if err != nil{
		return nil, fmt.Errorf("File Open* %w", err)
	}
    db, err := ConnectDB(cfg)
	if err != nil{
        return nil, fmt.Errorf("DB Connection: %w", err)
    }

	err = InitDB(db)
    if err != nil{
        return nil, fmt.Errorf("DB Initialization: %w", err)
    }
    return db,nil
}
