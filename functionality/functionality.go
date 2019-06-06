package functionality

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

const (
	ArticlesURL = "https://raw.githubusercontent.com/the-rileyj/Jetpacks/master/README.md"
)

type JetpackArticle struct {
	Title        string `json:"title"`
	BodyMarkdown string `json:"bodyMarkdown"`
}

type JetpackArticles struct {
	Articles        []JetpackArticle `json:"articles"`
	MainTitle       string           `json:"mainTitle"`
	MainDescription string           `json:"mainDescription"`
}

func chompWhiteSpace(markdownScanner *bufio.Scanner) error {
	for {
		markdownLine := markdownScanner.Text()

		if strings.TrimSpace(markdownLine) != "" {
			return nil
		}

		if !markdownScanner.Scan() {
			return errors.New("scanning ended early")
		}
	}
}

func parseHeading(markdownScanner *bufio.Scanner, headingPrefix string) (string, error) {
	var err error

	for {
		markdownLine := markdownScanner.Text()

		if strings.HasPrefix(markdownLine, headingPrefix) {
			// Chomp the heading before returning
			if !markdownScanner.Scan() {
				return "", errors.New("scanning ended early")
			}

			return markdownLine[len(headingPrefix):], nil
		} else if !markdownScanner.Scan() {
			return "", errors.New("scanning ended early")
		}

		// Chomp until we are no longer being fed whitespace
		if err = chompWhiteSpace(markdownScanner); err != nil {
			return "", err
		}
	}
}

func parseMainDescription(markdownScanner *bufio.Scanner) (string, error) {
	var textBuilder strings.Builder

	for {
		markdownLine := markdownScanner.Text()

		if strings.HasPrefix(markdownLine, "## Jetpacks") {
			// Chomp the "Jetpacks" heading before returning
			if !markdownScanner.Scan() {
				return "", errors.New("scanning ended early")
			}

			return textBuilder.String(), nil
		}

		textBuilder.WriteString(markdownLine + "\n")

		if !markdownScanner.Scan() {
			return "", errors.New("scanning ended early")
		}
	}
}

func parseArticleBody(markdownScanner *bufio.Scanner) (string, error) {
	var textBuilder strings.Builder

	bodyTopLevel := true

	for {
		markdownLine := markdownScanner.Text()

		if bodyTopLevel && strings.HasPrefix(markdownLine, "## ") {
			return textBuilder.String(), nil
		} else if strings.HasPrefix(markdownLine, "```") {
			// Handle code areas where there is a possibility for false positive "##"'s
			bodyTopLevel = !bodyTopLevel
		}

		textBuilder.WriteString(markdownLine + "\n")

		// If we reach the end of scanning while searching for the end of the article body,
		// that indicates that we have reached the end of the input
		if !markdownScanner.Scan() {
			return textBuilder.String(), io.EOF
		}
	}
}

func parseToArticles(r io.Reader) (JetpackArticles, error) {
	var (
		err             error
		jetpackArticles JetpackArticles
		jetpackArticle  JetpackArticle
	)

	markdownScanner := bufio.NewScanner(r)

	if jetpackArticles.MainTitle, err = parseHeading(markdownScanner, "# "); err != nil {
		return JetpackArticles{}, err
	}

	if err = chompWhiteSpace(markdownScanner); err != nil {
		return JetpackArticles{}, err
	}

	if jetpackArticles.MainDescription, err = parseMainDescription(markdownScanner); err != nil {
		return JetpackArticles{}, err
	}

	if err = chompWhiteSpace(markdownScanner); err != nil {
		return JetpackArticles{}, err
	}

	for {
		if jetpackArticle.Title, err = parseHeading(markdownScanner, "## "); err != nil {
			return JetpackArticles{}, err
		}

		if err = chompWhiteSpace(markdownScanner); err != nil {
			return JetpackArticles{}, err
		}

		if jetpackArticle.BodyMarkdown, err = parseArticleBody(markdownScanner); err != nil {
			if err == io.EOF {
				jetpackArticles.Articles = append(jetpackArticles.Articles, jetpackArticle)

				return jetpackArticles, nil
			} else {
				return JetpackArticles{}, err
			}
		}

		jetpackArticles.Articles = append(jetpackArticles.Articles, jetpackArticle)
	}
}

