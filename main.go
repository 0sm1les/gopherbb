package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/0sm1les/gopherbb/auth"
	"github.com/0sm1les/gopherbb/models"
	"github.com/0sm1les/gopherbb/querydb"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var store = sessions.NewCookieStore([]byte(os.Getenv("dkyc_cookie_key")))

var Categories []models.Category

var Sections = make(map[string]string)

var md = goldmark.New(goldmark.WithExtensions(extension.GFM))

var logger zerolog.Logger

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.With().Caller().Logger()

	gopherbb_main_log_file, supplied := os.LookupEnv("gopherbb_main_log")
	if !supplied {
		log.Fatal().Msg("env variable 'gopherbb_main_log' is not set")
	}

	mainLog, err := os.OpenFile(gopherbb_main_log_file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		log.Fatal().Err(err)
	}
	defer mainLog.Close()

	gopherbb_gin_log_file, supplied := os.LookupEnv("gopherbb_gin_log")
	if !supplied {
		log.Fatal().Msg("env variable 'gopherbb_gin_log' is not set")
	}

	gin.DisableConsoleColor()
	ginLog, err := os.OpenFile(gopherbb_gin_log_file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		log.Fatal().Err(err)
	}
	defer ginLog.Close()

	consoleLog, _ := os.LookupEnv("gopherbb_console_log")
	if consoleLog == "false" {
		log.Info().Msg(fmt.Sprintf("printing logs to files: %s, %s", gopherbb_main_log_file, gopherbb_gin_log_file))
		logger = zerolog.New(mainLog).With().Caller().Logger()
		gin.DefaultWriter = io.MultiWriter(ginLog)
	} else if consoleLog == "true" {
		log.Info().Msg("printing log to console")
		logger = zerolog.New(os.Stdout).With().Caller().Logger()
		gin.DefaultWriter = io.MultiWriter(os.Stdout)
	} else {
		log.Fatal().Msg("env variable 'gopherbb_console_log' is not supplied or incorrect")
	}

	file_cf, supplied := os.LookupEnv("gopherbb_conf")
	if !supplied {
		logger.Fatal().Msg("env variable 'gopherbb_conf' is not set")
	}
	readConf(file_cf)

	salt, supplied := os.LookupEnv("gopherbb_salt")
	if !supplied {
		logger.Fatal().Msg("env variable 'gopherbb_salt' is not set")
	}
	auth.SetSalt(salt)

	_, supplied = os.LookupEnv("gopherbb_cookie_key")
	if !supplied {
		logger.Fatal().Msg("env variable 'gopherbb_cookie_key' is not set")
	}

	pg_creds, supplied := os.LookupEnv("gopherbb_postgres_creds")
	if !supplied {
		logger.Fatal().Msg("env variable 'gopherbb_postgres_creds' is not set")
	}

	pg_addr, supplied := os.LookupEnv("gopherbb_postgres_addr")
	if !supplied {
		logger.Fatal().Msg("env variable 'gopherbb_postgres_addr' is not set")
	}

	pg_db, supplied := os.LookupEnv("gopherbb_postgres_db")
	if !supplied {
		logger.Fatal().Msg("env variable 'gopherbb_postgres_db' is not set")
	}

	if err := querydb.Connect(pg_creds, pg_addr, pg_db); err != nil {
		logger.Fatal().Err(err)
	}

	router := gin.Default()

	router.NoRoute(func(c *gin.Context) {
		index(c)
	})

	router.Static("/pictures", "html/user_pictures")
	router.LoadHTMLGlob("html/*.html")
	router.StaticFile("/custom.css", "./html/static/custom.css")
	router.StaticFile("/DroidSansMono.ttf", "./html/static/DroidSansMono.ttf")

	router.GET("/", index)
	router.GET("/login", login)
	router.POST("/login", login)
	router.GET("/register", register)
	router.POST("/register", register)
	router.GET("/logout", logout)
	router.GET("/search", search)

	router.GET("/user/settings", settings)
	router.POST("/user/settings/:setting", settings)
	router.GET("/user/:user", profile)
	router.GET("/user/:user/posts", posts)
	router.GET("/user/drafts", drafts)
	router.GET("/user/likes", likes)
	router.GET("/user/notifications", notifications)

	router.GET("/editor", editor)
	router.GET("/editor/:id", editor)

	router.POST("/editor/render", render)

	router.POST("/editor/save", save)
	router.POST("/editor/:id/save", save)

	router.POST("/editor/post", post)
	router.POST("/editor/:id/post", post)

	router.GET("/delete/post/:pid", deletePost)
	router.GET("/delete/reply/:cid", deleteReply)

	router.GET("/section/:section", section)
	router.GET("/section/:section/mostliked", mostLiked)
	router.GET("/section/:section/newest", newest)
	router.GET("/section/:section/:id/:title", viewPost)

	router.GET("/reply/:pid/comment/:cid", reply)
	router.POST("/reply/:pid/comment/:cid", reply)
	router.GET("/reply/:pid", reply)
	router.POST("/reply/:pid", reply)

	router.GET("/like/:pid", like)

	router.Run("localhost:8080")
}

