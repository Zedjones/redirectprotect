package routes

import (
	"net/http"
	"strings"
	"time"

	"github.com/zedjones/redirectprotect/internal"

	"github.com/google/uuid"
	"github.com/labstack/echo"
	"github.com/zedjones/redirectprotect/db"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2/bson"
)

//Doing this because I want to be able to mock these functions
var (
	generateFromPassword   = bcrypt.GenerateFromPassword
	compareHashAndPassword = bcrypt.CompareHashAndPassword
	getConnection          = db.GetConnection
	startTimeCheck         = internal.StartTimeCheck
	uuidNew                = uuid.New
)

func RegisterURL(c echo.Context) error {
	var duration time.Duration
	url := c.QueryParam("url")
	passphrase := c.QueryParam("passphrase")
	durationStr := c.QueryParam("ttl")
	var err error

	if url == "" || passphrase == "" {
		return c.String(http.StatusBadRequest, "URL or passphrase not provided")
	}
	if durationStr != "" {
		duration, err = time.ParseDuration(durationStr)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error parsing duration")
		}
	}
	if (strings.Contains(url, ":/") || strings.Contains(url, ":?")) &&
		!(strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://")) {
		return c.String(http.StatusBadRequest, "Invalid URL provided")
	} else if !strings.Contains(url, ":/") {
		url = "http://" + url
	}

	bytes, err := generateFromPassword([]byte(passphrase), 15)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	newRedirect := db.Redirect{URL: url, Password: string(bytes),
		TTL: duration.String(), Path: uuidNew().String()}

	connection, err := getConnection()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to acquire database connection")
	}
	collection := connection.Collection(db.CollectionName)

	err = collection.Save(&newRedirect)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to save redirect to the database")
	}
	go startTimeCheck(&newRedirect, collection)
	return c.String(http.StatusOK, newRedirect.Path)
}

func GetRedirect(c echo.Context) error {
	redir := &db.Redirect{}
	connection, err := getConnection()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to acquire database connection")
	}
	path := strings.TrimPrefix(c.Request().URL.Path, "/")
	err = connection.Collection(db.CollectionName).FindOne(bson.M{"path": path}, redir)
	if err != nil {
		return c.String(http.StatusBadRequest, "Shortened URL does not exist")
	}
	return c.Render(http.StatusOK, "redir.html", nil)
}

func CheckPassphrase(c echo.Context) error {
	redir := &db.Redirect{}
	path := c.QueryParam("path")
	passphrase := c.QueryParam("passphrase")
	connection, err := getConnection()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to acquire database connection")
	}
	err = connection.Collection(db.CollectionName).FindOne(bson.M{"path": path}, redir)
	if err != nil {
		return c.String(http.StatusBadRequest, "Shortened URL does not exist")
	}
	err = compareHashAndPassword([]byte(redir.Password), []byte(passphrase))
	if err == nil {
		c.JSON(http.StatusOK, map[string]string{"url": redir.URL})
	}
	return err
}
