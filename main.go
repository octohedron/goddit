package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"github.com/elgs/gojq"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	CLIENT_ID      = "5ao8tf2OzcUFJg"
	CLIENT_SECRET  = "yeRLdTb3oN6giRbbMs7Tmvm5sYk"
	SERVER_ADDRESS = "http://192.168.1.43:9000"
	SERVER_IP      = "192.168.1.43"
	REDIRECT_URI   = SERVER_ADDRESS + "/reddit_callback"
	letterBytes    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits  = 6                    // 6 bits to represent a letter index
	letterIdxMask  = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax   = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
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

const project_root = "/home/vagrant/go/src/github.com/octohedron/chat"

var addr = flag.String("addr", ":9000", "http service address")
var src = rand.NewSource(time.Now().UnixNano())
var users map[string]User
var AuthorizedIps []string

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
	// connect to the database
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	// close the session when done
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	c := session.DB("views").C("chatrooms")
	// find all subreddits
	var Rooms []Chatroom
	err = c.Find(nil).All(&Rooms)
	if err != nil {
		log.Println(err)
		panic(err) // didn't find any rooms, something wrong with the DB
	} else {
		log.Printf("Found chatrooms: %d \n", len(Rooms))
	}
	cookie, err := r.Cookie("goddit")
	/**
	 * Cookie not found or user not logged in
	 */
	if err != nil || users[cookie.Value].Name == "" {
		log.Println(err)
		// respond with forbidden
		template.Must(template.New("403.html").ParseFiles(
			project_root+"/403.html")).Execute(w, "")
	} else {
		log.Printf("User map in memory \n %+v", users[cookie.Value])
		template.Must(
			template.New("chat.html").ParseFiles(
				project_root+"/chat.html")).Execute(w, struct {
			Username  string
			Chatrooms []Chatroom
		}{cookie.Value, Rooms}) // remember to change to name.value!
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	cookie, err := r.Cookie("goddit")
	if err != nil || users[cookie.Value].Name == "" {
		state := getRandomString(8)
		url := "https://ssl.reddit.com/api/v1/authorize?" + "client_id=" +
			CLIENT_ID + "&response_type=code&state=" + state + "&redirect_uri=" +
			REDIRECT_URI + "&duration=temporary&scope=identity"
		template.Must(template.New("index.html").ParseFiles(
			project_root+"/index.html")).Execute(w, struct {
			Url string
		}{url})
	} else {
		log.Println("Found cookie " + cookie.Value)
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
		Name:    "goddit",
		Value:   user.Name,
		Path:    "/",
		Domain:  SERVER_IP,
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
	// log.Println("Getting subreddit data")
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
	// store in mongodb
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	// close the session when done
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	c := session.DB("views").C("chatrooms")
	bulkT := c.Bulk()
	bulkT.Unordered() // Avoid dupes (?)
	// Index
	index := mgo.Index{
		Key:        []string{"name"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	err = c.EnsureIndex(index)
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
	// log.Printf("BODY: \n %s", string(body[:]))
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
	// find the last 150 messages in the room
	err = m.Find(
		bson.M{"chatRoomId": room.Id}).Sort(
		"-timestamp").Limit(150).All(&messageSlice)
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
	// gets the subreddits and stores them in the DB
	go getPopularSubreddits()
	// initialize the user slice
	users = make(map[string]User)
	flag.Parse()
	r := mux.NewRouter()
	hub := newHub()
	go hub.run()
	r.HandleFunc("/", index)
	r.HandleFunc("/chat", chat)
	r.HandleFunc("/reddit_callback", redditCallback)
	r.HandleFunc("/history/{channel}", channelHistory)
	r.HandleFunc("/room/{channel}",
		func(w http.ResponseWriter, r *http.Request) {
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