func readConf(conf_file string) {
	data, err := ioutil.ReadFile(conf_file)
	if err != nil {
		log.Fatal().Err(err)
	}
	var conf []models.Category
	err = json.Unmarshal(data, &conf)
	if err != nil {
		log.Fatal().Err(err)
	}
	Categories = conf

	for i := 0; i < len(Categories); i++ {
		for j := 0; j < len(Categories[i].Sections); j++ {
			Sections[Categories[i].Sections[j].Id] = Categories[i].Sections[j].Section
		}
	}
}

func validateSection(sectionId string) (models.Section, error) {
	if val, ok := Sections[sectionId]; ok {
		return models.Section{sectionId, val}, nil
	}
	return models.Section{}, errors.New("section does not exist")
}

func randomString() string {
	var randbytes []byte
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 10; i++ {
		randbytes = append(randbytes, byte(rand.Intn(125)))
	}
	return hex.EncodeToString(randbytes)
}

func initsession(c *gin.Context) error {
	session, _ := store.Get(c.Request, "session")
	if session.IsNew {
		session.Values["id"] = int32(-1)
		if err := session.Save(c.Request, c.Writer); err != nil {
			return err
		}
	}
	return nil
}

func authsesssion(id int32, c *gin.Context) error {
	session, _ := store.Get(c.Request, "session")
	session.Values["id"] = id
	if err := session.Save(c.Request, c.Writer); err != nil {
		return err
	}
	return nil
}

func deauthsession(user_id int32, c *gin.Context) error {
	session, _ := store.Get(c.Request, "session")
	session.Values["id"] = int32(-1)
	if err := session.Save(c.Request, c.Writer); err != nil {
		return err
	}
	return nil
}

func index(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)

	recentPosts, err := querydb.RecentPosts()
	if err != nil {
		logger.Error().Err(err).Msg("")
		return
	}
	for i := 0; i < len(recentPosts); i++ {
		recentPosts[i].User, err = querydb.GetUser(recentPosts[i].Uid)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}
	}
	if uid != -1 {
		userinfo, _ := querydb.Userinfo(uid)
		html := template.Must(template.ParseFiles("html/auth_header.html", "html/index.html", "html/footer.html"))
		html.ExecuteTemplate(c.Writer, "html/auth_header.html", gin.H{"Title": "Index", "Userinfo": userinfo})
		html.ExecuteTemplate(c.Writer, "html/index.html", gin.H{"Categories": Categories, "Recentposts": recentPosts})
		html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
	} else {
		html := template.Must(template.ParseFiles("html/unauth_header.html", "html/index.html", "html/footer.html"))
		html.ExecuteTemplate(c.Writer, "html/unauth_header.html", gin.H{"Title": "Index"})
		html.ExecuteTemplate(c.Writer, "html/index.html", gin.H{"Categories": Categories, "Recentposts": recentPosts})
		html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
	}
}