func FetchJetpackArticles() (JetpackArticles, error) {
	response, err := http.Get(ArticlesURL)

	if err != nil {
		return JetpackArticles{}, err
	}

	defer response.Body.Close()

	return parseToArticles(response.Body)
}

func GetGithubSecret() (string, error) {
	infoStruct := struct {
		Secret string `json:"secret"`
	}{}

	infoFile, err := os.Open("info.json")

	if err != nil {
		return "", err
	}

	err = json.NewDecoder(infoFile).Decode(&infoStruct)

	if err != nil {
		return "", err
	}

	return infoStruct.Secret, nil
}

func GetHandlerForGetJetpackArticles(getJetpackArticles func() *JetpackArticles) func(c *gin.Context) {
	return func(c *gin.Context) { c.JSON(200, getJetpackArticles()) }
}

func GetHandlerForUpdateJetpackArticlesHandler(secret string, updateJetpackArticles func() error) func(c *gin.Context) {
	return func(c *gin.Context) {
		bodyBytes, err := ioutil.ReadAll(c.Request.Body)

		if err != nil {
			c.Status(500)
			c.Writer.Write([]byte(fmt.Sprintf("Articles Update Failed: %s", err.Error())))

			return
		}

		signature := c.GetHeader("X-Hub-Signature")

		if len(signature) < 6 {
			c.Status(500)
			c.Writer.Write([]byte(fmt.Sprintf("Articles Update Failed: %s", err.Error())))

			return
		}

		actual := make([]byte, 20)

		_, err = hex.Decode(actual, []byte(signature[5:]))

		if err != nil {
			c.Status(500)
			c.Writer.Write([]byte(fmt.Sprintf("Articles Update Failed: %s", err.Error())))

			return
		}

		requestHMAC := hmac.New(sha1.New, []byte(secret))

		_, err = requestHMAC.Write(bodyBytes)

		if err != nil {
			c.Status(500)
			c.Writer.Write([]byte(fmt.Sprintf("Articles Update Failed: %s", err.Error())))

			return
		}

		if !hmac.Equal(requestHMAC.Sum(nil), actual) {
			c.Status(500)
			c.Writer.Write([]byte("Articles Update Failed: Signature sent and signature generated do not match"))

			return
		}

		updateJetpackArticles()

		c.Status(202)
		c.Writer.Write([]byte("Articles Updated Successfully"))
	}
}

func GetJetpackRouter() *gin.Engine {
	var accessJetpackArticlesMutex sync.Mutex

	tmpJetpackArticles, err := FetchJetpackArticles()

	if err != nil {
		panic(err)
	}

	jetpackArticlesPointer := &tmpJetpackArticles

	router := gin.Default()

	secret, err := GetGithubSecret()

	if err != nil {
		panic(err)
	}

	router.GET("/api/jetpack/articles", GetHandlerForGetJetpackArticles(func() *JetpackArticles {
		accessJetpackArticlesMutex.Lock()

		defer accessJetpackArticlesMutex.Unlock()

		return jetpackArticlesPointer
	}))

	router.POST("/api/jetpack/articles", GetHandlerForUpdateJetpackArticlesHandler(secret, func() error {
		tmpJetpackArticles, err := FetchJetpackArticles()

		if err != nil {
			return err
		}

		accessJetpackArticlesMutex.Lock()

		jetpackArticlesPointer = &tmpJetpackArticles

		defer accessJetpackArticlesMutex.Unlock()

		return nil
	}))

	return router
}
