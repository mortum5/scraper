package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gocolly/colly"
)

const (
	baseURL = "https://leftasexercise.com/"
	port    = ":8080"
)

type Article struct {
	Title string
	Link  string
	Line  string
}

// Helper function to truncate text to a specified word count
func truncateToWords(text string, wordLimit int) string {
	words := strings.Fields(text)
	if len(words) > wordLimit {
		return strings.Join(words[:wordLimit], " ") + "..."
	}
	return text
}

// scrapePage scrapes articles from the given URL
func scrapePage(url string) ([]Article, bool, error) {
	c := colly.NewCollector()

	var (
		articles    []Article
		hasNextPage bool
	)

	c.OnHTML("article", func(e *colly.HTMLElement) {
		title := e.ChildText("h2 a")
		link := e.ChildAttr("h2 a", "href")
		firstLine := truncateToWords(strings.TrimSpace(e.ChildText("p")), 90)

		articles = append(articles, Article{
			Title: title,
			Link:  link,
			Line:  firstLine,
		})
	})

	c.OnHTML(".nav-links", func(e *colly.HTMLElement) {
		nextPage := e.ChildAttr("a.next.page-numbers", "href")
		if nextPage != "" {
			hasNextPage = true
		}
	})

	err := c.Visit(url)
	if err != nil {
		return nil, hasNextPage, err
	}

	return articles, hasNextPage, nil
}

// scrapeCategories retrieves all categories from the website
func scrapeCategories() ([]string, error) {
	c := colly.NewCollector()

	var categories []string

	c.OnHTML("nav[aria-label='Categories'] ul li a", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		category := strings.TrimPrefix(href, baseURL+"category/")
		category = strings.TrimSuffix(category, "/")
		categories = append(categories, category)
	})

	err := c.Visit(baseURL)
	if err != nil {
		return nil, err
	}

	return categories, nil
}

func main() {
	r := gin.Default()

	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")

	r.GET("/", func(c *gin.Context) {
		categories, err := scrapeCategories()
		if err != nil {
			c.HTML(http.StatusInternalServerError, "index.html", gin.H{"Error": "Failed to load categories"})
			return
		}
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Categories": categories,
		})
	})

	r.GET("/articles", func(c *gin.Context) {
		pageStr := c.DefaultQuery("page", "1")
		page, _ := strconv.Atoi(pageStr)
		nextPage := strconv.Itoa(page + 1)
		prevPage := strconv.Itoa(page - 1)
		category := c.DefaultQuery("category", "")

		var url string
		if category != "" {
			url = fmt.Sprintf(baseURL+"category/%s/page/%d/", category, page)
		} else {
			url = fmt.Sprintf(baseURL+"page/%d/", page)
		}

		articles, hasNextPage, err := scrapePage(url)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scrape"})
			return
		}

		if !hasNextPage {
			nextPage = ""
		}

		if page-1 < 1 {
			prevPage = ""
		}

		c.HTML(http.StatusOK, "articles.html", gin.H{
			"Articles": articles,
			"PrevPage": prevPage,
			"NextPage": nextPage,
			"Category": category,
		})
	})

	r.Run(port)
}