func login(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)
	if uid == -1 {
		if c.Request.Method == "GET" {
			html := template.Must(template.ParseFiles("html/unauth_header.html", "html/login.html", "html/footer.html"))
			html.ExecuteTemplate(c.Writer, "html/unauth_header.html", gin.H{"Title": "Login"})
			html.ExecuteTemplate(c.Writer, "html/login.html", nil)
			html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
		} else if c.Request.Method == "POST" {
			username := c.PostForm("username")
			password := c.PostForm("password")
			var inputErrors []string

			verified_user, err := auth.ValidateUser(username)
			if err != nil {
				inputErrors = append(inputErrors, err.Error())
			}

			verified_pass, err := auth.ValidatePassword(password)
			if err != nil {
				inputErrors = append(inputErrors, err.Error())
			}

			if len(inputErrors) == 0 {
				user_id, err := querydb.Authenticate(verified_user, auth.Hashpassword(verified_pass))
				if err != nil {
					inputErrors = append(inputErrors, err.Error())
				} else {
					authsesssion(user_id, c)
					index(c)
				}
			}
			if len(inputErrors) != 0 {
				html := template.Must(template.ParseFiles("html/unauth_header.html", "html/login.html", "html/footer.html"))
				html.ExecuteTemplate(c.Writer, "html/unauth_header.html", gin.H{"Title": "Login"})
				html.ExecuteTemplate(c.Writer, "html/login.html", gin.H{"Errors": []string{"Incorrect username/password."}})
				html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
			}

		}
	}
}

func register(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)
	if uid == -1 {
		if c.Request.Method == "GET" {
			html := template.Must(template.ParseFiles("html/unauth_header.html", "html/register.html", "html/footer.html"))
			html.ExecuteTemplate(c.Writer, "html/unauth_header.html", gin.H{"Title": "Register"})
			html.ExecuteTemplate(c.Writer, "html/register.html", nil)
			html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
		} else if c.Request.Method == "POST" {
			username := c.PostForm("username")
			password := c.PostForm("password")
			confirm_password := c.PostForm("confirm_password")
			var inputErrors []string

			verified_pass, err := auth.ValidatePassword(password)
			if err != nil {
				inputErrors = append(inputErrors, err.Error())
			}
			verified_user, err := auth.ValidateUser(username)
			if err != nil {
				inputErrors = append(inputErrors, err.Error())
			}

			if password != confirm_password {
				inputErrors = append(inputErrors, "passwords do not match")
			}

			if len(inputErrors) == 0 {
				if querydb.UserExists(verified_user) == -1 {
					err = querydb.CreateUser(verified_user, auth.Hashpassword(verified_pass))
					if err != nil {
						logger.Error().Err(err).Msg("")
						return
					}
					html := template.Must(template.ParseFiles("html/unauth_header.html", "html/login.html", "html/footer.html"))
					html.ExecuteTemplate(c.Writer, "html/unauth_header.html", gin.H{"Title": "Login"})
					html.ExecuteTemplate(c.Writer, "html/login.html", nil)
					html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
					return
				} else {
					inputErrors = append(inputErrors, "user already exists")
				}
			}
			html := template.Must(template.ParseFiles("html/unauth_header.html", "html/register.html", "html/footer.html"))
			html.ExecuteTemplate(c.Writer, "html/unauth_header.html", gin.H{"Title": "Register"})
			html.ExecuteTemplate(c.Writer, "html/register.html", gin.H{"Errors": inputErrors})
			html.ExecuteTemplate(c.Writer, "html/footer.html", nil)

		}
	}
}

func logout(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)
	if uid != -1 {
		deauthsession(uid, c)
		c.Redirect(302, "/")
	}
}

func profile(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)
	if uid != -1 {
		user, err := auth.ValidateUser(c.Param("user"))
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		userinfo, err := querydb.Userinfo(uid)
		if err != nil {
			logger.Error().Err(err).Msg("")
		}

		if other_uid := querydb.UserExists(user); other_uid != -1 {
			other_userinfo, err := querydb.Userinfo(other_uid)
			if err != nil {
				logger.Error().Err(err).Msg("")
			}

			other_userinfo.Date_formatted = other_userinfo.Date_Joined.Format("2006-02-02")

			userlisted, err := querydb.GetUser(other_uid)
			if err != nil {
				logger.Error().Err(err).Msg("")
			}

			posts, err := querydb.RecentUserPosts(other_uid)
			if err != nil {
				logger.Error().Err(err).Msg("")
			}
			for i := 0; i < len(posts); i++ {
				posts[i].User = userlisted
				posts[i].Time_formatted = posts[i].Time_posted.Format("2006-02-02")
			}

			html := template.Must(template.ParseFiles("html/auth_header.html", "html/profile.html", "html/footer.html"))
			html.ExecuteTemplate(c.Writer, "html/auth_header.html", gin.H{"Title": other_userinfo.Username, "Userinfo": userinfo})
			html.ExecuteTemplate(c.Writer, "html/profile.html", gin.H{"Userinfo": other_userinfo, "RecentPosts": posts})
			html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
		}

	} else {
		login(c)
	}
}

