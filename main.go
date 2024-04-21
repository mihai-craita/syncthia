package main

import (
    "bytes"
	"fmt"
	"log"
    // "encoding/json"
    "io"
    "strconv"
    "net/http"
    "net/url"

	"github.com/go-mysql-org/go-mysql/client"
	"github.com/go-mysql-org/go-mysql/mysql"
)

type SolrDocument struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    fmt.Println("Starting the connections.")
    // Connect to the MySQL database
    conn, err := client.Connect("127.0.0.1:3306", "user", "password", "testdb")
    if err != nil {
        log.Fatal(err)
    }
    // Get data
    r, err := conn.Execute(`select id, name, email from users`)

    // Close result for reuse memory (it's not necessary but very useful)
    defer r.Close()
    fmt.Println("Connected to the sender.")

    // Parse data for receiver
    var docs []byte
    docs = append(docs, '{')
    // Direct access to fields
    for rowIndex, row := range r.Values {
        docs = append(docs, []byte("\"add\": { \"doc\": {")...)
        for colIndex, val := range row {
            docs = append(docs, '"')
            docs = append(docs, r.Fields[colIndex].Name...)
            docs = append(docs, '"')
            docs = append(docs, ':')
            docs = append(docs, ' ')
            
            if val.Type == mysql.FieldValueTypeFloat {
                fmt.Println("Value float:", val.AsFloat64())
            }
            if val.Type == mysql.FieldValueTypeString {
                fmt.Println("Value:", colIndex, (val.AsString()))
                docs = append(docs, '"')
                docs = append(docs, val.AsString()...)
                docs = append(docs, '"')
            }
            if val.Type == mysql.FieldValueTypeSigned {
                v := []byte(strconv.FormatInt(val.AsInt64(), 10));
                docs = append(docs, v...)
            }
            if colIndex != len(row)-1 {
                docs = append(docs, ',')
                docs = append(docs, ' ')
            }
        }   
        docs = append(docs, '}')
        docs = append(docs, '}')
        if rowIndex != len(r.Values)-1 {
            docs = append(docs, ',')
        }
    }
    docs = append(docs, '}')
    defer conn.Close()
    fmt.Println(string(docs));

    // docJson := `{"id":"1", "name":"John Doe", "email":"john.doe@example.com"}`
    // jsonPayload := fmt.Sprintf(`{"add": {"doc": %s}}`, docJson)

    if err != nil {
        panic(err)  // Handle errors more gracefully in production code
    }

    // send data to receiver
    addDocumentToSolr([]byte(docs))

    // check data was received
    client := &http.Client{}

    solrUrl := "http://localhost:8983/solr/mycore/select"
    query := url.Values{}
    query.Add("q", "*:*")
    query.Add("wt", "json")

    fullUrl := fmt.Sprintf("%s?%s", solrUrl, query.Encode())

    response, err := client.Get(fullUrl)
    if err != nil {
        panic(err)
    }
    defer response.Body.Close()

    body, err := io.ReadAll(response.Body)
    if err != nil {
        panic(err)
    }

    fmt.Println(string(body))
}

func addDocumentToSolr(jsonData []byte) {
    solrUrl := "http://localhost:8983/solr/mycore/update?commit=true"
    req, err := http.NewRequest("POST", solrUrl, bytes.NewBuffer(jsonData))
    if err != nil {
        panic(err)  // Handle errors more gracefully in production code
    }

    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)  // Handle errors more gracefully in production code
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        panic(err)  // Handle errors more gracefully in production code
    }

    fmt.Println("Response Body:", string(body))
}
