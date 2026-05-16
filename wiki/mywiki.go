package wiki

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type Page struct {
	Title string
	Body  []byte
}

type SearchData struct {
	Query   string
	Results []*Page
}

var db *sql.DB
var templates *template.Template

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

func InitDB() error {
	user := getEnv("MYSQL_USER", "root")
	password := getEnv("MYSQL_PASSWORD", "")
	host := getEnv("MYSQL_HOST", "localhost")
	port := getEnv("MYSQL_PORT", "3306")
	dbName := getEnv("MYSQL_DB", "wiki")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, password, host, port, dbName)

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS Pages (
		PageID INT AUTO_INCREMENT PRIMARY KEY,
		Title  VARCHAR(255) NOT NULL UNIQUE,
		Body   TEXT
	)`)
	if err != nil {
		return fmt.Errorf("failed to create Pages table: %w", err)
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func Init() {
	templates = template.Must(template.ParseFiles("edit.html", "view.html", "search.html"))

	if err := InitDB(); err != nil {
		log.Fatalf("Database initialization failed: %v\nSet MYSQL_USER, MYSQL_PASSWORD, MYSQL_HOST, MYSQL_PORT, MYSQL_DB env vars as needed.", err)
	}

	// Home page resets on restart (static content)
	p1 := &Page{
		Title: "Home",
		Body:  []byte("This is the home page.<br>Create new pages by visiting /edit/ followed by the page name.<br>You can edit any page by clicking edit!<br>Use the search bar to find pages by title or content.<br><i>This page resets when the server restarts.</i>"),
	}
	if err := p1.save(); err != nil {
		log.Printf("Warning: could not save home page: %v", err)
	}

	p2, err := loadPage("Home")
	if err != nil {
		log.Printf("Warning: could not load home page: %v", err)
	} else {
		fmt.Println(string(p2.Body))
	}
}

// Handler functions

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.Redirect(w, r, "/view/Home", http.StatusFound)
			return
		}
		fn(w, r, m[2])
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	if err := p.save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.FormValue("q"))

	data := &SearchData{Query: query}

	if query != "" {
		results, err := searchPages(query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data.Results = results
	}

	renderSearchTemplate(w, data)
}

// Database helpers

func (p *Page) save() error {
	_, err := db.Exec(
		`INSERT INTO Pages (Title, Body) VALUES (?, ?) ON DUPLICATE KEY UPDATE Body = VALUES(Body)`,
		p.Title, string(p.Body),
	)
	return err
}

func loadPage(title string) (*Page, error) {
	var body string
	err := db.QueryRow("SELECT Body FROM Pages WHERE Title = ?", title).Scan(&body)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("page not found")
		}
		return nil, err
	}
	return &Page{Title: title, Body: []byte(body)}, nil
}

// searchPages uses parameterized queries to prevent SQL injection.
func searchPages(query string) ([]*Page, error) {
	pattern := "%" + query + "%"
	rows, err := db.Query(
		"SELECT Title, Body FROM Pages WHERE Title LIKE ? OR Body LIKE ?",
		pattern, pattern,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []*Page
	for rows.Next() {
		var title, body string
		if err := rows.Scan(&title, &body); err != nil {
			return nil, err
		}
		pages = append(pages, &Page{Title: title, Body: []byte(body)})
	}
	return pages, rows.Err()
}

// Template helpers

func (p *Page) HTMLBody() template.HTML {
	return template.HTML(p.Body)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func renderSearchTemplate(w http.ResponseWriter, data *SearchData) {
	err := templates.ExecuteTemplate(w, "search.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getTitle(w http.ResponseWriter, r *http.Request) (string, error) {
	m := validPath.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return "", errors.New("invalid Page Title")
	}
	return m[2], nil
}

func Main() {
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	http.HandleFunc("/search/", searchHandler)
	http.HandleFunc("/", makeHandler(func(w http.ResponseWriter, r *http.Request, title string) {
		http.Redirect(w, r, "/view/Home", http.StatusFound)
	}))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