func settings(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)
	if uid != -1 {
		if c.Request.Method == "GET" {
			userinfo, err := querydb.Userinfo(uid)
			if err != nil {
				logger.Error().Err(err).Msg("")
			}

			html := template.Must(template.ParseFiles("html/auth_header.html", "html/settings.html", "html/footer.html"))
			html.ExecuteTemplate(c.Writer, "html/auth_header.html", gin.H{"Title": "Settings", "Userinfo": userinfo})
			html.ExecuteTemplate(c.Writer, "html/settings.html", gin.H{"Userinfo": userinfo})
			html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
		} else if c.Request.Method == "POST" {
			if c.Param("setting") == "pfp" {
				pfp, err := c.FormFile("pfp")
				if err != nil {
					logger.Error().Err(err).Msg("")
					return
				}
				if pfp.Size > 500000 {
					html := template.Must(template.ParseFiles("html/htmx/form_feedback.html"))
					html.ExecuteTemplate(c.Writer, "html/htmx/form_feedback.html", gin.H{"Result": "error", "Message": "File is to big"})
					return
				}
				contentType := pfp.Header.Values("Content-Type")[0]
				rndname := func(fileype string) string {
					return randomString() + "." + fileype
				}
				if contentType == "image/png" {
					filename := rndname("png")
					c.SaveUploadedFile(pfp, "html/user_pictures/"+filename)
					querydb.SetPFP(uid, filename)
				} else if contentType == "image/jpg" {
					filename := rndname("jpg")
					c.SaveUploadedFile(pfp, "html/user_pictures/"+filename)
					querydb.SetPFP(uid, filename)
				} else if contentType == "image/jpeg" {
					filename := rndname("jpeg")
					c.SaveUploadedFile(pfp, "html/user_pictures/"+filename)
					querydb.SetPFP(uid, filename)
				} else if contentType == "image/gif" {
					filename := rndname("gif")
					c.SaveUploadedFile(pfp, "html/user_pictures/"+filename)
					querydb.SetPFP(uid, filename)
				} else {
					html := template.Must(template.ParseFiles("html/htmx/form_feedback.html"))
					html.ExecuteTemplate(c.Writer, "html/htmx/form_feedback.html", gin.H{"Result": "error", "Message": "invalid file type"})
					return
				}

				html := template.Must(template.ParseFiles("html/htmx/form_feedback.html"))
				html.ExecuteTemplate(c.Writer, "html/htmx/form_feedback.html", gin.H{"Result": "ok", "Message": "profile updated"})
				return

			} else if c.Param("setting") == "color" {
				fg := c.PostForm("fg")
				bg := c.PostForm("bg")
				fg = strings.Replace(fg, "#", "", 1)
				bg = strings.Replace(bg, "#", "", 1)
				if _, err := hex.DecodeString(fg); err != nil || len(fg) != 6 {
					logger.Error().Err(err).Msg("")
					html := template.Must(template.ParseFiles("html/htmx/form_feedback.html"))
					html.ExecuteTemplate(c.Writer, "html/htmx/form_feedback.html", gin.H{"Result": "error", "Message": "invalid color format"})
					return
				}

				if _, err := hex.DecodeString(bg); err != nil || len(bg) != 6 {
					logger.Error().Err(err).Msg("")
					html := template.Must(template.ParseFiles("html/htmx/form_feedback.html"))
					html.ExecuteTemplate(c.Writer, "html/htmx/form_feedback.html", gin.H{"Result": "error", "Message": "invalid color format"})
					return
				}

				if err := querydb.SetColor(uid, "fg", fg); err != nil {
					logger.Error().Err(err).Msg("")
					html := template.Must(template.ParseFiles("html/htmx/form_feedback.html"))
					html.ExecuteTemplate(c.Writer, "html/htmx/form_feedback.html", gin.H{"Result": "error", "Message": "error setting colors"})
					return
				}

				if err := querydb.SetColor(uid, "bg", bg); err != nil {
					logger.Error().Err(err).Msg("")
					html := template.Must(template.ParseFiles("html/htmx/form_feedback.html"))
					html.ExecuteTemplate(c.Writer, "html/htmx/form_feedback.html", gin.H{"Result": "error", "Message": "error setting colors"})
					return
				}

				html := template.Must(template.ParseFiles("html/htmx/form_feedback.html"))
				html.ExecuteTemplate(c.Writer, "html/htmx/form_feedback.html", gin.H{"Result": "ok", "Message": "set username colors"})
				return

			} else if c.Param("setting") == "bio" {
				bio := c.PostForm("profile-bio")
				err := querydb.SetBio(uid, bio)
				if err != nil {
					logger.Error().Err(err).Msg("")
					html := template.Must(template.ParseFiles("html/htmx/form_feedback.html"))
					html.ExecuteTemplate(c.Writer, "html/htmx/form_feedback.html", gin.H{"Result": "error", "Message": "error updating bio"})
					return
				}
				html := template.Must(template.ParseFiles("html/htmx/form_feedback.html"))
				html.ExecuteTemplate(c.Writer, "html/htmx/form_feedback.html", gin.H{"Result": "ok", "Message": "set bio"})
				return
			}
		}
	}
}

