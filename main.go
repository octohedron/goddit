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
	"time"
)

const (
	CLIENT_ID     = "5ao8tf2OzcUFJg"
	CLIENT_SECRET = "yeRLdTb3oN6giRbbMs7Tmvm5sYk"
	REDIRECT_URI  = "http://192.168.1.43:9000/reddit_callback"
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
	Access_token string `json:"access_token"`
	Token_type   string `json:"token_type"`
	Expires_in   int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

var serverAddress = "http://192.168.1.43:9000"

var addr = flag.String("addr", ":9000", "http service address")

const project_root = "/home/vagrant/GO/chat"

func serveChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	cookie, err := r.Cookie("chatterbot")
	if err != nil {
		log.Println(err)
	}
	log.Printf("%s", cookie.Value)
	user := users[cookie.Value]
	template.Must(
		template.New("chat.html").ParseFiles(
			project_root+"/chat.html")).Execute(w, struct {
		Username string
	}{user.Name})
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
	template.Must(template.New("index.html").ParseFiles(
		project_root+"/index.html")).Execute(w, struct {
		Url string
	}{url})
}

func serveRedditCallback(w http.ResponseWriter, r *http.Request) {
	err := r.FormValue("error")
	if err != "" {
		log.Println(err)
	}
	authData := getRedditAuth(r.FormValue("code"))
	user := getRedditUserData(authData)
	user.Auth = authData
	// store reddit auth data in the map, Username -> RedditAuth data
	users[user.Name] = *user
	// redirect to the chat
	cookie := &http.Cookie{
		Name:     "chatterbot",
		Value:    user.Name,
		Path:     serverAddress,
		Expires:  time.Now().Add(time.Hour * 720),
		MaxAge:   0,
		Secure:   true,
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, serverAddress+"/chat", 200)
}

func getRedditUserData(auth RedditAuth) *User {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://oauth.reddit.com/api/v1/me", nil)
	if err != nil {
		log.Println(err)
	}
	req.Header.Set("User-agent", "Web 1x83QLDFHequ8w 1.9.3 (by /u/SEND_ME_RARE_PEPES)")
	req.Header.Add("Authorization", "bearer "+auth.Access_token)
	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	defer res.Body.Close()
	user := User{}
	body, err := ioutil.ReadAll(res.Body)
	err = json.Unmarshal(body, &user)
	if err != nil {
		log.Println(err)
	}
	log.Println(string(body[:]))
	return &user
}

func getRedditAuth(code string) RedditAuth {
	client := &http.Client{}
	req, err := http.NewRequest("POST",
		"https://ssl.reddit.com/api/v1/access_token",
		bytes.NewBufferString(
			"grant_type=authorization_code&code="+code+"&redirect_uri="+REDIRECT_URI))
	req.Header.Add("User-agent", "Web 1x83QLDFHequ8w 1.9.3 (by /u/SEND_ME_RARE_PEPES)")
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
	log.Printf("BODY: \n \n %s", string(body[:]))
	err = json.Unmarshal(body, &redditAuth)
	log.Printf("Access token: %s", redditAuth.Access_token)
	return redditAuth
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
