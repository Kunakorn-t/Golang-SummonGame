package main

// ยังเขียนไม่เสร็จดีนะครับ ผมเขียนพอให้มีโปรไฟล์มาสมัครงานได้ก่อนครับผม จะได้ไม่กินเวลานาน ผมอยากทำงานแล้ว

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"text/template"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB
var err error
var account string // เดะทำ logout ให้ล้างค่าในนี้ด้วย

type Monster struct {
	Name  string
	Stack int
	Atk   int
	Def   int
}

func summonPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "summonPage.html")

}

func summon(w http.ResponseWriter, r *http.Request) {
	var claimMonster int
	err = db.QueryRow("select claim from users_table where username=?", account).Scan(&claimMonster)
	fmt.Println(claimMonster)
	if claimMonster < 1 {
		fmt.Println("error check claim.")
		http.ServeFile(w, r, "failSummon.html")
		return
	}

	var owner int
	err = db.QueryRow("select count(owner) from monster_table where owner=?", account).Scan(&owner)
	if err != nil {
		http.Error(w, "error count monster of user", 500)
		return
	}

	if owner > 9 {
		fmt.Println("error because you have monster 10 unit.")
		http.ServeFile(w, r, "failSummon.html")
		return
	}

	monsterName := r.FormValue("monsterName")
	if monsterName == "" {
		http.Error(w, "ต้องระบุชื่อ monster", 500)
		return
	}

	var MonsName string
	err = db.QueryRow("select monster_name from monster_table where monster_name=?", monsterName).Scan(&MonsName)
	// fmt.Println(err)
	if MonsName != "" {
		http.ServeFile(w, r, "failSummon.html")
		return
	}

	var templates = template.Must(template.ParseFiles("monster.html"))
	atk := rand.Intn(50)
	def := rand.Intn(50)
	stack := atk + def
	monster := Monster{monsterName, stack, atk, def}
	templates.ExecuteTemplate(w, "monster.html", monster)

	switch {
	case err == sql.ErrNoRows:
		_, err = db.Exec("insert into monster_table(monster_name, stack, atk, def, owner) values(?,?,?,?,?)", monsterName, stack, atk, def, account)
		if err != nil {
			// fmt.Println(err)
			http.Error(w, "error insert monster", 500)
		}
		break
	case err != nil:
		http.Error(w, "error summon insert monster", 500)
		break
	default:
		http.Redirect(w, r, "/summonpage", 301)
		break
	}
}

func refreshBot() { // เดะทำ template แสดงเวลาที่จะ cooldown เสร็จ

	var resultQueTimerRefrBot string
	var finish int
	now := time.Now()
	timeStamp := now.Unix()
	pesently := int(timeStamp)
	err = db.QueryRow("select timer,finish from timer_table where timer='cooldownRefreshBot'").Scan(&resultQueTimerRefrBot, &finish)

	if err == sql.ErrNoRows {
		fmt.Println(err)
		fmt.Println("in check err no rows")
		go cooldown(err)
		go createBot()
	} else if resultQueTimerRefrBot == "cooldownRefreshBot" {
		if pesently > finish {
			fmt.Println(err)
			fmt.Println("in check cooldown complete")
			go cooldown(err)
			go createBot()
		} else if pesently <= finish {
			fmt.Println("Cooldown in progress!")
		} else {
			panic(err)
		}
	} else if resultQueTimerRefrBot != "cooldownRefreshBot" {
		go cooldown(err)
		go createBot()
	} else {
		fmt.Printf("start: %v and finish: %v", pesently, finish)
		panic(err)
	}
	return
}

func cooldown(err error) {
	now := time.Now()
	timeStamp := now.Unix()
	finish := timeStamp + 30
	// 11400

	if err == sql.ErrNoRows {
		_, err = db.Exec("insert into timer_table(timer,now,finish) values(?,?,?)", "cooldownRefreshBot", timeStamp, finish)
		if err != nil {
			fmt.Println("error insert cooldown")
			panic(err)
		}
	} else {
		_, err = db.Exec("update timer_table set now=?,finish=? where timer='cooldownRefreshBot'", timeStamp, finish)
		if err != nil {
			fmt.Println("error update cooldown")
			panic(err)
		}
	}
	fmt.Println("New Cooldown!!")
	return
}