func editor(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)
	if uid != -1 {

		userinfo, err := querydb.Userinfo(uid)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		if c.Param("id") == "" {
			html := template.Must(template.ParseFiles("html/auth_header.html", "html/editor.html", "html/footer.html"))
			html.ExecuteTemplate(c.Writer, "html/auth_header.html", gin.H{"Title": "editor", "Userinfo": userinfo})
			html.ExecuteTemplate(c.Writer, "html/editor.html", gin.H{"Categories": Categories})
			html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
			return
		} else {

			pid, err := strconv.ParseInt(c.Param("id"), 10, 32)
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}

			postinfo, err := querydb.GetPost(int32(pid))
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}

			if postinfo.Uid != uid {
				logger.Error().Err(errors.New("user tried to access unauthorized resource"))
				return
			}

			postHTML := template.HTML(string(postinfo.Html))

			html := template.Must(template.ParseFiles("html/auth_header.html", "html/editor.html", "html/footer.html"))
			html.ExecuteTemplate(c.Writer, "html/auth_header.html", gin.H{"Title": "editor", "Userinfo": userinfo})
			html.ExecuteTemplate(c.Writer, "html/editor.html", gin.H{"Postinfo": postinfo, "PostHTML": postHTML, "Categories": Categories})
			html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
			return
		}
	}
}

func render(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)
	if uid != -1 {
		var raw_md models.Post
		var buf bytes.Buffer
		if err := c.ShouldBindJSON(&raw_md); err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		if err := md.Convert([]byte(raw_md.Md), &buf); err != nil {
			logger.Error().Err(err).Msg("")
			return
		}
		c.String(200, buf.String())
	}

}

func save(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)
	if uid != -1 {

		var post models.Post
		//html buf
		var buf bytes.Buffer

		if err := c.ShouldBindJSON(&post); err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		section, err := validateSection(post.Section)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		//compile html
		if err := md.Convert([]byte(post.Md), &buf); err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		//if no id in path create a new draft
		if c.Param("id") == "" {
			pid, err := querydb.NewPost(uid, section.Id, "draft", post.Title, post.Md, buf.String())
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}
			c.JSON(200, gin.H{"pid": pid, "html": buf.String()})
			return
			//if id update post
		} else {

			pid, err := strconv.ParseInt(c.Param("id"), 10, 32)
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}
			poster, _, _, err := querydb.GetPostOP(int32(pid))
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}

			if poster != uid {
				logger.Error().Err(errors.New("user tried to access unauthorized resource"))
				return
			}

			err = querydb.UpdatePost(int32(pid), post.Title, post.Md, buf.String(), section.Id)
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}
			c.JSON(200, gin.H{"html": buf.String()})
		}
	}
}

func post(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)
	if uid != -1 {
		var post models.Post
		var buf bytes.Buffer
		var err error

		if err := c.ShouldBindJSON(&post); err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		section, err := validateSection(post.Section)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		if err := md.Convert([]byte(post.Md), &buf); err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		if c.Param("id") == "" {
			pid, err := querydb.NewPost(uid, section.Id, "posted", post.Title, post.Md, buf.String())
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}
			c.JSON(200, gin.H{"pid": pid, "section": section.Id, "title": post.Title})
		} else {
			pid, err := strconv.ParseInt(c.Param("id"), 10, 32)
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}
			poster, _, _, err := querydb.GetPostOP(int32(pid))
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}

			if poster != uid {
				logger.Error().Err(errors.New("user tried to access unauthorized resource"))
				return
			}

			err = querydb.UpdatePost(int32(pid), post.Title, post.Md, buf.String(), section.Id)
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}
			err = querydb.UpdatePostStatus(int32(pid), "posted")
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}
			c.JSON(200, gin.H{"pid": pid, "section": section.Id, "title": post.Title})
		}
	}
}

