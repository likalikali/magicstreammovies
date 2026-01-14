package main

import (
	"context"
	"fmt"
	"log"
	"time"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/likalikali/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	"github.com/likalikali/MagicStreamMovies/Server/MagicStreamMoviesServer/routes"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func main() {
	router:=gin.Default()

	router.GET("/hello",func(c *gin.Context){
		c.String(200,"Hello, MagicStreamMovies!")
	})

	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Warning: unable to find .env file")
	}


	allowedOrigins := os.Getenv("ALLOW_ORIGINS")

	var origins []string
	if allowedOrigins != "" {
		origins = strings.Split(allowedOrigins, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
			log.Println("Allowed Origin:", origins[i])
		}
	} else {
		origins = []string{"http://localhost:5173"}
		log.Println("Allowed Origin: http://localhost:5173")
	}

	config := cors.Config{}
	config.AllowOrigins = origins
	config.AllowMethods = []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"}
	//config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	config.ExposeHeaders = []string{"Content-Length"}
	config.AllowCredentials = true
	config.MaxAge = 12 * time.Hour

	router.Use(cors.New(config))
	router.Use(gin.Logger())

	var client *mongo.Client = database.Connect()

	if err := client.Ping(context.Background(), nil); err != nil {
		log.Fatalf("Failed to reach server: %v", err)
	}
	defer func() {
		err := client.Disconnect(context.Background())
		if err != nil {
			log.Fatalf("Failed to disconnect from MongoDB: %v", err)
		}

	}()


	routes.SetupUnProtectedRoutes(router,client)
	routes.SetupProtectedRoutes(router,client)

	if err:=router.Run(":8080");err!=nil{
		fmt.Println("Failed to start server:",err)
	}

}
//在Go语言中，*和&是两个非常重要的运算符，它们分别用于指针操作和取地址操作。
//* 是指针运算符，有两种主要用途：

//声明指针类型：表示一个变量是指针类型。

//解引用：获取指针所指向的存储单元的值。

//& 是取地址运算符，用于获取变量的内存地址。