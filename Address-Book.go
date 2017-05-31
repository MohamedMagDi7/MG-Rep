package main

import (
	"strconv"
	"net/http"
	"html/template"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"github.com/gorilla/sessions"
	gmux "github.com/gorilla/mux"
	"fmt"
	"github.com/codegangsta/negroni"

	"encoding/json"
)
 type Contact struct{
	 Id int
	 FirstName string
	 LastName string
	 Email string
	 PhoneNumber []string

 }
type User_Contancts struct {
	UserName string
	Id string
	Contacts []Contact

}

var db *sql.DB
var err error
var templates *template.Template
var User =User_Contancts{}



func Login(w http.ResponseWriter, r *http.Request){

	if err = db.Ping(); err !=nil {
		fmt.Println("Database is closed!")
		http.Error(w, err.Error(),http.StatusInternalServerError)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")

	if r.FormValue("register")!="" {
		fmt.Println("registered")
		var user string

		err = db.QueryRow("SELECT username FROM users WHERE username=?", username).Scan(&user)

		switch {
		// Username is available
		case err == nil:
			fmt.Println("Username is not available")
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		case err == sql.ErrNoRows:
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				fmt.Println("Couldn't Incrypt")
				http.Error(w,err.Error(),http.StatusInternalServerError)
			}

			_, err = db.Exec("INSERT INTO users(username, password) VALUES(?, ?)", username, hashedPassword)
			if err != nil {
				http.Error(w,err.Error(),http.StatusInternalServerError)
				return
			}
			session , _ := store.Get(r,"CurrentSession")
			session.Values["user"]=username
			session.Save(r,w)
			http.Redirect(w, r, "/userpage", http.StatusFound)
			return
		case err != nil:
			http.Error(w, "Server error, unable to create your account.", 500)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		default:
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		}


	}

	if r.FormValue("login")!="" {
		fmt.Println("logged in")
		// Grab from the database
		var databaseUsername string
		var databasePassword string

		err = db.QueryRow("SELECT username, password FROM users WHERE username=?", username).Scan(&databaseUsername, &databasePassword)
		if err == sql.ErrNoRows {

			fmt.Println("no such user")
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return

		} else if err != nil {

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(databasePassword), []byte(password))
		// If wrong password redirect to the login
		if err != nil {
			fmt.Println("wrong password")
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		} else {
			fmt.Println("password match")
			// If the login succeeded
			session , _ := store.Get(r,"CurrentSession")
			session.Values["user"]=username
			session.Save(r,w)
			http.Redirect(w, r, "/userpage", 301)
			return
		}
	}
}

func Delete(w http.ResponseWriter, r *http.Request){
	_ ,err := db.Exec("delete from contact where contactID = ?",r.FormValue("id"))

	if err !=nil{
		fmt.Println("DB error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println("row Deleted")
}
///////////////////////////////////////////////////////////////////////////////////////////////////////
func Check(w http.ResponseWriter, r *http.Request) {
	session , _ := store.Get(r,"CurrentSession")
	Usr :=session.Values["user"].(string)
	if Usr !="" {
		fmt.Println("lesa")
		http.Redirect(w,r,"/userpage",http.StatusFound)
		return
	}else {
		fmt.Println("logged out")
		http.Redirect(w,r,"/home",http.StatusFound)
		return
		}
	}

///////////////////////////////////////////////////////////////////////////////////////////////////////
func HomePage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("home")
	templates := template.Must(template.ParseFiles("index.html"))
	if err := templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		fmt.Println("error home")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}
///////////////////////////////////////////////////////////////////////////////////////////////////////
func UserPage(w http.ResponseWriter, r *http.Request) {
	templates := template.Must(template.ParseFiles("userpage.html"))
	User.Contacts = []Contact{}
	session , _ := store.Get(r,"CurrentSession")
	UsernameIn :=session.Values["user"]
	Username :=UsernameIn.(string)
	if Username==""{
		http.Redirect(w,r,"/home",http.StatusFound)
		return
	}

	User.UserName =Username
	row := db.QueryRow("select id from users where username= ?",Username)

	row.Scan(&User.Id)
	 rows, err := db.Query("select contactID,fname,lname,email from contact where userID= ?",User.Id)
	if err!=nil{
		fmt.Println("DB error")
		http.Error(w,err.Error(),http.StatusInternalServerError)

	}

	for rows.Next() {
		var c Contact
		rows.Scan(&c.Id, &c.FirstName, &c.LastName , &c.Email )

		res, err := db.Query("select phonenumber from phonenumbers where contact_id= ?",c.Id)
		if err!=nil{
			fmt.Println("DB error")
			http.Error(w,err.Error(),http.StatusInternalServerError)

		}

		for res.Next() {
			var N string
			res.Scan(&N)
			c.PhoneNumber = append(c.PhoneNumber, N)

		}
		User.Contacts = append(User.Contacts, c)

	}


	if err := templates.ExecuteTemplate(w, "userpage.html", User); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

	}

}

func AddContact(w http.ResponseWriter, r *http.Request) {

	_, err := db.Exec("insert into contact values(? ,? ,? ,? ,? ) ", nil, r.FormValue("first-name"),
		r.FormValue("last-name"), r.FormValue("email"), User.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	row := db.QueryRow("select MAX(contactID) from contact")
	var id int
	row.Scan(&id)

	c := Contact{
		FirstName:r.FormValue("first-name"),
		LastName:r.FormValue("last-name"),
		Email:r.FormValue("email"),
		//PhoneNumber:r.FormValue("phone"),
	}
	i := 1
	for r.FormValue("phone" + strconv.Itoa(i)) != "" {
		str := r.FormValue("phone" + strconv.Itoa(i))
		c.PhoneNumber = append(c.PhoneNumber, str)
		_, err := db.Exec("insert into phonenumbers values(?,?,?)", nil, str , id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		i++
	}

	User.Contacts = append(User.Contacts, c)
	if err := json.NewEncoder(w).Encode(c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}



func Logout(w http.ResponseWriter, r *http.Request){
	session , _ := store.Get(r,"CurrentSession")
	session.Values["user"]=""
	session.Save(r,w)

	http.Redirect(w,r,"/home",http.StatusFound)
	return

}

///////////////////////////////////////////////////////////////////////////////////////////////////////
var store = sessions.NewCookieStore([]byte("1819"))

func main() {

	db, err = sql.Open("mysql", "root:1819@tcp(127.0.0.1:3306)/my_add_bookDB")
	if err != nil {
		panic(err)
	}

	mux :=gmux.NewRouter()
	defer db.Close()

	mux.HandleFunc("/", Check)
	mux.HandleFunc("/home", HomePage)
	mux.HandleFunc("/login", Login).Methods("POST")
	mux.HandleFunc("/userpage", UserPage)
	mux.HandleFunc("/addcontact", AddContact)
	mux.HandleFunc("/logout", Logout)
	mux.HandleFunc("/delete", Delete)
	n:= negroni.Classic()
	n.UseHandler(mux)
	n.Run(":9000")
	//mux.ListenAndServe(":8080", nil)
}


