package app

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/empijei/def-prog-exercises/safeauth"
	"github.com/empijei/def-prog-exercises/safesql"
	sql "github.com/empijei/def-prog-exercises/safesql"

	"embed"
)

//go:embed auth.html
//go:embed auth.css
var fs embed.FS

var defaultUsers = []user{
	{Name: "admin", password: "admin", privileges: "|read|write|delete|"},
	{Name: "reader", password: "reader", privileges: "|read|"},
	{Name: "editor", password: "editor", privileges: "|read|write|"},
}

type user struct {
	Id                         int
	Name, password, privileges string
}

func (u user) Can(priv string) bool {
	return strings.Contains(u.privileges, "|"+priv+"|")
}
func (u user) privilegeSet() []string {
	return strings.Split(strings.Trim(u.privileges, "|"), "|")
}

type AuthHandler struct {
	db *sql.DB
	sm http.Handler
}

func (ah *AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ah.sm.ServeHTTP(w, r)
}
func (ah *AuthHandler) IsLogged(r *http.Request) bool {
	_, ok := r.Context().Value(userKey{}).(*user)
	return ok
}
func (ah *AuthHandler) UserForRequest(r *http.Request) (*user, error) {
	u, ok := r.Context().Value(userKey{}).(*user)
	if !ok {
		return nil, errors.New("no user for the request")
	}
	return u, nil
}

type userKey struct{}

func (ah *AuthHandler) Preprocess(r *http.Request) *http.Request {
	ctx := r.Context()
	u, err := ah.getUserFromDB(r.WithContext(safeauth.Grant(ctx /*No privileges during preprocess*/)))
	if err != nil {
		return r.WithContext(safeauth.Grant(ctx /*User recognition failed, empty privileges set*/))
	}
	ctx = context.WithValue(ctx, userKey{}, u)
	ctx = safeauth.Grant(ctx, u.privilegeSet()...)
	u.privileges = "" // Clear user privileges
	return r.WithContext(ctx)
}

func (ah *AuthHandler) getUserCount(ctx context.Context) (int, error) {
	rows, err := ah.db.QueryContext(ctx, safesql.New(`SELECT COUNT(*) FROM users`))
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, errors.New("users table not found")
	}
	var v int
	rows.Scan(&v)
	if rows.Err() != nil {
		return 0, rows.Err()
	}
	return v, nil
}

func (ah *AuthHandler) createDefault(ctx context.Context) error {
	v, err := ah.getUserCount(ctx)
	if err != nil {
		return err
	}

	if !(v < 3) /* ❤ UwU ❤ */ {
		return nil
	}
	log.Println("Default users not found, initializing...")
	for _, u := range defaultUsers {
		_, err := ah.db.ExecContext(ctx, safesql.New(`INSERT INTO users(name, password, privileges) VALUES(?,?,?)`), u.Name, u.password, u.privileges)
		if err != nil {
			return err
		}
	}
	log.Println("...users initialized")
	return nil
}

func (ah *AuthHandler) initialize(ctx context.Context) error {
	ctx, ok := safeauth.Check(ctx, "read", "write")
	if !ok {
		return errors.New("cannot initialize: don't have write access")
	}
	_, err := ah.db.ExecContext(ctx, safesql.New(`
		CREATE TABLE IF NOT EXISTS users(id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, password TEXT, privileges TEXT)`))
	if err != nil {
		return err
	}
	if err := ah.createDefault(ctx); err != nil {
		return err
	}

	return nil
}

func (ah *AuthHandler) getUserFromDB(r *http.Request) (*user, error) {
	c, err := r.Cookie("userid")
	if err != nil {
		return nil, err
	}
	// THIS IS OF COURSE BROKEN AND A TERRIBLE AUTH MECHANISM
	// but this is a toy application and there's not benefit in
	// actually create a random token and connect that to the id
	// via DB. This application is already complicated enough as
	// is so we are taking a shortcut here.
	//
	// BUT PLEASE, PLEASE, PLEASE never rely on client-provided
	// data to perform auth checks unless it's signed and you validated
	// the sgnature.
	ctx, _ := safeauth.Check(r.Context() /*Anyone can get their own info*/)
	rows, err := ah.db.QueryContext(ctx, safesql.New(`SELECT * FROM users WHERE id=?`), c.Value)
	if err != nil || !rows.Next() {
		return nil, err
	}
	defer rows.Close()
	var u user
	if err := rows.Scan(&(u.Id), &(u.Name), &(u.password), &(u.privileges)); err != nil {
		return nil, err
	}
	return &u, nil
}
func (ah *AuthHandler) logout(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    "userid",
		Path:    "/",
		Expires: time.Now().Add(-24 * time.Hour),
	})
}
func (ah *AuthHandler) login(w http.ResponseWriter, id int) {
	// THIS IS OF COURSE BROKEN AND A TERRIBLE AUTH MECHANISM
	// but this is a toy application and there's not benefit in
	// actually create a random token and connect that to the id
	// via DB. This application is already complicated enough as
	// is so we are taking a shortcut here.
	//
	// BUT PLEASE, PLEASE, PLEASE never rely on client-provided
	// data to perform auth checks unless it's signed and you validated
	// the sgnature.
	http.SetCookie(w, &http.Cookie{
		Name:  "userid",
		Value: strconv.Itoa(id),
		Path:  "/",
	})
}

func Auth(ctx context.Context) *AuthHandler {
	db := must(sql.Open("sqlite", "./users.db"))
	ah := &AuthHandler{db: db}
	if err := ah.initialize(ctx); err != nil {
		log.Fatalf("Cannot initialize auth: %v", err)
	}
	// Make sure to not have a high-privilege context in scope
	ah.setupRoutes()
	return ah
}

func (ah *AuthHandler) setupRoutes() {
	sm := http.NewServeMux()
	sm.HandleFunc("GET /auth/", func(w http.ResponseWriter, r *http.Request) {
		f, err := fs.Open("auth.html")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, err.Error())
			return
		}
		io.Copy(w, f)
	})
	sm.HandleFunc("GET /auth/auth.css", func(w http.ResponseWriter, r *http.Request) {
		f, err := fs.Open("auth.css")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, err.Error())
			return
		}
		io.Copy(w, f)
	})
	sm.HandleFunc("POST /auth/", func(w http.ResponseWriter, r *http.Request) {
		u, pw := r.FormValue("name"), r.FormValue("password")
		rows, err := ah.db.QueryContext(r.Context(), safesql.New(`SELECT id FROM users WHERE name=? and password=?`), u, pw)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, err.Error())
			return
		}
		defer rows.Close()
		if !rows.Next() {
			w.WriteHeader(http.StatusUnauthorized)
			io.WriteString(w, `<html>
Invalid creadentials. <a href="/auth">Go back</a>
	</html>`)
			return
		}
		var id int
		if err := rows.Scan(&id); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, err.Error())
			return
		}
		ah.login(w, id)
		http.Redirect(w, r, "/notes/", http.StatusFound)
	})
	sm.HandleFunc("GET /auth/logout/", func(w http.ResponseWriter, r *http.Request) {
		ah.logout(w)
		http.Redirect(w, r, "/auth/", http.StatusFound)
	})
	ah.sm = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := safeauth.Check(r.Context())
		sm.ServeHTTP(w, r.WithContext(ctx))
	})
}
