package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/cors"

	"todolist.go/db"
	"todolist.go/service"
)

const port = 8000

func main() {
	// initialize DB connection
	dsn := db.DefaultDSN(
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"))
	if err := db.Connect(dsn); err != nil {
		log.Fatal(err)
	}

	// initialize Gin engine
	engine := gin.Default()
	engine.LoadHTMLGlob("views/*.html")

	// allow cors
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:8081"}
	config.AllowHeaders=  []string{"Access-Control-Allow-Credentials"}
	config.AllowCredentials = true
	engine.Use(cors.New(config))

	// prepare session
	store := cookie.NewStore([]byte("my-secret"))
	engine.Use(sessions.Sessions("user-session", store))

	// routing
	engine.Static("/assets", "./assets")
	engine.GET("/", service.Home)
	engine.GET("/list", service.LoginCheck, service.TaskList)

	taskGroup := engine.Group("/task")
	taskGroup.Use(service.LoginCheck) 
	{
		// taskGroup.GET("/:id", service.ShowTask) // ":id" is a parameter
		// taskGroup.GET("/new", service.NewTaskForm)
		taskGroup.POST("/new", service.RegisterTask)
		// taskGroup.GET("/edit/:id", service.EditTaskForm)
		// taskGroup.POST("/edit/:id", service.UpdateTask)
		taskGroup.PUT("/:id", service.UpdateTask)
		taskGroup.DELETE("/:id", service.DeleteTask)
	}

	// engine.GET("/login", service.LoginForm)
	engine.POST("/login", service.Login)
	engine.GET("/logout", service.Logout)

	// engine.GET("/user/new", service.NewUserForm)
	engine.POST("/user/new", service.RegisterNewUser)

	userGroup := engine.Group("/user")
	userGroup.Use(service.LoginCheck)
	{
		userGroup.GET("/name", service.GetUsername)
		userGroup.PUT("/name", service.ChangeUsername)
		userGroup.PUT("/password", service.ChangePassword)
		userGroup.POST("/delete", service.DeleteUser)
	}

	// start server
	engine.Run(fmt.Sprintf(":%d", port))
}
