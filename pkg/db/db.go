package db

import (
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	AfrusDB  = "afrus"
	EventsDB = "eventosdb"
)

type DBConfig struct {
	Host     string
	User     string
	Password string
	DBName   string
	Port     string
}

type DatabaseManager struct {
	connections map[string]*gorm.DB
	mu          sync.RWMutex
}

func NewDatabaseManager() *DatabaseManager {
	return &DatabaseManager{
		connections: make(map[string]*gorm.DB),
	}
}

func (dm *DatabaseManager) Connect(dbType string, config DBConfig, models ...interface{}) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.Host, config.User, config.Password, config.DBName, config.Port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("error opening %s database: %v", dbType, err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("error getting underlying SQL DB for %s: %v", dbType, err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Auto migrate models if provided
	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			return fmt.Errorf("error auto-migrating models for %s: %v", dbType, err)
		}
	}

	dm.connections[dbType] = db
	log.Printf("[DATABASE] - %s database connection established", dbType)
	return nil
}

func (dm *DatabaseManager) GetDB(dbType string) (*gorm.DB, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	db, exists := dm.connections[dbType]
	if !exists {
		return nil, fmt.Errorf("no connection found for database: %s", dbType)
	}
	return db, nil
}

func (dm *DatabaseManager) CloseAll() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	for dbType, db := range dm.connections {
		sqlDB, err := db.DB()
		if err != nil {
			log.Printf("error getting underlying SQL DB for %s: %v", dbType, err)
			continue
		}
		if err := sqlDB.Close(); err != nil {
			log.Printf("error closing %s database connection: %v", dbType, err)
		}
	}
	dm.connections = make(map[string]*gorm.DB)
}
