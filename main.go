package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime/pprof"
)

func setupRoutes(path string) *Router {
	context := &Context{path}

	router := NewRouter()
	router.HeadFunc("/config", CheckConfig(context))
	router.GetFunc("/config", GetConfig(context))
	router.PostFunc("/config", SaveConfig(context))
	router.GetFunc("/:dir/", ListBlobs(context))
	router.HeadFunc("/:dir/:name", CheckBlob(context))
	router.GetFunc("/:type/:name", GetBlob(context))
	router.PostFunc("/:type/:name", SaveBlob(context))
	router.DeleteFunc("/:type/:name", DeleteBlob(context))

	return router
}

func main() {
	log.SetFlags(0)

	var cpuprofile = flag.String("cpuprofile", "", "write CPU profile to file")
	var listen = flag.String("listen", ":8000", "listen address")
	var path = flag.String("path", "/tmp/restic", "data directory")
	var tls = flag.Bool("tls", false, "turn on TLS support")
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
		}
		log.Println("CPU profiling enabled")
		defer pprof.StopCPUProfile()
	}

	router := setupRoutes(*path)

	var handler http.Handler
	htpasswdFile, err := NewHtpasswdFromFile(filepath.Join(*path, ".htpasswd"))
	if err != nil {
		handler = router
		log.Println("Authentication disabled")
	} else {
		handler = AuthHandler(htpasswdFile, router)
		log.Println("Authentication enabled")
	}

	if !*tls {
		log.Printf("Starting server on %s\n", *listen)
		err = http.ListenAndServe(*listen, handler)
	} else {
		privateKey := filepath.Join(*path, "private_key")
		publicKey := filepath.Join(*path, "public_key")
		log.Println("TLS enabled")
		log.Printf("Private key: %s", privateKey)
		log.Printf("Public key: %s", publicKey)
		log.Printf("Starting server on %s\n", *listen)
		err = http.ListenAndServeTLS(*listen, publicKey, privateKey, handler)
	}
	if err != nil {
		log.Fatal(err)
	}
}