func posts(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)
	if uid != -1 {
		userinfo, err := querydb.Userinfo(uid)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}
		user := c.Param("user")
		user_id := querydb.UserExists(models.Username(user))

		userListed, err := querydb.GetUser(user_id)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		posts, err := querydb.UserPosts(user_id, "posted")
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		for i := 0; i < len(posts); i++ {
			posts[i].User = userListed
			posts[i].Status = "posted"
			posts[i].Time_formatted = posts[i].Time_posted.Format("2006-02-02")
		}

		html := template.Must(template.ParseFiles("html/auth_header.html", "html/user-posts.html", "html/footer.html"))
		html.ExecuteTemplate(c.Writer, "html/auth_header.html", gin.H{"Title": "posts", "Userinfo": userinfo})
		html.ExecuteTemplate(c.Writer, "html/user-posts.html", gin.H{"Posts": posts, "Status": "Posts"})
		html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
	}
}

func drafts(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)
	if uid != -1 {
		userinfo, err := querydb.Userinfo(uid)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		userListed, err := querydb.GetUser(uid)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		posts, err := querydb.UserPosts(uid, "draft")
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		for i := 0; i < len(posts); i++ {
			posts[i].User = userListed
			posts[i].Status = "draft"
			posts[i].Time_formatted = posts[i].Time_posted.Format("2006-02-02")
		}

		html := template.Must(template.ParseFiles("html/auth_header.html", "html/user-posts.html", "html/footer.html"))
		html.ExecuteTemplate(c.Writer, "html/auth_header.html", gin.H{"Title": "drafts", "Userinfo": userinfo})
		html.ExecuteTemplate(c.Writer, "html/user-posts.html", gin.H{"Posts": posts, "Status": "Drafts"})
		html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
	}
}

func section(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)

	section, err := validateSection(c.Param("section"))
	if err != nil {
		logger.Error().Err(err).Msg("")
		return
	}
	if uid != -1 {
		userinfo, err := querydb.Userinfo(uid)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		html := template.Must(template.ParseFiles("html/auth_header.html", "html/section.html", "html/footer.html"))
		html.ExecuteTemplate(c.Writer, "html/auth_header.html", gin.H{"Title": section.Section, "Userinfo": userinfo})
		html.ExecuteTemplate(c.Writer, "html/section.html", gin.H{"Section": section.Id, "Logged_in": true})
		html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
	} else {
		html := template.Must(template.ParseFiles("html/unauth_header.html", "html/section.html", "html/footer.html"))
		html.ExecuteTemplate(c.Writer, "html/unauth_header.html", gin.H{"Title": section.Section})
		html.ExecuteTemplate(c.Writer, "html/section.html", gin.H{"Section": section.Id, "Logged_in": false})
		html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
	}
}

func viewPost(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)

	pid, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		logger.Error().Err(err).Msg("")
		return
	}

	postinfo, err := querydb.GetPost(int32(pid))
	if err != nil {
		logger.Error().Err(err).Msg("")
		return
	}

	postinfo.Time_formatted = postinfo.Time_posted.Format("2006-02-02")

	userlisted, err := querydb.GetUser(postinfo.Uid)
	if err != nil {
		logger.Error().Err(err).Msg("")
		return
	}

	comments, err := querydb.GetComments(postinfo.Pid)
	if err != nil {
		logger.Error().Err(err).Msg("")
		return
	}
	for i := 0; i < len(comments); i++ {
		comments[i].User, err = querydb.GetUser(comments[i].User_id)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}
	}

	if uid != -1 {

		userinfo, err := querydb.Userinfo(uid)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		liked, _ := querydb.Liked(uid, postinfo.Pid)
		html := template.Must(template.ParseFiles("html/auth_header.html", "html/post.html", "html/footer.html"))
		html.ExecuteTemplate(c.Writer, "html/auth_header.html", gin.H{"Title": postinfo.Title, "Userinfo": userinfo})
		html.ExecuteTemplate(c.Writer, "html/post.html", gin.H{"Postinfo": postinfo,
			"User":      userinfo,
			"Comments":  comments,
			"Liked":     liked,
			"Logged_in": true,
			"Editable":  postinfo.Uid == uid})
		html.ExecuteTemplate(c.Writer, "html/footer.html", nil)

	} else {
		html := template.Must(template.ParseFiles("html/unauth_header.html", "html/post.html", "html/footer.html"))
		html.ExecuteTemplate(c.Writer, "html/unauth_header.html", gin.H{"Title": postinfo.Title})
		html.ExecuteTemplate(c.Writer, "html/post.html", gin.H{"Postinfo": postinfo,
			"User":      userlisted,
			"Comments":  comments,
			"Liked":     false,
			"Logged_in": false,
			"Editable":  postinfo.Uid == uid})
		html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
	}
}

