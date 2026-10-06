// Запуск веб-сервера визуализатора: go run ./cmd/web
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/EvgeniiIvanov/algo/visual/server"
)

func main() {
	addr := flag.String("addr", ":8080", "адрес HTTP-сервера")
	flag.Parse()

	log.Printf("Визуализатор запущен на http://localhost%s", *addr)
	log.Printf("Демо: curl 'http://localhost%s/api/run/binary-search?values=1,3,5,7,9&target=7'", *addr)

	log.Fatal(http.ListenAndServe(*addr, server.NewMux()))
}
