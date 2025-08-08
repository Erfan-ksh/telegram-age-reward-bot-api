package main

import (
	"context"
	"flag"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"time"
)

var Router *mux.Router

func main() {
	SetupEnvs()
	SetupBot()
	SetupDB()
	SetupLogFile()

	// code for adding rank tasks do not uncomment
	// for j := 1; j < 101; j++ {
	// 	title := fmt.Sprintf("Reach rank level %d", j)
	// 	calcrew := 100 * math.Pow(1.14, float64(j-1))
	// 	reward := int(math.Ceil(calcrew))

	// 	totaltap := 1600 + 400*j

	// 	_, err := DB.Exec("INSERT INTO nothings_tasks (title, description, task_type, reward, total_taps) VALUES ( ?, '', 'rank', ?, ?);", title, reward, totaltap)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }

	Router = mux.NewRouter()

	pwd, _ := filepath.Abs("./images/")
	Router.HandleFunc("/images/{filename}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		filename := vars["filename"]
		filePath := path.Join(pwd, filename)

		w.Header().Set("Cache-Control", "public, max-age=2592000")

		http.ServeFile(w, r, filePath)
	})

	InitUserRoutes()

	srv := &http.Server{
		Addr:         "127.0.0.1:9000",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      Router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c

	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	srv.Shutdown(ctx)
	log.Println("shutting down")
	os.Exit(0)
}
