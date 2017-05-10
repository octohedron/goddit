package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/elgs/gojq"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

type User struct {
	Comment_karma    int     `json:"comment_karma"`
	Created          float32 `json:"created"`
	Created_utc      float32 `json:"created_utc"`
	Has_mail         bool    `json:"has_mail"`
	Has_mod_mail     bool    `json:"has_mod_mail"`
	Id               string  `json:"id"`
	Is_gold          bool    `json:"is_gold"`
	Is_mod           bool    `json:"is_mod"`
	Link_karma       int     `json:"link_karma"`
	Over_18          bool    `json:"over_18"`
	Name             string  `bson:"name" json:"name"`
	Level            string  `bson:"level" json:"level"`
	Active           string  `bson:"active" json:"active"`
	Activation_token string  `bson:"activation_token" json:"activation_token"`
	Created_at       string  `bson:"created_at" json:"created_at"`
	Auth             RedditAuth
	IP               string
}

type Chatroom struct {
	Id        bson.ObjectId   `bson:"_id,omitempty" json:"_id,omitempty" inline`
	Name      string          `bson:"name" json:"name"`
	Level     string          `bson:"level" json:"level"`
	Active    string          `bson:"active" json:"active"`
	Timestamp time.Time       `bson:"timestamp,omitempty" json:"timestamp,omitempty"`
	Messages  []bson.ObjectId `bson:"messages,omitempty" json:"messages" inline`
}

type Message struct {
	MessageId    bson.ObjectId `bson:"_id,omitempty" json:"_id,omitempty" inline`
	Level        int           `bson:"level" json:"level"`
	Text         string        `bson:"text" json:"text"`
	UserName     string        `bson:"name" json:"name"`
	ChatRoomName string        `bson:"room_name" json:"room_name"`
	ChatRoomId   bson.ObjectId `bson:"chatRoomId,omitempty" json:"chatRoomId,omitempty"`
	Timestamp    time.Time     `bson:"timestamp,omitempty" json:"timestamp,omitempty"`
}

type RedditAuth struct {
	Access_token string `json:"access_token"`
	Token_type   string `json:"token_type"`
	Expires_in   int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

type MongoDBConnections struct {
	Session   *mgo.Session
	Messages  *mgo.Collection
	Chatrooms *mgo.Collection
}

// env
var CLIENT_ID = "YOUR_APP_ID"
var CLIENT_SECRET = "YOUR_APP_SECRET"
var DOMAIN = "192.168.1.43"
var GPORT = "9000"
var REDIRECT_URI = SERVER_ADDRESS + "/reddit_callback"
var SERVER_ADDRESS = "http://192.168.1.43:9000"
var PROJ_ROOT = "/home/ubuntu/go/src/github.com/octohedron/goddit"
var COOKIE_NAME = "goddit"

// mem
var users map[string]User
var AuthorizedIps []string
var Mongo *MongoDBConnections
var MessageChannel chan []byte

func newMongoDBConnections(session *mgo.Session, m *mgo.Collection,
	c *mgo.Collection) *MongoDBConnections {
	return &MongoDBConnections{
		Session:   session,
		Messages:  m,
		Chatrooms: c,
	}
}

/**
 * Serve the /chat route
 *
 * Checks the cookie in the request, if the cookie is not found or the value
 * is not found in the server memory map, then return 403.
 */
func chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	var Rooms []Chatroom
	err := Mongo.Chatrooms.Find(nil).All(&Rooms)
	if err != nil {
		log.Println(err)
		panic(err) // didn't find any rooms, something wrong with the DB
	}
	cookie, err := r.Cookie(COOKIE_NAME)
	/**
	 * Cookie not found or user not logged in
	 */
	if err != nil || users[cookie.Value].Name == "" {
		// respond with forbidden
		template.Must(template.New("403.html").ParseFiles(
			PROJ_ROOT+"/403.html")).Execute(w, "")
	} else {
		template.Must(
			template.New("chat.html").ParseFiles(
				PROJ_ROOT+"/chat.html")).Execute(w, struct {
			CookieName string
			ServerAddr string
			Username   string
			Chatrooms  []Chatroom
		}{COOKIE_NAME, SERVER_ADDRESS, users[cookie.Value].Name, Rooms})
	}
}

// If the user is already logged in, redirect to the /chat
func index(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	cookie, err := r.Cookie(COOKIE_NAME)
	if err != nil || users[cookie.Value].Name == "" {
		state := getRandomString(8)
		url := "https://ssl.reddit.com/api/v1/authorize?" + "client_id=" +
			CLIENT_ID + "&response_type=code&state=" + state + "&redirect_uri=" +
			REDIRECT_URI + "&duration=temporary&scope=identity"
		template.Must(template.New("index.html").ParseFiles(
			PROJ_ROOT+"/index.html")).Execute(w, struct {
			Url string
		}{url})
	} else {
		http.Redirect(w, r, SERVER_ADDRESS+"/chat", 302)
	}
}

