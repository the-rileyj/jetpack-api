package functionality

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestParseToArticles(t *testing.T) {
	var (
		testInputs = []string{`

# Test Jetpack Articles

Yep, you guessed it; this is for testing.

## Jetpacks

## Test article 1

The first test article

## Test article 2

The second test article
`,
			`

# Test Jetpack Articles

Yep, you guessed it; this is for testing.

## Jetpacks

## Test article 3

### The first test article

Would you believe it?

`,
			`

# Test Jetpack Articles



Yep, you guessed it; this is for testing.

## Jetpacks




## Test article 4



### The first test article

Would you believe it?

`,
			`

# Test Jetpack Articles

Yep, you guessed it; this is for testing.

## Jetpacks

## Test article 5

### The first test article

Would you believe it?

` + "```" + `

This is a test to handle code blocks,

## This mock heading should not be mistaken as an actual heading

` + "```" + `

`,
		}
		expectedOutputs = []JetpackArticles{
			JetpackArticles{
				Articles: []JetpackArticle{
					{"Test article 1", "The first test article\n\n"},
					{"Test article 2", "The second test article\n"},
				},
				MainDescription: "Yep, you guessed it; this is for testing.\n\n",
				MainTitle:       "Test Jetpack Articles",
			},
			JetpackArticles{
				Articles: []JetpackArticle{
					{"Test article 3", "### The first test article\n\nWould you believe it?\n\n"},
				},
				MainDescription: "Yep, you guessed it; this is for testing.\n\n",
				MainTitle:       "Test Jetpack Articles",
			},
			JetpackArticles{
				Articles: []JetpackArticle{
					{"Test article 4", "### The first test article\n\nWould you believe it?\n\n"},
				},
				MainDescription: "Yep, you guessed it; this is for testing.\n\n",
				MainTitle:       "Test Jetpack Articles",
			},
			JetpackArticles{
				Articles: []JetpackArticle{
					{"Test article 5", "### The first test article\n\nWould you believe it?\n\n```\n\nThis is a test to handle code blocks,\n\n## This mock heading should not be mistaken as an actual heading\n\n```\n\n"},
				},
				MainDescription: "Yep, you guessed it; this is for testing.\n\n",
				MainTitle:       "Test Jetpack Articles",
			},
		}

		allArticlesEqual bool
		err              error
		jetpackArticles  JetpackArticles
	)

	for i, testInput := range testInputs {
		jetpackArticles, err = parseToArticles(strings.NewReader(testInput))

		if err != nil {
			t.Errorf("Unexpected error occurred \"%v\"", err)
		}

		allArticlesEqual = true

		for j, testArticle := range jetpackArticles.Articles {
			if testArticle.BodyMarkdown != expectedOutputs[i].Articles[j].BodyMarkdown || testArticle.Title != expectedOutputs[i].Articles[j].Title {
				allArticlesEqual = false
				break
			}
		}

		if !allArticlesEqual || jetpackArticles.MainDescription != expectedOutputs[i].MainDescription || jetpackArticles.MainTitle != expectedOutputs[i].MainTitle {
			t.Errorf("Expected the articles to be parsed as:\n\n\"%v\",\n\nbut got:\n\n\"%v\"\n\n instead\n", expectedOutputs[i], jetpackArticles)
		}
	}
}
