package main

import (
	"bytes"
	"context"
	"log"

	"encoding/json"
	"io"
	// "strconv"
	"net/http"
	// "net/url"
	"os"
	"os/signal"
	"syscall"

	// "github.com/go-mysql-org/go-mysql/client"
	// "github.com/go-mysql-org/go-mysql/mysql"
	// "github.com/mihai-craita/syncthia/serializer"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SolrDocument struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Set up a channel to listen for interrupt signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    // Start a goroutine to handle the interrupt signal
    go func() {
        <-sigChan
        log.Println("Received interrupt signal. Shutting down...")
        cancel()
    }()

    log.Println("Start connecting")
    client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://mongo1:27017,mongo2:27018,mongo3:27019/?replicaSet=rs0"))
    if err != nil {
        log.Fatal("Eroare conexiune", err)
    }
    defer client.Disconnect(ctx)
    log.Println("Connected")

    database := client.Database("testdb")
    collection := database.Collection("users")

    // Set up pipeline to filter operations
    pipeline := mongo.Pipeline{
        {{Key: "$match", Value: bson.M{
            "operationType": bson.M{"$in": bson.A{"insert", "update", "delete"}},
        }}},
    }

    options := options.ChangeStream().SetFullDocument(options.UpdateLookup)

     // Create a change stream
    stream, err := collection.Watch(ctx, pipeline, options)

    if err != nil {
        log.Fatal("Error oplog collection watch ", err)
    }
    defer stream.Close(ctx)

    log.Println("Watching for changes...")

    // Continuously read from the change stream
    for {
        if ctx.Err() != nil {
            log.Println("Context cancelled, stopping...")
            break
        }
        ok := stream.Next(ctx)
        if !ok {
            if err := stream.Err(); err != nil {
                log.Println("Error in change stream:", err)
                if err == mongo.ErrNilCursor {
                    log.Println("Cursor not found, recreating stream...")
                    stream, err = collection.Watch(ctx, pipeline, options)
                    if err != nil {
                        log.Fatal("Error recreating change stream: ", err)
                    }
                    continue
                }
            }
            log.Println("Stream closed, recreating...")
            stream, err = collection.Watch(ctx, pipeline, options)
            if err != nil {
                log.Fatal("Error recreating change stream: ", err)
            }
            continue
        }

        var changeEvent struct {
            FullDocument bson.M `bson:"fullDocument"`
        }
        if err := stream.Decode(&changeEvent); err != nil {
            log.Println("Error decoding change event:", err)
            continue
        }
        fullDoc, err := json.Marshal(changeEvent.FullDocument)
        if err != nil {
            log.Println("error:", err)
        }

        log.Printf("Received full document: -%v-\n", string(fullDoc))
        addDocumentToSolr(fullDoc)

    }

    log.Println("Program stopped")


    // log.Println("Starting the connections.")
    // // Connect to the MySQL database
    // conn, err := client.Connect("127.0.0.1:3306", "user", "password", "testdb")
    // if err != nil {
    //     log.Fatal(err)
    // }
    // // Get data
    // r, err := conn.Execute(`select id, name, email from users`)
    //
    // // Close result for reuse memory (it's not necessary but very useful)
    // defer r.Close()
    // log.Println("Connected to the sender.")
    //
    // // Parse data for receiver
    // var docs []byte
    // docs = append(docs, '{')
    // // Direct access to fields
    // for rowIndex, row := range r.Values {
    //     docs = append(docs, []byte("\"add\": { \"doc\": {")...)
    //     for colIndex, val := range row {
    //         docs = append(docs, '"')
    //         docs = append(docs, r.Fields[colIndex].Name...)
    //         docs = append(docs, '"')
    //         docs = append(docs, ':')
    //         docs = append(docs, ' ')
    //
    //         if val.Type == mysql.FieldValueTypeFloat {
    //             log.Println("Value float:", val.AsFloat64())
    //         }
    //         if val.Type == mysql.FieldValueTypeString {
    //             log.Println("Value:", colIndex, (val.AsString()))
    //             docs = append(docs, '"')
    //             docs = append(docs, val.AsString()...)
    //             docs = append(docs, '"')
    //         }
    //         if val.Type == mysql.FieldValueTypeSigned {
    //             v := []byte(strconv.FormatInt(val.AsInt64(), 10));
    //             docs = append(docs, v...)
    //         }
    //         if colIndex != len(row)-1 {
    //             docs = append(docs, ',')
    //             docs = append(docs, ' ')
    //         }
    //     }   
    //     docs = append(docs, '}')
    //     docs = append(docs, '}')
    //     if rowIndex != len(r.Values)-1 {
    //         docs = append(docs, ',')
    //     }
    // }
    // docs = append(docs, '}')
    // defer conn.Close()
    // log.Println(string(docs));
    //
    // // docJson := `{"id":"1", "name":"John Doe", "email":"john.doe@example.com"}`
    // // jsonPayload := log.Sprintf(`{"add": {"doc": %s}}`, docJson)
    //
    // if err != nil {
    //     panic(err)  // Handle errors more gracefully in production code
    // }
    //
    // // send data to receiver
    // addDocumentToSolr([]byte(docs))
    //
    // // check data was received
    // client := &http.Client{}
    //
    // solrUrl := "http://localhost:8983/solr/mycore/select"
    // query := url.Values{}
    // query.Add("q", "*:*")
    // query.Add("wt", "json")
    //
    // fullUrl := log.Sprintf("%s?%s", solrUrl, query.Encode())
    //
    // response, err := client.Get(fullUrl)
    // if err != nil {
    //     panic(err)
    // }
    // defer response.Body.Close()
    //
    // body, err := io.ReadAll(response.Body)
    // if err != nil {
    //     panic(err)
    // }
    //
    // log.Println(string(body))
}

func addDocumentToSolr(jsonData []byte) {
    solrUrl := "http://localhost:8983/solr/mycore/update?commit=true"

    var output bytes.Buffer
    output.Grow(len(jsonData) + 2)

    // Set the first byte to "["
    output.WriteByte('[')

    for _, b := range jsonData {
        if b != '_' {
            output.WriteByte(b)
        }
    }
    output.WriteByte(']')
    
    req, err := http.NewRequest("POST", solrUrl, &output)
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

    log.Println("Response Body:", string(body))
}

func processEvent(event bson.M) {
    log.Printf("Received event: %+v\n", event)
    // Here you would implement the logic to update Solr based on the MongoDB change
}