func redditCallback(w http.ResponseWriter, r *http.Request) {
	err := r.FormValue("error")
	if err != "" {
		log.Println(err)
	}
	authData := getRedditAuth(r.FormValue("code"))
	user := getRedditUserData(authData)
	// failure to get data
	if user.Name == "" {
		http.Error(w, "Not authorized", 403)
		return
	}
	clientIp := strings.Split(r.RemoteAddr, ":")[0]
	AuthorizedIps = append(AuthorizedIps, clientIp)
	user.Auth = authData
	user.IP = clientIp
	// store reddit auth data in the map, Username -> RedditAuth data
	users[user.Name] = *user
	expire := time.Now().AddDate(0, 0, 1)
	cookie := &http.Cookie{
		Expires: expire,
		MaxAge:  86400,
		Name:    COOKIE_NAME,
		Value:   user.Name,
		Path:    "/",
		Domain:  DOMAIN,
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, SERVER_ADDRESS+"/chat", 302)
}

func getRedditUserData(auth RedditAuth) *User {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://oauth.reddit.com/api/v1/me", nil)
	if err != nil {
		log.Println(err)
	}
	req.Header.Set("User-agent",
		"Web 1x83QLDFHequ8w 1.9.3 (by /u/SEND_ME_RARE_PEPES)")
	req.Header.Add("Authorization", "bearer "+auth.Access_token)
	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	user := User{}
	body, err := ioutil.ReadAll(res.Body)
	err = json.Unmarshal(body, &user)
	if err != nil {
		log.Println(err)
	}
	return &user
}