func createBot() { // เดะทำ random ให้ดีขึ้น ไม่ให้ซ้ำบ่อย

	stmt, err := db.Prepare("delete from monster_table where owner=?")
	stmt.Exec("bot")
	defer stmt.Close()
	if err != nil {
		panic(err)
	}

	for i := 0; i < 20; i++ {
		atk := rand.Intn(50)
		def := rand.Intn(50)
		stack := atk + def
		// เดะมา condition loop ตรงนี้
		randNameBot := rand.Intn(10000)
		name := strconv.Itoa(randNameBot)
		_, err = db.Exec("insert into monster_table(monster_name,stack,atk,def,owner) values(?,?,?,?,?)", name, stack, atk, def, "bot")
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
	}
	fmt.Println("New create bot!")
	return
}

func fieldPage(w http.ResponseWriter, r *http.Request) {

	refresh := r.FormValue("callRefresh")
	if refresh == "callRefresh" {
		go refreshBot()
		refresh = ""

	}

	var botTemplates = template.Must(template.ParseFiles("fieldPage.html"))
	rows, err := db.Query("select monster_name,stack,atk,def from monster_table where owner='bot'")
	defer rows.Close()
	if err != nil {
		fmt.Println(w, "error query monster of bot")
		fmt.Println(err)
		panic(err)
	}
	wMonster := Monster{}
	var arrMonster []Monster
	for rows.Next() {
		var name string
		var stack int
		var atk int
		var def int
		err = rows.Scan(&name, &stack, &atk, &def)
		wMonster.Name = name
		wMonster.Stack = stack
		wMonster.Atk = atk
		wMonster.Def = def
		arrMonster = append(arrMonster, wMonster)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
	}
	botTemplates.Execute(w, arrMonster)
}

func main() {
	db, err = sql.Open("mysql", "root:@/summon_db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}

	r := mux.NewRouter()

	r.HandleFunc("/", homePage)
	r.HandleFunc("/login", loginPage)
	r.HandleFunc("/regis", regisPage)
	r.HandleFunc("/summonpage", summonPage)
	r.HandleFunc("/summon", summon)
	r.HandleFunc("/field", fieldPage)
	// r.HandleFunc("/war", warPage)
	// r.HandleFunc("/profile", profilePage)

	http.ListenAndServe(":8080", r)
}

func homePage(w http.ResponseWriter, r *http.Request) {
	// fmt.Println(account)
	if account == "" {
		http.Redirect(w, r, "/login", 301)
		return
	}

	http.ServeFile(w, r, "homePage.html")
}

func regisPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.ServeFile(w, r, "regisPage.html")
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	var user string
	err = db.QueryRow("select username from users_table where username=?", username).Scan(&user)

	switch {
	case err == sql.ErrNoRows:
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "error ตอน hashed password: create account", 500)
			return
		}
		_, err = db.Exec("insert into users_table(username, password,claim) values(?,?,?)", username, hashedPassword, 1)
		if err != nil {
			http.Error(w, "error ตอน insert: create account", 500)
			return
		}
		http.ServeFile(w, r, "createAccountFinish.html")
		return
	case err != nil:
		http.Error(w, "error create account: case err!=nil มีคนใช้ชื่อนี้แล้ว", 500)
		return
	default:
		http.Redirect(w, r, "/", 301)
	}
}

func loginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.ServeFile(w, r, "loginPage.html")
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	var dbName string
	var dbPass string
	err = db.QueryRow("select username, password from users_table where username=?", username).Scan(&dbName, &dbPass)
	if err != nil {
		http.Redirect(w, r, "/login", 301)
		// fmt.Println(err)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(dbPass), []byte(password))
	if err != nil {
		http.Redirect(w, r, "/login", 301)
		// fmt.Println(err)
		return
	}
	account = dbName
	http.Redirect(w, r, "/", 301)
}

// battle
// profile
// level
// dual user
// etc.