func reply(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)
	if uid != -1 {
		var cid int64
		pid, err := strconv.ParseInt(c.Param("pid"), 10, 32)

		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		if c.Param("cid") != "" {
			cid, err = strconv.ParseInt(c.Param("cid"), 10, 32)

			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}
		}

		if c.Request.Method == "GET" {
			html := template.Must(template.ParseFiles("html/htmx/reply.html"))
			html.ExecuteTemplate(c.Writer, "html/htmx/reply.html", gin.H{"Pid": pid, "Cid": cid})
			return

		} else if c.Request.Method == "POST" {
			comment := c.PostForm("comment")
			var buf bytes.Buffer

			if len(comment) < 10 {
				fmt.Println(errors.New("comment to short"))
				return
			}

			OP, section, title, err := querydb.GetPostOP(int32(pid))
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}
			if cid == 0 {

				if err := md.Convert([]byte(comment), &buf); err != nil {
					logger.Error().Err(err).Msg("")
					return
				}

				_, err = querydb.PostComment(uid, int32(pid), -1, comment, buf.String())
				if err != nil {
					logger.Error().Err(err).Msg("")
					return
				}
				if OP != uid {
					err = querydb.NewNotification(OP, uid, fmt.Sprintf(`Left a comment on your post <a href="/section/%s/%d/%s">%s</a>`, section, pid, title, title))
					if err != nil {
						logger.Error().Err(err).Msg("")
						return
					}
				}

			} else if cid != 0 {
				if err := md.Convert([]byte(comment), &buf); err != nil {
					logger.Error().Err(err).Msg("")
					return
				}

				_, err = querydb.PostComment(uid, int32(pid), int32(cid), comment, buf.String())
				if err != nil {
					logger.Error().Err(err).Msg("")
					return
				}

				comment_poster, err := querydb.GetCommentPoster(int32(cid))
				if err != nil {
					logger.Error().Err(err).Msg("")
					return
				}

				if comment_poster != uid {
					err = querydb.NewNotification(comment_poster, uid, fmt.Sprintf(`Responsed to your comment on <a href="/section/%s/%d/%s">%s</a>`, section, pid, title, title))
					if err != nil {
						logger.Error().Err(err).Msg("")
						return
					}
				}
			}
			c.Header("HX-Refresh", "true")
		}
	}

}

func like(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)
	if uid != -1 {
		pid, err := strconv.ParseInt(c.Param("pid"), 10, 32)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}
		err = querydb.LikeUnlike(uid, int32(pid))
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}
	}
}

func likes(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)
	if uid != -1 {
		userinfo, err := querydb.Userinfo(uid)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		posts, err := querydb.Likes(uid)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		for i := 0; i < len(posts); i++ {
			posts[i].User, err = querydb.GetUser(posts[i].Uid)
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}
			posts[i].Time_formatted = posts[i].Time_posted.Format("2006-02-02")
		}

		html := template.Must(template.ParseFiles("html/auth_header.html", "html/user-posts.html", "html/footer.html"))
		html.ExecuteTemplate(c.Writer, "html/auth_header.html", gin.H{"Title": "likes", "Userinfo": userinfo})
		html.ExecuteTemplate(c.Writer, "html/user-posts.html", gin.H{"Status": "likes", "Posts": posts})
		html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
	}
}

