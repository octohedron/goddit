package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/elgs/gojq"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
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
	Name             string  `json:"name"`
	Level            string  `json:"level"`
	Active           string  `json:"active"`
	Activation_token string  `json:"activation_token"`
	Created_at       string  `json:"created_at"`
	Auth             RedditAuth
	IP               string
}

type Message struct {
	Level        int       `json:"level"`
	Text         string    `json:"text"`
	UserName     string    `json:"name"`
	ChatRoomName string    `json:"room_name"`
	Timestamp    time.Time `json:"timestamp,omitempty"`
}

type DBMessage struct {
	Room string
	Json []byte
}

type RedditAuth struct {
	Access_token string `json:"access_token"`
	Token_type   string `json:"token_type"`
	Expires_in   int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

// env
var CLIENT_ID = "YOUR_APP_ID"
var CLIENT_SECRET = "YOUR_APP_SECRET"
var DOMAIN = "192.168.1.43"
var GPORT = "9000"
var REDIRECT_URI = SERVER_ADDRESS + "/reddit_callback"
var SERVER_ADDRESS = "http://192.168.1.43:9000"
var COOKIE_NAME = "goddit"
var PROJ_ROOT = ""

// mem
var users map[string]User
var AuthorizedIps []string
var MessageChannel chan DBMessage

// Declare a global variable to store the Redis connection pool.
var POOL *redis.Pool

func init() {
	// env variables
	CLIENT_ID = os.Getenv("APPID")
	CLIENT_SECRET = os.Getenv("APPSECRET")
	SERVER_ADDRESS = os.Getenv("GODDITADDR")
	DOMAIN = os.Getenv("GODDITDOMAIN")
	GPORT = os.Getenv("GPORT")
	COOKIE_NAME = os.Getenv("GCOOKIE")
	REDIRECT_URI = SERVER_ADDRESS + "/reddit_callback"
	// set root directory
	ROOT, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	PROJ_ROOT = ROOT
	// Establish a pool of 5 Redis connections to the Redis server
	POOL = newPool("localhost:6379")
}

func newPool(addr string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     5,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", addr) },
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
	conn := POOL.Get()
	defer conn.Close()
	Rooms, err := redis.Strings(conn.Do("SMEMBERS", "rooms"))
	if err != nil {
		log.Println(err)
	}
	sort.Strings(Rooms)
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
			Chatrooms  []string
		}{COOKIE_NAME, SERVER_ADDRESS, users[cookie.Value].Name, Rooms})
	}
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
	conn := POOL.Get()
	defer conn.Close()
	for i := 0; i < 25; i++ {
		name, err := parser.Query("data.children.[" +
			strconv.Itoa(i) + "].data.display_name")
		// add subreddit as room to redis
		_, err = conn.Do("SADD", "rooms", name)
		if err != nil {
			log.Println(err)
		}
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
		url := "https://ssl.reddit.com/api/v1/authorize.compact?" + "client_id=" +
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
	// failure to get data, redirect to /
	if user.Name == "" {
		http.Redirect(w, r, SERVER_ADDRESS+"/", 302)
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
	var err error
	vars := mux.Vars(r)
	name := r.Header.Get("name")
	if name == "" || users[name].Name == "" {
		http.Error(w, "Forbidden", 403)
		return
	}
	// Fetch a single Redis connection from the pool.
	conn := POOL.Get()
	defer conn.Close()
	// first get list size
	llength, err := redis.Int(conn.Do("LLEN", vars["channel"]))
	result, err := redis.Strings(conn.Do("LRANGE", vars["channel"], llength-150, llength))
	if err != nil {
		log.Println(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("[" + strings.Join(result[:], ",") + "]"))
}

/**
 * Channel to save messages to the database
 */
func saveMessages(m *chan DBMessage) {
	for {
		message, ok := <-*m
		if !ok {
			log.Println("Error when trying to save")
			return
		}
		saveMessage(&message)
	}
}

func saveMessage(msg *DBMessage) {
	var err error
	conn := POOL.Get()
	if err != nil {
		log.Println(err)
	}
	defer conn.Close()
	_, err = conn.Do("RPUSH", msg.Room, msg.Json)
	if err != nil {
		log.Println(err)
	}
}

func main() {
	MessageChannel = make(chan DBMessage, 256)
	// a goroutine for saving messages
	go saveMessages(&MessageChannel)
	go getPopularSubreddits()
	//for keeping track of users in memory
	users = make(map[string]User)
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
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(PROJ_ROOT+"/icons"))))
	srv := &http.Server{
		Handler:      r,
		Addr:         ":" + GPORT,
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
	}
	err := srv.ListenAndServe()
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
