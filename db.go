package main

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"path"
	"time"
)

var (
	db     gorm.DB
	dbPath string = "./data/champloo.db"
)

type Server struct {
	Id               int `gorm:"primary_key:yes"`
	Host             string
	Ip               string `sql:"not null;unique"`
	Port             int
	Tags             string `form:"tags"`
	Disable          bool   `sql:"not null"` // 是否停用
	LastHeatbeatTime time.Time
}

type SystemConfig struct {
	Id              int    `gorm:"primary_key:yes" form:"id"`
	Name            string `sql:"not null;unique" form:"name"`
	EnableDevStage  bool   `sql:"not null" form:"dev-stage"`
	EnableProdStage bool   `sql:"not null" form:"prod-stage"`
	Way             string `form:"way"`
	Path            string `form:"path"`
	Shared          string `form:"shared"`
	BackupNum       int    `form:"num"`
	Repo            string `form:"repo"`
	UserName        string `form:"username"`
	Password        string `form:"password"`
	Tags            string `form:"tags"`
	BeforeCmd       string `form:"before-cmd"`
	AfterCmd        string `form:"after-cmd"`
	DevBeforeCmd    string `form:"dev-before-cmd"`
	DevAfterCmd     string `form:"dev-after-cmd"`
	ProdBeforeCmd   string `form:"prod-before-cmd"`
	ProdAfterCmd    string `form:"prod-after-cmd"`
	AutoUpdate      bool   `sql:"not null"` // 是否自动更新，只有way是update方式时才有效
	IsUserStar      bool   `sql:"-"`        // 是否被用户收藏
	EnableDeploy    Deploy `sql:"-"`        // 当前正在启用的部署版本
}

type Deploy struct {
	Id          int    `gorm:"primary_key:yes" form:"id"`
	SystemId    int    `sql:"not null"`
	Version     string `sql:"not null" form:"version"`
	Stage       string
	ElapsedTime int
	Operator    string
	Status      int
	Output      string
	Enable      bool `sql:"not null"` // 是否当前使用版本
	CreatedAt   time.Time
}

type User struct {
	Id        int    `gorm:"primary_key:yes" form:"id"`
	Name      string `sql:"not null;unique" form:"name"`
	Password  string `form:"password"`
	Avatar    string `form:"avatar"`
	Email     string `form:"email"`
	IsAdmin   bool   `sql:"not null" form:"isadmin"`
	CreatedAt time.Time
}

type UserStar struct {
	Id       int `gorm:"primary_key:yes" form:"id"`
	UserId   int
	SystemId int
}

func InitDb() error {
	os.MkdirAll(path.Dir(dbPath), os.ModePerm)
	var err error
	db, err = gorm.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("models.init(fail to conntect database): %v", err)
	}
	db.AutoMigrate(Server{}, SystemConfig{}, Deploy{}, User{}, UserStar{})

	var count int
	db.Model(User{}).Count(&count)
	if count <= 0 {
		db.Save(User{
			Name:      "admin",
			Password:  "123",
			IsAdmin:   true,
			CreatedAt: time.Now(),
		})
	}

	return nil
}