func notifications(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)
	if uid != -1 {
		userinfo, err := querydb.Userinfo(uid)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		notifications, err := querydb.Notifications(uid)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		for i := 0; i < len(notifications); i++ {
			notifications[i].From_Uid_Listing, _ = querydb.GetUser(notifications[i].From_Uid)
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}
		}
		html := template.Must(template.ParseFiles("html/auth_header.html", "html/notifications.html", "html/footer.html"))
		html.ExecuteTemplate(c.Writer, "html/auth_header.html", gin.H{"Title": "notifications", "Userinfo": userinfo})
		html.ExecuteTemplate(c.Writer, "html/notifications.html", gin.H{"Notifications": notifications})
		html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
	}
}

func search(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)

	qry := c.Query("search")

	if qry == "" {
		if uid != -1 {
			userinfo, err := querydb.Userinfo(uid)
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}
			html := template.Must(template.ParseFiles("html/auth_header.html", "html/search.html", "html/footer.html"))
			html.ExecuteTemplate(c.Writer, "html/auth_header.html", gin.H{"Title": "search", "Userinfo": userinfo})
			html.ExecuteTemplate(c.Writer, "html/search.html", nil)
			html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
			return
		} else {
			html := template.Must(template.ParseFiles("html/unauth_header.html", "html/search.html", "html/footer.html"))
			html.ExecuteTemplate(c.Writer, "html/unauth_header.html", gin.H{"Title": "search"})
			html.ExecuteTemplate(c.Writer, "html/search.html", nil)
			html.ExecuteTemplate(c.Writer, "html/footer.html", nil)
			return
		}
	} else if qry != "" {
		posts, err := querydb.Search(qry)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		for i := 0; i < len(posts); i++ {
			posts[i].User, err = querydb.GetUser(posts[i].Uid)
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}

			posts[i].Time_formatted = posts[i].Time_posted.Format("2006-02-02")
		}

		html := template.Must(template.ParseFiles("html/htmx/results.html"))
		html.ExecuteTemplate(c.Writer, "html/htmx/results.html", gin.H{"Posts": posts})
	}
}

func deletePost(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)
	if uid != -1 {
		pid, err := strconv.ParseInt(c.Param("pid"), 10, 32)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		userlisted, err := querydb.GetUser(uid)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		postop, _, _, err := querydb.GetPostOP(int32(pid))
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		if postop == uid {
			err = querydb.DeletePost(int32(pid))
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}
			c.Header("HX-Redirect", fmt.Sprintf("/user/%s/posts", userlisted.Username))
		}
	}
}

func deleteReply(c *gin.Context) {
	initsession(c)
	session, _ := store.Get(c.Request, "session")
	uid := session.Values["id"].(int32)
	if uid != -1 {
		cid, err := strconv.ParseInt(c.Param("cid"), 10, 32)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}

		commentPost, err := querydb.GetCommentPoster(int32(cid))
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}
		if commentPost == uid {
			err = querydb.DeleteReply(int32(cid))
			if err != nil {
				logger.Error().Err(err).Msg("")
				return
			}
		}
	}
}

func mostLiked(c *gin.Context) {
	section, err := validateSection(c.Param("section"))
	if err != nil {
		logger.Error().Err(err).Msg("")
		return
	}

	posts, err := querydb.MostLiked(section)
	if err != nil {
		logger.Error().Err(err).Msg("")
		return
	}

	for i := 0; i < len(posts); i++ {
		posts[i].User, err = querydb.GetUser(posts[i].Uid)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}
		posts[i].Time_formatted = posts[i].Time_posted.Format("2006-02-02")
	}

	html := template.Must(template.ParseFiles("html/htmx/results.html"))
	html.ExecuteTemplate(c.Writer, "html/htmx/results.html", gin.H{"Posts": posts, "Section": section.Id})
}

func newest(c *gin.Context) {
	section, err := validateSection(c.Param("section"))
	if err != nil {
		logger.Error().Err(err).Msg("")
		return
	}

	posts, err := querydb.GetSectionPosts(section.Id)
	if err != nil {
		logger.Error().Err(err).Msg("")
		return
	}

	for i := 0; i < len(posts); i++ {
		posts[i].User, err = querydb.GetUser(posts[i].Uid)
		if err != nil {
			logger.Error().Err(err).Msg("")
			return
		}
		posts[i].Time_formatted = posts[i].Time_posted.Format("2006-02-02")
	}

	html := template.Must(template.ParseFiles("html/htmx/results.html"))
	html.ExecuteTemplate(c.Writer, "html/htmx/results.html", gin.H{"Posts": posts, "Section": section.Id})
}
