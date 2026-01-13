package controllers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"io"
	"bytes"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/likalikali/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	"github.com/likalikali/MagicStreamMovies/Server/MagicStreamMoviesServer/models"
	"github.com/likalikali/MagicStreamMovies/Server/MagicStreamMoviesServer/utils"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"


)
type SFRequest struct {
	Model    string `json:"model"`
	Messages []map[string]string `json:"messages"`
	// 其他你需要的参数
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

type SFResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}


var validate = validator.New()

func GetMovies(client *mongo.Client) gin.HandlerFunc{
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c,100*time.Second)
		defer cancel()
		var movieCollection *mongo.Collection = database.OpenCollection("movies",client)


		var movies []models.Movie

		cursor, err := movieCollection.Find(ctx, bson.M{}) 

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch movies"})
			
		}
		defer cursor.Close(ctx)

		if err = cursor.All(ctx, &movies); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode movies"})
			
		}
		
		c.JSON(http.StatusOK, movies)
	}
}
func GetMovie( client *mongo.Client) gin.HandlerFunc {
	// Get a single movie by its IMDb ID *gin.Context是Gin框架中的上下文对象，包含了HTTP请求和响应的信息，return func(c *gin.Context)是一个闭包匿名函数，作为Gin的处理函数使用
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c, 100*time.Second)
		defer cancel()

		movieId := c.Param("imdb_id")
		if movieId == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Movie ID is required"})
			return 
		}

		var movie models.Movie
		var movieCollection *mongo.Collection = database.OpenCollection("movies",client)

		err := movieCollection.FindOne(ctx, bson.M{"imdb_id": movieId}).Decode(&movie)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Movie not found"})
			return
		}

		c.JSON(http.StatusOK, movie)
	}
}


func AddMovie( client *mongo.Client) gin.HandlerFunc {

	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c, 100*time.Second)
		defer cancel()
		
		var movie models.Movie
		if err := c.ShouldBind(&movie); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}
		if err := validate.Struct(movie); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
			return
		}
		var movieCollection *mongo.Collection = database.OpenCollection("movies",client)

		result, err := movieCollection.InsertOne(ctx, movie)
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add movie"})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}


func AdminReviewUpdate( client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {

		role, err := utils.GetRoleFromContext(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Role not found in context"})
			return
		}
		if role != "ADMIN" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User must be part of the ADMIN role"})
			return
		}

		movieId := c.Param("imdb_id")
		if movieId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Movie ID  required"})
		return
	}
	var req struct{
		AdminReview string `json:"admin_review"`
	}
	var resp struct{
		RankingName string `json:"ranking_name"`
		AdminReview string `json:"admin_review"`

	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return

	}
	sentiment, rankVal, err := GetReviewRanking( req.AdminReview,client,c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting review ranking"})
		return
	}
	filter := bson.M{"imdb_id": movieId}
	update := bson.M{
		"$set": bson.M{
			"admin_review": req.AdminReview,
			"ranking":bson.M{
			"ranking_value": rankVal,
			"ranking_name": sentiment,
			
				},
			},
	
		}
		var ctx, cancel = context.WithTimeout(c, 100*time.Second)
		defer cancel()
		var movieCollection *mongo.Collection = database.OpenCollection("movies",client)

		result, err := movieCollection.UpdateOne(ctx, filter, update)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movie review"})
			return
		}
		if result.MatchedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Movie not found"})
			return
		}
		resp.RankingName = sentiment
		resp.AdminReview = req.AdminReview
		c.JSON(http.StatusOK, resp)
	}
}


