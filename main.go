package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const (
	CLIENT_ID     = "1x83QLDFHequ8w"
	CLIENT_SECRET = "A9R-RZ0kuflGvhR0LJoRYVa9vRE"
	REDIRECT_URI  = "http://192.168.1.43:9000/reddit_callback"
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

type User struct {
	Name             string `bson:"name" json:"name"`
	Level            string `bson:"level" json:"level"`
	Active           string `bson:"active" json:"active"`
	Activation_token string `bson:"activation_token" json:"activation_token"`
	Created_at       string `bson:"created_at" json:"created_at"`
	Auth             RedditAuth
}

var users map[string]User

type Chatroom struct {
	Id         bson.ObjectId   `bson:"_id,omitempty" json:"_id,omitempty" inline`
	Name       string          `bson:"name" json:"name"`
	Level      string          `bson:"level" json:"level"`
	Active     string          `bson:"active" json:"active"`
	Created_at string          `bson:"created_at" json:"created_at"`
	Messages   []bson.ObjectId `bson:"messages,omitempty" json:"messages" inline`
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
	access_token string
	token_type   string
	expires_in   int
	scope        string
}

var serverAddress = "http://192.168.1.43:9000/"

var addr = flag.String("addr", ":9000", "http service address")

const project_root = "/home/vagrant/GO/chat"

/**
 * Chat channel
 */
func serveChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	template.Must(
		template.New("chat.html").ParseFiles(
			project_root+"/chat.html")).Execute(w, struct {
		Token string
	}{accessToken})
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	state := getRandomString(8)
	url := "https://ssl.reddit.com/api/v1/authorize?" + "client_id=" + CLIENT_ID +
		"&response_type=code&state=" + state + "&redirect_uri=" +
		REDIRECT_URI + "&duration=temporary&scope=identity"
	log.Println(url)
	// t, _ := template.ParseFiles(project_root + "/index.html")
	template.Must(template.New("index.html").ParseFiles(
		project_root+"/index.html")).Execute(w, struct {
		Url string
	}{url})
}

func serveRedditCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	log.Println(r.URL)
	err := r.FormValue("error")
	log.Println("err: " + err)
	state := r.FormValue("state")
	log.Println("state: " + state)
	code := r.FormValue("code")
	log.Println("code: " + code)
	accessToken := getToken(code)

	// store reddit auth data in the map
	log.Println("accessToken: " + accessToken)
	// redirect to the chat
	http.Redirect(w, r, serverAddress, 200)
}

func getToken(code string) string {
	client := &http.Client{}
	req, err := http.NewRequest("POST",
		"https://ssl.reddit.com/api/v1/access_token",
		bytes.NewBufferString(
			"grant_type=authorization_code&code="+code+"&redirect_uri="+REDIRECT_URI))
	req.Header.Add("User-agent", "Reddit-chatterbots")
	encoded := base64.StdEncoding.EncodeToString(
		[]byte(CLIENT_ID + ":" + CLIENT_SECRET))
	// authString = base64.encodestring()
	// headers = {'Authorization':"Basic %s" % authString}
	log.Println(encoded)
	req.Header.Add("Authorization", "Basic "+encoded)
	res, err := client.Do(req)
	defer res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	redditAuth := RedditAuth{}
	err = json.Unmarshal(res.Body, &redditAuth)
	log.Printf("%v", redditAuth)
	return string(redditAuth.access_token)
}

/**
 * Load the previous messages from this channel from the database
 */
func serveChannelHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	// connect to the database
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	// close the session when done
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	c := session.DB("views").C("chatrooms")
	m := session.DB("views").C("messages")
	var room Chatroom
	// find the chatroom at this request
	err = c.Find(bson.M{"name": vars["channel"]}).One(&room)
	if err != nil { // channel not found
		log.Printf("Creating new channel: %s ...", vars["channel"])
		// create new channel
		room.Id = bson.NewObjectId()
		room.Name = vars["channel"]
		room.Level = "0"
		room.Active = "true"
		err := c.Insert(room)
		if err != nil {
			log.Println(err)
		} else {
			// new welcome message for the room
			welcomeMessage := Message{
				MessageId:    bson.NewObjectId(),
				Text:         "Welcome to this new channel",
				ChatRoomName: vars["channel"],
				UserName:     "Moderator",
				ChatRoomId:   room.Id,
				Timestamp:    time.Now(),
				Level:        1, // level = power
			}
			room.Messages = append(room.Messages, welcomeMessage.MessageId)
			// insert the new welcome message into the messages
			// collection, with this chatroom id and the user id
			err = m.Insert(welcomeMessage)
			if err != nil {
				panic(err) // error inserting
			}
		}
	} else { // channel found
		log.Printf("Found history for channel: %s \n", vars["channel"])
	}
	// initialize a slice of size messageAmount to store the messages
	var messageSlice []Message
	// find all the messages in this chatroom
	err = m.Find(bson.M{"chatRoomId": room.Id}).Sort("-timestamp").All(&messageSlice)
	log.Printf("Messages in the channel: %d \n", len(messageSlice))
	// get json
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

func main() {
	// initialize the user slice
	users = make(map[string]User)
	flag.Parse()
	r := mux.NewRouter()
	hub := newHub()
	go hub.run()
	r.HandleFunc("/", serveIndex)
	r.HandleFunc("/chat", serveChat)
	r.HandleFunc("/reddit_callback", serveRedditCallback)
	r.HandleFunc("/history/{channel}", serveChannelHistory)
	r.HandleFunc("/room/{channel}", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	srv := &http.Server{
		Handler: r,
		Addr:    *addr,
		// Enforcing timeouts
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

var src = rand.NewSource(time.Now().UnixNano())

func getRandomString(n int) string {
	b := make([]byte, n)
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
