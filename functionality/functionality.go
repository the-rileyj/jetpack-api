package functionality

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
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

func parseToArticles(r io.Reader) (JetpackArticles, error) {
	var (
		jetpackArticles JetpackArticles
		jetpackArticle  JetpackArticle
		markdownBytes   []byte
		textBuilder     strings.Builder
	)

	markdownScanner := bufio.NewScanner(r)

	for markdownScanner.Scan() {
		markdownBytes = markdownScanner.Bytes()

		switch lineBeginning := string(markdownBytes); {
		case strings.HasPrefix(lineBeginning, "# "):
			jetpackArticles.MainTitle = string(markdownBytes[2:])

			if !markdownScanner.Scan() {
				break
			}

			textBuilder.Reset()

		case strings.HasPrefix(lineBeginning, "## Jetpacks"):
			jetpackArticles.MainDescription = textBuilder.String()

			if !markdownScanner.Scan() {
				break
			}

			textBuilder.Reset()

		case jetpackArticles.MainDescription != "" && strings.HasPrefix(lineBeginning, "### "):
			if jetpackArticle.Title != "" {
				jetpackArticle.BodyMarkdown = textBuilder.String()

				jetpackArticles.Articles = append(jetpackArticles.Articles, jetpackArticle)
			}

			jetpackArticle.Title = string(markdownBytes[4:])

			if !markdownScanner.Scan() {
				break
			}

			textBuilder.Reset()

		default:
			_, err := textBuilder.Write(append(markdownBytes, '\n'))

			if err != nil {
				return JetpackArticles{}, err
			}
		}
	}

	jetpackArticle.BodyMarkdown = textBuilder.String()

	jetpackArticles.Articles = append(jetpackArticles.Articles, jetpackArticle)

	return jetpackArticles, nil
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
	return "", nil
}

func GetHandlerForGetJetpackArticles(getJetpackArticles func() *JetpackArticles) func(c *gin.Context) {
	return func(c *gin.Context) { c.JSON(200, getJetpackArticles()) }
}

func GetHandlerForUpdateJetpackArticlesHandler(secret string, updateJetpackArticles func() error) func(c *gin.Context) {
	return func(c *gin.Context) {
		requestHMAC := hmac.New(sha1.New, []byte(secret))

		bodyBytes, err := ioutil.ReadAll(c.Request.Body)

		if err != nil {
			c.Status(500)
			c.Writer.Write([]byte(fmt.Sprintf("Articles Update Failed: %s", err.Error())))

			return
		}

		requestHMAC.Write(bodyBytes)

		if c.GetHeader("X-Hub-Signature") != base64.StdEncoding.EncodeToString(requestHMAC.Sum(nil)) {
			c.Status(500)
			c.Writer.Write([]byte(fmt.Sprintf("Articles Update Failed: Signature sent and signature generated do not match")))

			return
		}

		err = updateJetpackArticles()

		if err != nil {
			c.Status(500)
			c.Writer.Write([]byte(fmt.Sprintf("Articles Update Failed: %s", err.Error())))

			return
		}

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
	secret := ""

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