func GetReviewRanking(admin_review string,client *mongo.Client,c *gin.Context) (string, int, error) {
	rankings, err := GetRankings(client,c)
	if err != nil {
		return "", 0, err
	}

	// 构造可接受的 ranked prompt
	sentimentDelimited := ""
	for _, ranking := range rankings {
		if ranking.RankingValue != 999 {
			sentimentDelimited += ranking.RankingName + ","
		}
	}
	sentimentDelimited = strings.Trim(sentimentDelimited, ",")

	base_prompt_template := os.Getenv("BASE_PROMPT_TEMPLATE")
	base_prompt := strings.Replace(base_prompt_template, "{rankings}", sentimentDelimited, 1)
	userPrompt := base_prompt + admin_review

	// 从环境读取你的 SiliconFlow API Key
	apiKey := os.Getenv("SILICONFLOW_API_KEY")
	if apiKey == "" {
		return "", 0, errors.New("could not read SILICONFLOW_API_KEY")
	}

	// 构造请求对象
	reqBody := SFRequest{
		Model: "deepseek-ai/DeepSeek-V3.2",
		Messages: []map[string]string{
			{"role": "user", "content": userPrompt},
		},
		Temperature: 0.7,
		MaxTokens:   2048,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, err
	}

	// 发送 HTTP POST 到硅基流动的 API
	clients := http.DefaultClient
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		"https://api.siliconflow.cn/v1/chat/completions", // 官方 REST endpoint
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := clients.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	if resp.StatusCode >= 400 {
		return "", 0, errors.New(string(respBytes))
	}

	var sfResp SFResponse
	err = json.Unmarshal(respBytes, &sfResp)
	if err != nil {
		return "", 0, err
	}

	// 默认取第一个结果
	assistantText := ""
	if len(sfResp.Choices) > 0 {
		assistantText = sfResp.Choices[0].Message.Content
	}

	// 找到 ranking 对应的 ranking value
	rankVal := 0
	for _, ranking := range rankings {
		if ranking.RankingName == assistantText {
			rankVal = ranking.RankingValue
			break
		}
	}
	return assistantText, rankVal, nil
}



func GetRankings(client *mongo.Client,c *gin.Context) ([]models.Ranking, error){
	var rankings []models.Ranking

	var ctx, cancel = context.WithTimeout(c,100*time.Second)
	defer cancel()
	var rankingCollection *mongo.Collection = database.OpenCollection("rankings", client)

	cursor, err := rankingCollection.Find(ctx,  bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err = cursor.All(ctx, &rankings); err != nil {
		return nil, err
	}
	return rankings, nil
}


func GetRecommendedMovies( client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, err := utils.GetUserIdFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User Id not found in context"})
			return
		}
		favourite_genres , err := GetUserFavouriteGenres(userId,client,c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		err = godotenv.Load(".env")
		if err != nil {
			log.Println("Warning: .env file not found")
		}
		var recommendMovieLimitVal int64 = 5

		recommendMovieLimitStr := os.Getenv("RECOMMEND_MOVIE_LIMIT")

		if recommendMovieLimitStr != ""{ 
			recommendMovieLimitVal, _ = strconv.ParseInt(recommendMovieLimitStr, 10, 64)
		}
		findOptions := options.Find()
		findOptions.SetSort(bson.D{{Key:"ranking.ranking_value", Value:1}})
		findOptions.SetLimit(recommendMovieLimitVal)
		filter := bson.M{"genre.genre_name": bson.M{"$in": favourite_genres}}

		var ctx, cancel = context.WithTimeout(c,100*time.Second)
		defer cancel()
		var movieCollection *mongo.Collection = database.OpenCollection("movies",client)

		cursor, err := movieCollection.Find(ctx, filter, findOptions)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching recommended movies"})
			return
		}
		defer cursor.Close(ctx)
		var recommendedMovies []models.Movie
		if err = cursor.All(ctx, &recommendedMovies); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, recommendedMovies)

	}
}

func GetUserFavouriteGenres(userId string, client *mongo.Client,c *gin.Context)([]string, error){
	var ctx, cancel = context.WithTimeout(c,100*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userId}
	projection := bson.M{"favourite_genres.genre_name": 1, "_id": 0}

	opts := options.FindOne().SetProjection(projection)
	var result bson.M

	var userCollection *mongo.Collection = database.OpenCollection("users",client)
	err := userCollection.FindOne(ctx, filter, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {	
			return []string{}, nil
		}
	}
	favGenresArray, ok := result["favourite_genres"].(bson.A)
	if !ok {
		return []string{}, errors.New("Unable to retrieve favourite genres for user")
	}
	var genreNames []string
	for _, item := range favGenresArray {
		if genreMap, ok := item.(bson.D); ok{
			for _, elem := range genreMap {
				if elem.Key == "genre_name" {
					if name, ok := elem.Value.(string); ok {
						genreNames = append(genreNames, name)
					}
				}
			}
		}

	}

	return genreNames, nil
}
func GetGenres(client *mongo.Client) gin.HandlerFunc{
	return func(c *gin.Context){
		var ctx, cancel = context.WithTimeout(c,100*time.Second)
		defer cancel()

		var genres []models.Genre
		var genreCollection *mongo.Collection = database.OpenCollection("genres", client)
		cursor,err := genreCollection.Find(ctx,bson.M{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching movie genres"})
			return
		}
		defer cursor.Close(ctx)
		if err := cursor.All(ctx, &genres); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, genres)

	}
}