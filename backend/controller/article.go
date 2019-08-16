package controller

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"golang.org/x/net/html"

	"github.com/voyagegroup/treasure-app/httputil"
	"github.com/voyagegroup/treasure-app/model"
	"github.com/voyagegroup/treasure-app/repository"
	"github.com/voyagegroup/treasure-app/service"
)

type Paper struct {
	Title    string
	Abstract string
}
type Article struct {
	dbx *sqlx.DB
}

type ArticleComment struct {
	dbx *sqlx.DB
}

type ArticleTag struct {
	dbx *sqlx.DB
}

func NewArticle(dbx *sqlx.DB) *Article {
	return &Article{dbx: dbx}
}

func NewArticleComment(dbx *sqlx.DB) *ArticleComment {
	return &ArticleComment{dbx: dbx}
}

func NewArticleTag(dbx *sqlx.DB) *ArticleTag {
	return &ArticleTag{dbx: dbx}
}

// 返り値のintは status code
func (a *Article) Index(w http.ResponseWriter, r *http.Request) (int, interface{}, error) {
	articles, err := repository.AllArticle(a.dbx)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	return http.StatusOK, articles, nil
}

func (a *Article) SearchIndex(w http.ResponseWriter, r *http.Request) (int, interface{}, error) {
	vars := mux.Vars(r)
	tag, ok := vars["tag"]
	if !ok {
		return http.StatusBadRequest, nil, &httputil.HTTPError{Message: "invalid path parameter"}

	}

	articles, err := repository.SearchArticle(a.dbx, tag)
	if err != nil && err == sql.ErrNoRows {
		return http.StatusNotFound, nil, err
	} else if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return http.StatusOK, articles, nil
}

func (a *Article) Show(w http.ResponseWriter, r *http.Request) (int, interface{}, error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return http.StatusBadRequest, nil, &httputil.HTTPError{Message: "invalid path parameter"}
	}
	fmt.Println("show")
	aid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	article, err := repository.FindArticle(a.dbx, aid)
	if err != nil && err == sql.ErrNoRows {
		return http.StatusNotFound, nil, err
	} else if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return http.StatusCreated, article, nil
}

func (a *Article) Create(w http.ResponseWriter, r *http.Request) (int, interface{}, error) {
	newArticle := &model.Article{}
	fmt.Println("tes")

	if err := json.NewDecoder(r.Body).Decode(&newArticle); err != nil {
		return http.StatusBadRequest, nil, err
	}

	user, err := httputil.GetUserFromContext(r.Context())
	if err != nil {
		fmt.Println(err)
	}
	newArticle.UserID = &user.ID

	articleService := service.NewArticleService(a.dbx)
	fmt.Println(newArticle)
	id, err := articleService.Create(newArticle)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	newArticle.ID = id

	return http.StatusCreated, newArticle, nil
}

func isDescription(attrs []html.Attribute) bool {
	for _, attr := range attrs {
		if attr.Key == "title" && attr.Val == "summary" {
			return true
		}
	}
	return false
}

func (a *Article) CreatePaper(w http.ResponseWriter, r *http.Request) (int, interface{}, error) {
	// vars := mux.Vars(r)
	vars := r.URL.Query()
	keyword := vars["keyword"][0]

	resp, err := http.Get("http://export.arxiv.org/api/query?search_query=all:" + keyword + "&start=0&max_results=100")

	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	// 修正して :pray:
	var count int = 0

	var paper Paper
	var f func(*html.Node)

	f = func(n *html.Node) {

		// fmt.Println(*n)
		if n.Type == html.ElementNode && n.Data == "title" {
			paper.Title = n.FirstChild.Data
		}
		if n.Type == html.ElementNode && n.Data == "summary" {
			paper.Abstract = n.FirstChild.Data
		}
		if n.Type == html.ElementNode && n.Data == "entry" {
			fmt.Println("ok", count)
			count++
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	fmt.Println("Search Start")
	fmt.Printf("%#v", paper)

	tag := "tes"

	articles, err := repository.SearchArticle(a.dbx, tag)
	if err != nil && err == sql.ErrNoRows {
		return http.StatusNotFound, nil, err
	} else if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return http.StatusOK, articles, nil
}

func (a *Article) Update(w http.ResponseWriter, r *http.Request) (int, interface{}, error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return http.StatusBadRequest, nil, &httputil.HTTPError{Message: "invalid path parameter"}
	}
	fmt.Println("update")

	aid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	reqArticle := &model.Article{}
	if err := json.NewDecoder(r.Body).Decode(&reqArticle); err != nil {
		return http.StatusBadRequest, nil, err
	}

	articleService := service.NewArticleService(a.dbx)
	err = articleService.Update(aid, reqArticle)
	if err != nil && errors.Cause(err) == sql.ErrNoRows {
		return http.StatusNotFound, nil, err
	} else if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return http.StatusNoContent, nil, nil
}

func (a *Article) Destroy(w http.ResponseWriter, r *http.Request) (int, interface{}, error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return http.StatusBadRequest, nil, &httputil.HTTPError{Message: "invalid path parameter"}
	}
	fmt.Println("destroy")

	aid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	articleService := service.NewArticleService(a.dbx)
	err = articleService.Destroy(aid)
	if err != nil && errors.Cause(err) == sql.ErrNoRows {
		return http.StatusNotFound, nil, err
	} else if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return http.StatusNoContent, nil, nil
}

func (a *Article) DestroyAll(w http.ResponseWriter, r *http.Request) (int, interface{}, error) {

	articleService := service.NewArticleService(a.dbx)
	err := articleService.DestroyAll()
	if err != nil && errors.Cause(err) == sql.ErrNoRows {
		return http.StatusNotFound, nil, err
	} else if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return http.StatusNoContent, nil, nil
}

func (a *ArticleComment) CreateArticleComment(w http.ResponseWriter, r *http.Request) (int, interface{}, error) {
	newArticleComment := &model.ArticleComment{}
	if err := json.NewDecoder(r.Body).Decode(&newArticleComment); err != nil {
		return http.StatusBadRequest, nil, err
	}

	user, err := httputil.GetUserFromContext(r.Context())
	if err != nil {
		fmt.Println(err)
	}
	newArticleComment.UserID = &user.ID

	articleCommentService := service.NewArticleCommentService(a.dbx)
	id, err := articleCommentService.CreateArticleComment(newArticleComment)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	newArticleComment.ID = id

	return http.StatusCreated, newArticleComment, nil
}

func (a *ArticleTag) CreateArticleTag(w http.ResponseWriter, r *http.Request) (int, interface{}, error) {
	newArticleTag := &model.ArticleTag{}
	if err := json.NewDecoder(r.Body).Decode(&newArticleTag); err != nil {
		return http.StatusBadRequest, nil, err
	}

	articleTagService := service.NewArticleTagService(a.dbx)
	id, err := articleTagService.CreateArticleTag(newArticleTag)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	newArticleTag.ID = id

	return http.StatusCreated, newArticleTag, nil
}
