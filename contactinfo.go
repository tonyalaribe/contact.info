package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

var HAddress string = "http://localhost:64210"

type quad struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
	Label     string `json:"label"`
}

type triads []quad

func Write(add string, q triads) {
	address := add + "/api/v1/write"
	t, err := json.Marshal(q)
	triad := string(t)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println(string(triad))
	var x bytes.Buffer
	x.Write([]byte(triad))
	resp, err := http.Post(address, "text/json", &x)
	if err != nil {
		fmt.Println(err)
	}
	a, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(a))
	resp.Body.Close()

}

func Delete(add string, q triads) {
	address := add + "/api/v1/delete"
	t, err := json.Marshal(q)
	triad := string(t)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println(triad)
	var x bytes.Buffer
	x.Write([]byte(triad))
	resp, err := http.Post(address, "text/json", &x)
	if err != nil {
		fmt.Println(err)
	}
	a, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(a))
	resp.Body.Close()

}

func Gremlin(add string, q string) []byte {
	address := add + "/api/v1/query/gremlin"
	var x bytes.Buffer
	x.Write([]byte(q))
	resp, err := http.Post(address, "text/plain", &x)
	if err != nil {
		fmt.Println(err)
	}
	a, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(a))
	resp.Body.Close()
	return a

}

func NewAccount(add string, email string, password string) error {
	query := "g.V(\"" + email + "\").Out(\"HasPasswordHash\").All()"
	a := Gremlin(add, query)
	fmt.Printf("New Account Reply %s", []byte(a))
	u := []byte(a)
	var b interface{}
	json.Unmarshal(u, &b)
	m := b.(map[string]interface{})
	if m["result"] == nil {
		pHash := sha1.New()
		io.WriteString(pHash, password)
		q := quad{
			Subject:   email,
			Predicate: "HasPasswordHash",
			Object:    base64.URLEncoding.EncodeToString(pHash.Sum(nil)),
		}
		r := [...]quad{q}
		Write(add, r[:])
		return nil
	}
	//n := m["result"]
	return errors.New("An account with this email exists already")

}

func Auth(add string, email string, password string) error {
	fmt.Println("Password: ", password)
	query := "g.V(\"" + email + "\").Out(\"HasPasswordHash\").All()"
	a := Gremlin(add, query)
	u := []byte(a)
	var b interface{}
	json.Unmarshal(u, &b)
	m := b.(map[string]interface{})
	if m["result"] != nil {
		n := m["result"].([]interface{})
		o := n[0].(map[string]interface{})
		p := o["id"].(string)
		storedPass := base64.URLEncoding.EncodeToString([]byte(p))
		fmt.Println(storedPass)
		pHash := sha1.New()
		io.WriteString(pHash, password)
		pass := base64.URLEncoding.EncodeToString(pHash.Sum(nil))
		fmt.Println(pass)
		if storedPass == pass {
			fmt.Println(storedPass + " vs " + pass)
			return nil
		} else {
			fmt.Println(storedPass + " vs " + pass)
			return errors.New("Incorrect Password")
		}
	}
	return errors.New("Account Doesn't Exist")
	//n := m["result"]

}

func LoadView(add string, uid string) map[string]interface{} {
	preds := [...]string{"is", "follows"}
	items := StringList(preds[:])
	query := fmt.Sprintf(`
        c = graph.V("%s").Out(%v, "pred").TagArray();
        var k = {}
        k["id"] = "bob";
        for (var i = 0; i<c.length; i++){
          j = c[i]["pred"];
          var f = c[i]["id"];
          k[j] = f;
        }
        g.Emit(k)
        `, uid, items)
	a := Gremlin(add, query)
	u := []byte(a)
	var b interface{}
	json.Unmarshal(u, &b)
	m := b.(map[string]interface{})
	n := m["result"].([]interface{})
	o := n[0].(map[string]interface{})
	return o
}
func LoadContactsView(add string, uid string) []interface{} {
	preds := [...]string{"is", "follows"}
	items := StringList(preds[:])
	query := fmt.Sprintf(`graph.V("%s").Out("follows").ForEach(function(d){
            c = g.V(d.id).Out(%v, "pred").TagArray();
              var s=[];
                var k = {};
                  k["id"] = d.id;
                    for (var i=0; i<c.length; i++) {

                          var f = c[i]["id"];
                              j = c[i]["pred"];

                                  k[j] = f;

                                    }
                                      g.Emit(k)

                                    })`, uid, items)
	fmt.Print(query)
	a := Gremlin(add, query)
	u := []byte(a)
	var b interface{}
	json.Unmarshal(u, &b)
	m := b.(map[string]interface{})
	n := m["result"].([]interface{})
	return n
}
func StringList(preds []string) string {
	var items string
	items = "["
	for i, _ := range preds {
		items = items + `"` + preds[i] + `",`
	}
	items = items + "]"

	return items
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	fmt.Println("email :", r.Form["email"])
	fmt.Println("password :", r.Form["password"])
	err := Auth(HAddress, strings.Join(r.Form["email"], ""), strings.Join(r.Form["password"], ""))
	if err != nil {
		fmt.Println("invalid username and password", err)
	}
}

func RootHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("tmpl/signup.html")
	t.Execute(w, nil)
}

func SignupHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	fmt.Println("Name: ", r.Form["sname"])
	fmt.Println("Email: ", r.Form["semail"])
	fmt.Println("Password: ", r.Form["spassword"])

	err := NewAccount(HAddress, strings.Join(r.Form["semail"], ""), strings.Join(r.Form["spassword"], ""))
	if err != nil {
		fmt.Println(err)
	}
	q := quad{
		Subject:   strings.Join(r.Form["semail"], ""),
		Predicate: "http://schema.org/name",
		Object:    strings.Join(r.Form["sname"], ""),
	}
	k := [...]quad{q}
	Write(HAddress, k[:])

}

func server() {
	r := mux.NewRouter()
	r.HandleFunc("/", RootHandler).Methods("GET")
	r.HandleFunc("/signin", LoginHandler).Methods("POST")
	r.HandleFunc("/signup", SignupHandler).Methods("POST")
	http.Handle("/", r)
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func main() {
	//	inf := new(cayley)
	//	inf.address = "http://localhost:64210"
	//inf.address = "http://localhost:8080"
	/*	triad1 := quad{
			Subject:   "ITEM2",
			Predicate: "HasPasswordHash",
			Object:    "DFDSFDSSDFDSF",
			Label:     "labeled",
		}
		triad2 := quad{
			Subject:   "ITEM3",
			Predicate: "HasPasswordHash",
			Object:    "DFDSFDSSDFDSF",
			Label:     "labeled",
		}

		xxx := [...]quad{triad1, triad2}
		fmt.Println(xxx)
		a := LoadView(inf, "bob")
		fmt.Print(a["id"])
	*/
	server()
}