func getPopularSubreddits() {
	client := &http.Client{}
	req, err := http.NewRequest("GET",
		"https://www.reddit.com/subreddits/popular/.json", nil)
	if err != nil {
		log.Println(err)
	}
	req.Header.Set("User-agent",
		"Web 1x83QLDFHequ8w 1.9.3 (by /u/SEND_ME_RARE_PEPES)")
	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	body, err := ioutil.ReadAll(res.Body)
	parser, err := gojq.NewStringQuery(string(body[:]))
	if err != nil {
		log.Println(err)
		return
	}

	bulkT := Mongo.Chatrooms.Bulk()
	bulkT.Unordered() // Avoid dupes (?)
	// Index
	index := mgo.Index{
		Key:        []string{"name"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	err = Mongo.Chatrooms.EnsureIndex(index)
	for i := 0; i < 25; i++ {
		name, err := parser.Query("data.children.[" +
			strconv.Itoa(i) + "].data.display_name")
		if err != nil {
			log.Println(err)
		}
		subreddit := Chatroom{
			Id:        bson.NewObjectId(),
			Name:      name.(string),
			Level:     "0",
			Active:    "1",
			Timestamp: time.Now(),
		}
		bulkT.Insert(subreddit)
	}
	_, err = bulkT.Run()
	if err != nil {
		log.Println("Found duplicate subreddits...")
	}
}

func getRedditAuth(code string) RedditAuth {
	client := &http.Client{}
	req, err := http.NewRequest("POST",
		"https://ssl.reddit.com/api/v1/access_token",
		bytes.NewBufferString(
			"grant_type=authorization_code&code="+code+"&redirect_uri="+REDIRECT_URI))
	req.Header.Add("User-agent",
		"Web 1x83QLDFHequ8w 1.9.3 (by /u/SEND_ME_RARE_PEPES)")
	encoded := base64.StdEncoding.EncodeToString(
		[]byte(CLIENT_ID + ":" + CLIENT_SECRET))
	req.Header.Add("Authorization", "Basic "+encoded)
	res, err := client.Do(req)
	defer res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	redditAuth := RedditAuth{}
	body, err := ioutil.ReadAll(res.Body)
	err = json.Unmarshal(body, &redditAuth)
	return redditAuth
}

/**
 * Load the previous messages from this channel from the database
 */
func channelHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := r.Header.Get("name")
	if name == "" || users[name].Name == "" {
		http.Error(w, "Forbidden", 403)
		return
	}
	var room Chatroom
	// find the chatroom at this request
	err := Mongo.Chatrooms.Find(bson.M{"name": vars["channel"]}).One(&room)
	if err != nil { // channel not found
		log.Printf("Creating new channel: %s ...", vars["channel"])
		// create new channel
		room.Id = bson.NewObjectId()
		room.Name = vars["channel"]
		room.Level = "0"
		room.Active = "true"
		err := Mongo.Chatrooms.Insert(room)
		if err != nil {
			log.Println(err)
		} else {
			// new welcome message for the room
			welcomeMessage := Message{
				MessageId:    bson.NewObjectId(),
				Text:         "Welcome to the new " + vars["channel"] + " chat",
				ChatRoomName: vars["channel"],
				UserName:     "Moderator",
				ChatRoomId:   room.Id,
				Timestamp:    time.Now(),
				Level:        1, // level = power
			}
			room.Messages = append(room.Messages, welcomeMessage.MessageId)
			// insert the new welcome message into the messages
			// collection, with this chatroom id and the user id
			err = Mongo.Messages.Insert(welcomeMessage)
			if err != nil {
				panic(err) // error inserting
			}
		}
	}
	// initialize a slice of size messageAmount to store the messages
	var messageSlice []Message
	// find the last 150 messages in the room
	err = Mongo.Messages.Find(
		bson.M{"chatRoomId": room.Id}).Sort(
		"-timestamp").Limit(150).All(&messageSlice)
	js, err := json.Marshal(messageSlice)
	if err != nil {
		panic(err)
	}
	// serve
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

/**
 * Channel to save messages to the database
 */
func saveMessages(m *chan []byte) {
	for {
		message, ok := <-*m
		if !ok {
			log.Println("Error when trying to save")
			return
		}
		saveMessage(&message)
	}
}

func saveMessage(msg *[]byte) {
	message := Message{}
	err := json.Unmarshal(*msg, &message)
	message.MessageId = bson.NewObjectId()
	message.Timestamp = time.Now()
	var room Chatroom
	// find the chatroom at this request
	err = Mongo.Chatrooms.Find(bson.M{"name": message.ChatRoomName}).One(&room)
	if err != nil { // channel not found
		// create new channel
		room.Name = message.ChatRoomName
		room.Level = "0"
		room.Active = "true"
		room.Id = bson.NewObjectId()
		err := Mongo.Chatrooms.Insert(room)
		if err != nil {
			log.Println(err)
		} else {
			room.Messages = append(room.Messages, message.MessageId)
		}
	}
	// construct the new message
	message.ChatRoomId = room.Id
	// insert the message into the messages collection, with this chatroom
	// and the user id
	err = Mongo.Messages.Insert(message)
	if err != nil {
		log.Println(err)
		// panic(err) // error inserting
	}
	var messageSlice []Message
	var bsonMessageSlice []bson.ObjectId
	// find all the messages that have this room as chatRoomId
	err = Mongo.Messages.Find(
		bson.M{"chatRoomId": room.Id}).Sort("-timestamp").All(&messageSlice)
	if err != nil {
		panic(err)
	}
	if len(messageSlice) > 0 {
		if err != nil {
			log.Println(err)
		}
		// if there is no messages it won't enter the loop
		for i := 0; i < len(messageSlice); i++ {
			bsonMessageSlice = append(bsonMessageSlice, messageSlice[i].MessageId)
		}
	}
	// append the new message
	bsonMessageSlice = append(bsonMessageSlice, message.MessageId)
	// update the room with the new messsage
	err = Mongo.Chatrooms.Update(bson.M{"_id": room.Id},
		bson.M{"$set": bson.M{"messages": bsonMessageSlice}})
	if err != nil {
		panic(err)
	}
}

func main() {
	// env variables
	CLIENT_ID = os.Getenv("APPID")
	CLIENT_SECRET = os.Getenv("APPSECRET")
	SERVER_ADDRESS = os.Getenv("GODDITADDR")
	DOMAIN = os.Getenv("GODDITDOMAIN")
	GPORT = os.Getenv("GPORT")
	COOKIE_NAME = os.Getenv("GCOOKIE")
	REDIRECT_URI = SERVER_ADDRESS + "/reddit_callback"
	PROJ_ROOT, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Println(err)
	}
	// connect to the database
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	session.SetMode(mgo.Monotonic, true)
	Mongo = newMongoDBConnections(
		session, session.DB("views").C("messages"),
		session.DB("views").C("chatrooms"))
	MessageChannel = make(chan []byte, 256)
	// a goroutine for saving messages
	go saveMessages(&MessageChannel)
	go getPopularSubreddits()
	//for keeping track of users in memory
	users = make(map[string]User)
	r := mux.NewRouter()
	hub := newHub()
	go hub.run()
	log.Println(PROJ_ROOT + "/icons")
	r.HandleFunc("/", index)
	r.HandleFunc("/chat", chat)
	r.HandleFunc("/reddit_callback", redditCallback)
	r.HandleFunc("/history/{channel}", channelHistory)
	r.HandleFunc("/room/{channel}",
		func(w http.ResponseWriter, r *http.Request) {
			serveWs(hub, w, r)
		})
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(PROJ_ROOT+"/icons"))))
	srv := &http.Server{
		Handler:      r,
		Addr:         ":" + GPORT,
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
	}
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func getRandomString(n int) string {
	b := make([]byte, n)
	src := rand.NewSource(time.Now().UnixNano())
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
