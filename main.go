package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

type User struct {
	ID   primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Name string             `json:"name,omitempty" bson:"name,omitempty"`
	Mail string             `json:"mail,omitempty" bson:"mail,omitempty"`
	Pwd  string             `json:"pwd,omitempty" bson:"pwd,omitempty"`
}

type Post struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Uid       string             `json:"uid,omitempty" bson:"uid, omitempty"`
	Caption   string             `json:"caption,omitempty" bson:"caption,omitempty"`
	ImgUrl    string             `json:"imgurl,omitempty" bson:"imgurl,omitempty"`
	Timestamp time.Time          `json:"timestamp,omitempty" bson:"timestamp,omitempty"`
}

func CreateUserEndpoint(response http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodPost {
		response.Header().Set("content-type", "application/json")
		user := &User{}
		err := json.NewDecoder(request.Body).Decode(user)
		if err != nil {
			// If there is something wrong with the request body, return a 400 status
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		key := []byte("the-key-has-to-be-32-bytes-long!")
		hashedPassword, err := encrypt([]byte(user.Pwd), key)
		collection := client.Database("appointy").Collection("users")
		oneDoc := User{
			Name: user.Name,
			Pwd:  string(hashedPassword),
			Mail: user.Mail,
		}
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		result, inserterr := collection.InsertOne(ctx, oneDoc)
		if inserterr != nil {
			response.WriteHeader(http.StatusNotAcceptable)
		}
		json.NewEncoder(response).Encode(result)
	}

}

func GetPersonEndpoint(response http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodGet {
		response.Header().Set("content-type", "application/json")
		// params := mux.Vars(request)
		query := request.URL.Query()
		// u, err := url.Parse(ur)
		// if err != nil {
		// 	log.Fatal(err)
		// }
		uid := query.Get("uid")
		id, _ := primitive.ObjectIDFromHex(uid)
		fmt.Println("the uid is", uid)
		var user User
		collection := client.Database("appointy").Collection("users")
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
		err := collection.FindOne(ctx, User{ID: id}).Decode(&user)
		if err != nil {
			response.WriteHeader(http.StatusInternalServerError)
			response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
			return
		}
		json.NewEncoder(response).Encode(user)

	}

}

func GetUsersEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("content-type", "application/json")
	var users []User
	collection := client.Database("appointy").Collection("users")
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var user User
		cursor.Decode(&user)
		users = append(users, user)
	}
	if err := cursor.Err(); err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	json.NewEncoder(response).Encode(users)
}

func encrypt(plaintext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func CreatePostEndpoint(response http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodPost {
		response.Header().Set("content-type", "application/json")
		var post Post
		_ = json.NewDecoder(request.Body).Decode(&post)
		post.Timestamp = time.Now()
		collection := client.Database("appointy").Collection("posts")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		result, _ := collection.InsertOne(ctx, post)
		json.NewEncoder(response).Encode(result)
	}

}

func GetPostEndpoint(response http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodGet {
		response.Header().Set("content-type", "application/json")
		query := request.URL.Query()

		pid := query.Get("pid")
		id, _ := primitive.ObjectIDFromHex(pid)
		fmt.Println("the uid is", pid)
		var post Post
		collection := client.Database("appointy").Collection("posts")
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
		err := collection.FindOne(ctx, Post{ID: id}).Decode(&post)
		if err != nil {
			response.WriteHeader(http.StatusInternalServerError)
			response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
			return
		}
		json.NewEncoder(response).Encode(post)

	}

}

func GetPostByUidEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("content-type", "application/json")
	var posts []Post
	query := request.URL.Query()
	uid := query.Get("uid")
	collection := client.Database("appointy").Collection("posts")
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var post Post
		cursor.Decode(&post)
		//fmt.Println("post.uid", post.Uid)
		//post is appended to list only if it has been posted by the required user id
		if post.Uid == uid {
			posts = append(posts, post)
		}

	}
	if err := cursor.Err(); err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	json.NewEncoder(response).Encode(posts)
}

func main() {
	fmt.Println("Starting the application...")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, _ = mongo.Connect(ctx, clientOptions)

	mux := http.NewServeMux()
	userbyid := http.HandlerFunc(GetPersonEndpoint)
	mux.Handle("/user/", userbyid)
	createuser := http.HandlerFunc(CreateUserEndpoint)
	mux.Handle("/user", createuser)
	listusers := http.HandlerFunc(GetUsersEndpoint)
	mux.Handle("/users", listusers)
	createpost := http.HandlerFunc(CreatePostEndpoint)
	mux.Handle("/posts", createpost)
	postbypid := http.HandlerFunc(GetPostEndpoint)
	mux.Handle("/posts/", postbypid)
	postbyuid := http.HandlerFunc(GetPostByUidEndpoint)
	mux.Handle("/posts/users/", postbyuid)
	// router.HandleFunc("/person/{id}", GetPersonEndpoint).Methods("GET")
	http.ListenAndServe(":12345", mux)
}
