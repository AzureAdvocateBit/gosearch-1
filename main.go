package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// for next time
// - make a UI for all this stuff!
// - caching the results for a few mins or something?
// - maybe proactively getting search results in the background
// - doing image, video, etc... search?
// - deploy this to the actual internet
//		- rate limiting will be needed
// - WRITE OUR OWN CRAWLER!!!

// const bingEndpoint = "https://gosearch.cognitiveservices.azure.com/bing/v7.0"
const bingEndpoint = "https://api.cognitive.microsoft.com/bing/v7.0/search"

// localhost:123/api/search?term=thing

func newSearchHandler(token string) echo.HandlerFunc {
	type result struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	type queryResults struct {
		ResultList []result `json:"results"`
	}

	appendResult := func(results *queryResults, name, url string) {
		results.ResultList = append(results.ResultList, result{
			Name: name,
			URL:  url,
		})
	}

	return func(ctx echo.Context) error {
		term := ctx.QueryParam("term")
		if term == "" {
			return newError(
				ctx,
				http.StatusBadRequest,
				"No search term in query string",
			)
		}
		// start making the request to Bing
		// Declare a new GET request.
		req, err := http.NewRequest("GET", bingEndpoint, nil)
		if err != nil {
			return newError(
				ctx,
				http.StatusInternalServerError,
				"%s",
				err,
			)
		}

		// Add the payload to the request.
		param := req.URL.Query()
		param.Add("q", term)
		req.URL.RawQuery = param.Encode()

		// Insert the request header.
		req.Header.Add("Ocp-Apim-Subscription-Key", token)

		// Create a new client.
		client := new(http.Client)

		// Send the request to Bing.
		resp, err := client.Do(req)
		if err != nil {
			return newError(
				ctx,
				http.StatusInternalServerError,
				"%s",
				err,
			)
		}

		// Close the response.
		defer resp.Body.Close()

		// Create a new answer.
		ans := &BingAnswer{} // same thing as new(BingAnswer)

		// TODO: this code is from the example site:
		// (https://docs.microsoft.com/en-us/azure/cognitive-services/bing-web-search/quickstarts/go#handle-the-response)
		// it's wrong because passing in &ans to json.Unmarshal
		// passes in a pointer to a pointer - we should just be
		// passing in a pointer.
		// err = json.Unmarshal(body, &ans)

		if err := json.NewDecoder(resp.Body).Decode(ans); err != nil {
			return newError(
				ctx,
				http.StatusInternalServerError,
				"%s",
				err,
			)
		}

		results := &queryResults{}
		// Iterate over search results and print the
		// result name and URL.
		for _, result := range ans.WebPages.Value {
			if result.IsFamilyFriendly {
				appendResult(
					results,
					result.Name,
					result.URL,
				)
			}
		}

		// 	ResultList: []string{
		// 		"definatelyevil",
		// 		"Joker_Dan",
		// 		"bobbingbaboon",
		// 		"rockerBOO",
		// 		"isiahvander",
		// 		"erikdotdev",
		// 		"zanuss",
		// 	},
		// }

		return ctx.JSONPretty(http.StatusOK, results, "  ")
	}
}

func main() {
	token := os.Getenv("BING_SEARCH_KEY")
	if token == "" {
		log.Fatal("BING_SEARCH_KEY not found")
	}
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// curl localhost:12334/api/search?term="how+do+you+mine+bitcoin"

	g := e.Group("/api")
	g.GET("/search", newSearchHandler(token))

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}
