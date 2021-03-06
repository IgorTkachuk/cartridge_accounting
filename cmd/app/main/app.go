package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/IgorTkachuk/cartridge_accounting/internal/config"
	user2 "github.com/IgorTkachuk/cartridge_accounting/internal/domain/user"
	user "github.com/IgorTkachuk/cartridge_accounting/internal/domain/user/db"
	"github.com/IgorTkachuk/cartridge_accounting/pkg/client/postgresql"
	"github.com/IgorTkachuk/cartridge_accounting/pkg/logging"
	"github.com/IgorTkachuk/cartridge_accounting/pkg/shutdown"
	"github.com/julienschmidt/httprouter"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"
)

func main() {
	logger := logging.GetLogger()
	cfg := config.GetConfig()
	//cfg := postgresql.NewPgConfig("postgres", "mg0208", "localhost", "5432", "ctr")
	cli, _ := postgresql.NewClient(context.Background(), 3, 5*time.Second, cfg.Storage)
	r := user.NewRepository(cli)
	svc := user2.NewService(r)

	userHandler := user2.Handler{
		UserService: svc,
	}

	logger.Info("create router")
	router := httprouter.New()
	logger.Info("register user handler")
	userHandler.Register(router)

	start(router)

	//users, _ := r.FindAll(context.Background())
	//
	//for _, u := range users {
	//	fmt.Println(u)
	//}
}

func start(router http.Handler) {
	logger := logging.GetLogger()
	logger.Info("start application")

	var server *http.Server
	var listener net.Listener
	var listenerErr error

	listener, listenerErr = net.Listen("tcp", fmt.Sprintf("%s:%s", "0.0.0.0", "3001"))
	if listenerErr != nil {
		logger.Fatal(listenerErr)
	}

	server = &http.Server{
		Handler:      router,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go shutdown.Graceful([]os.Signal{syscall.SIGABRT, syscall.SIGQUIT, syscall.SIGHUP, os.Interrupt, syscall.SIGTERM}, server)

	logger.Println("application initialized and started")

	if err := server.Serve(listener); err != nil {
		switch {
		case errors.Is(err, http.ErrServerClosed):
			logger.Warn("server shutdown")
		default:
			logger.Fatal(err)
		}
	}
}
