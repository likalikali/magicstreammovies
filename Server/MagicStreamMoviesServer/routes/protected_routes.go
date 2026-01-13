package routes

import (
	"github.com/gin-gonic/gin"
	controllers "github.com/likalikali/MagicStreamMovies/Server/MagicStreamMoviesServer/controllers"
	"github.com/likalikali/MagicStreamMovies/Server/MagicStreamMoviesServer/middleware"
	"go.mongodb.org/mongo-driver/v2/mongo"
)
func SetupProtectedRoutes(router *gin.Engine, client *mongo.Client) {
	
	router.Use(middleware.AuthMiddleware())

	router.GET("/movie/:imdb_id", controllers.GetMovie(client))
	router.POST("/addmovie", controllers.AddMovie(client))
	router.GET("/recommendedmovies", controllers.GetRecommendedMovies(client))
	router.PATCH("/updatereview/:imdb_id", controllers.AdminReviewUpdate(client))
}
